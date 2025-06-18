package zro

import (
	"alpaca/pkg/alpaca"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	// TODO: domeUID should be unique for each device.
	domeUID       = "0a0af300-b0fc-4178-b758-caa109fc836f"
	deviceName    = "ZRO Dome"
	deviceType    = "Dome"
	driverName    = "ZRO Dome Driver"
	driverVersion = "1.0"
)

type connState int

const (
	connStateDisconnected connState = iota
	connStateConnecting
	connStateConnected
)

// createMQTTClient initializes and returns a new MQTT client using the configuration
// retrieved from the provided alpaca.Store. It allows overriding the MQTT broker,
// username, and password via CLI context flags.
func createMQTTClient(cfg MQTTConfig) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.SetClientID("zro-alpaca")
	opts.AddBroker(cfg.Host)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}
	return mqttClient, nil
}

// Driver represents the ZRO dome Alpaca driver.
type Driver struct {
	number int                // Driver number
	store  *store             // Configuration store
	tmpl   *template.Template // HTML template for rendering the setup form
	state  connState          // Connection state
	slaved bool               // Slaved state
	logger log.FieldLogger

	// The MQTT client and the controller are created when the driver is connected
	client mqtt.Client        // MQTT client
	dome   *Dome              // ZRO dome controller
	cancel context.CancelFunc // Context cancel function
}

func NewDriver(number int, db *bolt.DB, tmpl *template.Template, logger log.FieldLogger) (*Driver, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %v", err)
	}

	driver := Driver{
		number: number,
		tmpl:   tmpl,
		store:  store,
		state:  connStateDisconnected,
		logger: logger,
	}

	return &driver, nil
}

func (d *Driver) Close() {
	d.logger.Info("Closing ZRO driver")

	if d.state == connStateDisconnected {
		if d.cancel != nil {
			d.cancel()
			d.cancel = nil
		}
		return
	}
	if err := d.Disconnect(); err != nil {
		d.logger.Errorf("failed to disconnect: %v", err)
	}
}

func (d *Driver) Connect() error {
	config, err := d.store.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get dome config: %v", err)
	}

	if d.state != connStateDisconnected {
		return fmt.Errorf("driver is already connected")
	}

	d.state = connStateConnecting

	client, err := createMQTTClient(config.MQTTConfig)
	if err != nil {
		return fmt.Errorf("failed to create MQTT client: %v", err)
	}

	d.client = client
	d.dome, err = NewDome(client, config, d.logger)
	if err != nil {
		d.client.Disconnect(100)
		d.state = connStateDisconnected
		return fmt.Errorf("failed to create ZRO dome controller: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	go func() {
		d.dome.Run(ctx)
	}()

	d.state = connStateConnected

	d.logger.Info("Connected to MQTT broker")

	return nil
}

func (d *Driver) Disconnect() error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.client.Disconnect(100)
	d.state = connStateDisconnected
	d.logger.Info("Disconnected from MQTT broker")
	return nil
}

func (d *Driver) Connecting() bool {
	return d.state == connStateConnecting
}

func (d *Driver) Connected() bool {
	return d.state == connStateConnected
}

func (d *Driver) GetState() []alpaca.StateProperty {
	props := []alpaca.StateProperty{
		{
			Name:  "TimeStamp",
			Value: time.Now().Format(time.RFC3339),
		},
	}

	if d.state == connStateConnected {
		props = append(props, d.Status().ToProperties()...)
	}

	return props
}

func (d *Driver) Status() alpaca.DomeStatus {
	if d.state != connStateConnected {
		return alpaca.DomeStatus{}
	}

	st := d.dome.GetStatus()

	status := alpaca.DomeStatus{
		Azimuth:  d.dome.ticksToDegrees(st.Position),
		AtHome:   st.AtHome,
		AtPark:   st.AtHome, // TODO: Implement park status
		Slewing:  st.Slewing,
		Slaved:   d.slaved,
		Altitude: 0.0,
		Shutter:  alpaca.ShutterOpen,
	}
	return status
}

func (d *Driver) Capabilities() alpaca.DomeCapabilities {
	return alpaca.DomeCapabilities{
		CanFindHome:    true,
		CanPark:        true,
		CanSetAltitude: false,
		CanSetAzimuth:  true,
		CanSetPark:     true,
		CanSetShutter:  true,
		CanSlave:       true,
		CanSyncAzimuth: true,
	}
}

func (d *Driver) DeviceInfo() alpaca.DeviceInfo {
	return alpaca.DeviceInfo{
		Name:     deviceName,
		Type:     deviceType,
		Number:   d.number,
		UniqueID: domeUID,
	}
}

func (d *Driver) DriverInfo() alpaca.DriverInfo {
	return alpaca.DriverInfo{
		Name:             driverName,
		Version:          driverVersion,
		InterfaceVersion: 1,
	}
}

func (d *Driver) SlewToAzimuth(az float64) error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	return d.dome.SlewToAzimuth(az)
}

func (d *Driver) SyncToAzimuth(azimuth float64) error {
	if d.state != connStateConnected {
		return alpaca.ErrNotConnected
	}
	d.logger.Warn("SyncToAzimuth not implemented")
	return nil
}

func (d *Driver) SlewToAltitude(altitude float64) error {
	return alpaca.ErrPropertyNotImplemented
}

func (d *Driver) SyncToAltitude(altitude float64) error {
	return alpaca.ErrPropertyNotImplemented
}

func (d *Driver) AbortSlew() error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	return d.dome.AbortSlew()
}

func (d *Driver) FindHome() error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	return d.dome.FindHome()
}

func (d *Driver) Park() error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	return d.dome.Park()
}

func (d *Driver) SetPark() error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	// TODO: store the park position in the config
	return d.dome.SetPark()
}

func (d *Driver) SetSlaved(slaved bool) error {
	d.logger.Infof("Dome slaved: %v", slaved)
	d.slaved = slaved
	return nil
}

func (d *Driver) SetShutter(command alpaca.ShutterCommand) error {
	if d.state != connStateConnected {
		return ErrNotConnected
	}

	var cmd ShutterCommand
	switch command {
	case alpaca.ShutterCommandOpen:
		cmd = ShutterOpen
	case alpaca.ShutterCommandClose:
		cmd = ShutterClose
	default:
		return fmt.Errorf("invalid shutter command: %v", command)
	}
	return d.dome.SetShutter(cmd)
}

func (d *Driver) HandleSetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := d.store.GetConfig()
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
		if err := d.store.SetConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		d.renderSetupForm(w, cfg, true, "")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *Driver) renderSetupForm(w http.ResponseWriter, cfg Config, success bool, err string) {
	data := struct {
		Config
		Success bool
		Error   string
	}{cfg, success, err}

	if err := d.tmpl.ExecuteTemplate(w, "dome_zro_setup.html", data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		d.logger.Errorf("Error rendering template: %v", err)
	}
}

func parseDomeSetupForm(r *http.Request) (Config, error) {
	if err := r.ParseForm(); err != nil {
		return Config{}, fmt.Errorf("error parsing form: %v", err)
	}

	cfg := defaultConfig
	cfg.Host = r.FormValue("mqtt-host")
	cfg.Username = r.FormValue("mqtt-username")
	cfg.Password = r.FormValue("mqtt-password")
	cfg.TopicRoot = r.FormValue("mqtt-topic-root")

	cfg.TicksPerTurn, _ = strconv.Atoi(r.FormValue("ticks-per-turn"))
	cfg.Tolerance, _ = strconv.Atoi(r.FormValue("tolerance"))
	cfg.HomePosition, _ = strconv.ParseFloat(r.FormValue("home-position"), 64)
	cfg.ParkPosition, _ = strconv.ParseFloat(r.FormValue("park-position"), 64)
	cfg.AzimuthTimeout, _ = strconv.Atoi(r.FormValue("azimuth-timeout"))
	cfg.MaxSpeed, _ = strconv.Atoi(r.FormValue("max-speed"))
	cfg.MinSpeed, _ = strconv.Atoi(r.FormValue("min-speed"))
	cfg.BrakeSpeed, _ = strconv.Atoi(r.FormValue("brake-speed"))
	cfg.VelTimeout, _ = strconv.Atoi(r.FormValue("vel-timeout"))
	cfg.ShortDistance, _ = strconv.Atoi(r.FormValue("short-distance"))
	cfg.ShutterTimeout, _ = strconv.Atoi(r.FormValue("shutter-timeout"))

	cfg.ParkOnShutter = r.FormValue("park-on-shutter") == "true"
	cfg.UseShutter = r.FormValue("use-shutter") == "true"

	return cfg, nil
}
