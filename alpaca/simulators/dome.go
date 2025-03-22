package simulators

import (
	"alpaca/alpaca"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	domeUID       = "621ca2e0-399a-43f6-b9e7-e6575d953507"
	deviceName    = "Dome Simulator"
	deviceType    = "Dome"
	driverName    = "ZRO Dome Driver"
	driverVersion = "1.0"
)

// DomeSimulator implements the alpaca.Dome interface
type DomeSimulator struct {
	logger log.FieldLogger
	tmpl   *template.Template
	store  *store
	config DomeConfig

	info         alpaca.DeviceInfo
	driver       alpaca.DriverInfo
	capabilities alpaca.DomeCapabilities
	status       alpaca.DomeStatus

	connected  bool
	connecting bool
}

func NewDomeSimulator(number int, db *bolt.DB, tmpl *template.Template, logger log.FieldLogger) *DomeSimulator {
	store, err := NewStore(db)
	if err != nil {
		logger.Fatalf("Error creating store: %v", err)
	}

	config, err := store.GetDomeConfig()
	if err != nil {
		logger.Fatalf("Error getting dome config: %v", err)
	}

	return &DomeSimulator{
		logger: logger,
		tmpl:   tmpl,
		store:  store,
		config: config,

		info: alpaca.DeviceInfo{
			Name:     deviceName,
			Type:     deviceType,
			Number:   number,
			UniqueID: domeUID,
		},
		driver: alpaca.DriverInfo{
			Name:             driverName,
			Version:          driverVersion,
			InterfaceVersion: 1,
		},
		capabilities: alpaca.DomeCapabilities{
			CanFindHome:    true,
			CanPark:        true,
			CanSetAltitude: false,
			CanSetAzimuth:  true,
			CanSetPark:     true,
			CanSetShutter:  true,
			CanSlave:       true,
			CanSyncAzimuth: true,
		},
		status: alpaca.DomeStatus{
			AtHome:   false,
			AtPark:   true,
			Slewing:  false,
			Slaved:   false,
			Altitude: 0.0,
			Azimuth:  0.0,
			Shutter:  alpaca.ShutterOpen,
		},
	}
}

func (d *DomeSimulator) DeviceInfo() alpaca.DeviceInfo {
	return d.info
}

func (d *DomeSimulator) DriverInfo() alpaca.DriverInfo {
	return d.driver
}

func (d *DomeSimulator) GetState() []alpaca.StateProperty {
	props := []alpaca.StateProperty{
		{
			Name:  "TimeStamp",
			Value: time.Now().Format(time.RFC3339),
		},
	}

	if d.connected {
		// If connected, add status properties
		props = append(props, d.status.ToProperties()...)
	}

	return props
}

func (d *DomeSimulator) Connected() bool {
	return d.connected
}

func (d *DomeSimulator) Connecting() bool {
	return d.connecting
}

func (d *DomeSimulator) Connect() error {
	d.connected = true
	d.logger.Infof("%s connected", d.info.Name)
	return nil
}

func (d *DomeSimulator) Disconnect() error {
	d.connected = false
	d.logger.Infof("%s disconnected", d.info.Name)
	return nil
}

func (d *DomeSimulator) Capabilities() alpaca.DomeCapabilities {
	return d.capabilities
}

func (d *DomeSimulator) Status() alpaca.DomeStatus {
	return d.status
}

func (d *DomeSimulator) SetSlaved(slaved bool) error {
	d.logger.Infof("Dome slaved: %v", slaved)
	d.status.Slaved = slaved
	return nil
}

func (d *DomeSimulator) SlewToAltitude(altitude float64) error {
	d.logger.Infof("Slewing to altitude: %f", altitude)
	d.status.Altitude = altitude
	return nil
}

func (d *DomeSimulator) SlewToAzimuth(azimuth float64) error {
	d.logger.Infof("Slewing to azimuth: %f", azimuth)
	d.status.Azimuth = azimuth
	d.status.Slewing = false
	d.status.AtPark = false
	d.status.AtHome = false
	return nil
}

func (d *DomeSimulator) SyncToAzimuth(azimuth float64) error {
	d.logger.Infof("Syncing to azimuth: %f", azimuth)
	d.status.Azimuth = azimuth
	return nil
}

func (d *DomeSimulator) AbortSlew() error {
	d.logger.Info("Aborting slew")
	d.status.Slewing = false
	return nil
}

func (d *DomeSimulator) FindHome() error {
	d.logger.Info("Finding home")
	d.status.AtHome = true
	d.status.AtPark = false
	d.status.Slewing = false
	d.status.Azimuth = float64(d.config.HomePosition)
	return nil
}

func (d *DomeSimulator) Park() error {
	d.logger.Info("Parking")
	d.status.AtHome = false
	d.status.AtPark = true
	d.status.Slewing = false
	d.status.Azimuth = float64(d.config.ParkPosition)
	return nil
}

func (d *DomeSimulator) SetPark() error {
	d.logger.Info("Setting park position")
	d.status.AtHome = false
	d.status.AtPark = true
	return nil
}

func (d *DomeSimulator) SetShutter(cmd alpaca.ShutterCommand) error {
	d.logger.Infof("Setting shutter: %v", cmd)
	switch cmd {
	case alpaca.ShutterCommandOpen:
		d.status.Shutter = alpaca.ShutterOpen
	case alpaca.ShutterCommandClose:
		d.status.Shutter = alpaca.ShutterClosed
	}
	return nil
}

func (d *DomeSimulator) HandleSetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := d.store.GetDomeConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.renderSetupForm(w, cfg, false, "")

	case http.MethodPost:
		cfg, err := parseDomeSetupForm(r)
		if err != nil {
			d.renderSetupForm(w, cfg, false, err.Error())
			return
		}

		d.logger.Infof("Setting dome config: %+v", cfg)
		d.config = cfg
		if err := d.store.SetDomeConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		d.renderSetupForm(w, cfg, true, "")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *DomeSimulator) renderSetupForm(w http.ResponseWriter, cfg DomeConfig, success bool, err string) {
	data := struct {
		DomeConfig
		Success bool
		Error   string
	}{cfg, success, err}

	if err := d.tmpl.ExecuteTemplate(w, "dome_setup.html", data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		d.logger.Errorf("Error rendering template: %v", err)
	}
}

func parseDomeSetupForm(r *http.Request) (DomeConfig, error) {
	if err := r.ParseForm(); err != nil {
		return DomeConfig{}, fmt.Errorf("error parsing form: %v", err)
	}

	homePosition, err := getFormUint(r, "home-position")
	if err != nil {
		return DomeConfig{}, err
	}
	parkPosition, err := getFormUint(r, "park-position")
	if err != nil {
		return DomeConfig{}, err
	}
	shutterTimeout, err := getFormUint(r, "shutter-timeout")
	if err != nil {
		return DomeConfig{}, err
	}
	ticksPerRevolution, err := getFormUint(r, "ticks-per-rev")
	if err != nil {
		return DomeConfig{}, err
	}

	return DomeConfig{
		HomePosition:   homePosition,
		ParkPosition:   parkPosition,
		ShutterTimeout: shutterTimeout,
		TicksPerRev:    ticksPerRevolution,
	}, nil
}

func getFormUint(r *http.Request, key string) (uint, error) {
	value := r.FormValue(key)
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", key, err)
	}
	return uint(intValue), nil
}
