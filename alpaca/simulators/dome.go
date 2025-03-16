// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package simulators

import (
	"alpaca/alpaca"
	"html/template"
	"net/http"
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
	db     *bolt.DB

	info         alpaca.DeviceInfo
	driver       alpaca.DriverInfo
	capabilities alpaca.DomeCapabilities
	status       alpaca.DomeStatus

	connected  bool
	connecting bool
}

func NewDomeSimulator(number int, db *bolt.DB, tmpl *template.Template, logger log.FieldLogger) *DomeSimulator {
	return &DomeSimulator{
		logger: logger,
		tmpl:   tmpl,
		db:     db,

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
			CanSetAltitude: true,
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
	return nil
}

func (d *DomeSimulator) SyncToAzimuth(azimuth float64) error {
	d.logger.Infof("Syncing to azimuth: %f", azimuth)
	d.status.Azimuth = azimuth
	return nil
}

func (d *DomeSimulator) AbortSlew() error {
	d.logger.Info("Aborting slew")
	return nil
}

func (d *DomeSimulator) FindHome() error {
	d.logger.Info("Finding home")
	d.status.AtHome = true
	d.status.AtPark = false
	return nil
}

func (d *DomeSimulator) Park() error {
	d.logger.Info("Parking")
	d.status.AtHome = false
	d.status.AtPark = true
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
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use the pre-parsed template
	err := d.tmpl.ExecuteTemplate(w, "dome_setup.html", d)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		d.logger.Errorf("Error rendering template: %v", err)
	}
}
