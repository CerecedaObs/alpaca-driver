// This code does not depend on any other code in the repository.
// It is a standalone implementation of the ZRO dome controller driver.
// It uses the Paho MQTT library to communicate with the ZRO dome controller
// and handles the configuration, telemetry, and commands for the dome.
// The code is structured to be easily integrated into a larger system,
// with logging and error handling in place.

package dome

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

var ErrNotConnected = fmt.Errorf("driver is not connected")

type Direction int

const (
	DirCW Direction = iota
	DirCCW
)

type ShutterCommand int

const (
	ShutterOpen ShutterCommand = iota
	ShutterClose
)

type ShutterStatus int

const (
	ShutterStatusClosed ShutterStatus = iota
	ShutterStatusOpening
	ShutterStatusOpen
	ShutterStatusClosing
	ShutterStatusAborted
	ShutterStatusError
)

type cmdCode uint8

// Dome commands
const (
	// Configuration commands
	cmdLoad    cmdCode = 'L' // Load dome configuration parameters
	cmdSetPark cmdCode = 'P' // Set park coordinates and policy (it does't work yet)
	cmdTicks   cmdCode = 'T' // Set the number of ticks per revolution

	// Shutter commands
	cmdConnectShutter    cmdCode = 'X' // Connect to the shutter
	cmdDisconnectShutter cmdCode = 'Z' // Disconnect from the shutter
	cmdOpenShutter       cmdCode = 'O' // Open shutter
	cmdCloseShutter      cmdCode = 'C' // Close shutter
	cmdShutter           cmdCode = 'U' // Send a command to the shutter

	// Dome movement commands
	cmdAbort cmdCode = 'A' // Abort azimuth movement
	cmdMove  cmdCode = 'M' // Move azimuth
	cmdHome  cmdCode = 'H' // Move to 'home' position
	cmdGoto  cmdCode = 'G' // Go to a specific azimuth position
	cmdPark  cmdCode = 'K' // Park the dome

	// Information commands
	cmdStatus      cmdCode = 'S' // Read the dome status
	cmdVersion     cmdCode = 'V' // Read firmware version
	cmdBattery     cmdCode = 'B' // Read shutter's battery voltage and current
	cmdTemperature cmdCode = 't' // Read temperature in Celsius
	cmdHumidity    cmdCode = 'u' // Read humidity percentage
	cmdHelp        cmdCode = 'h' // Return a list of available commands

	// cmdUnknown cmdCode = '?' // Unknown command
)

type MQTTConfig struct {
	Host      string
	Username  string
	Password  string
	TopicRoot string // Root topic for the ZRO dome controller
}

type Config struct {
	MQTTConfig

	TicksPerTurn   int     // Encoder ticks per dome revolution
	Tolerance      int     // Tolerance in encoder ticks
	HomePosition   float64 // Home position in degrees
	ParkPosition   float64 // Park position in degrees
	AzimuthTimeout int     // Azimuth timeout in seconds
	MaxSpeed       int     // Maximum speed in encoder ticks per second
	MinSpeed       int     // Minimum speed in encoder ticks per second
	BrakeSpeed     int     // Brake speed in encoder ticks per second
	EncoderDiv     int     // Encoder divisor (for high-resolution encoders)
	VelTimeout     int     // Velocity timeout in seconds
	ShortDistance  int     // Short distance in encoder ticks
	ParkOnShutter  bool    // True if the dome should park on shutter
	ShutterTimeout int     // Shutter timeout in seconds
	UseShutter     bool    // True if the shutter is used
}

func DefaultConfig() Config {
	return Config{
		MQTTConfig: MQTTConfig{
			Host:      "tcp://localhost:1883",
			Username:  "",
			Password:  "",
			TopicRoot: "/ZRO",
		},
		TicksPerTurn:   10476,
		Tolerance:      4,
		HomePosition:   0,
		ParkPosition:   0,
		AzimuthTimeout: 20000,
		MaxSpeed:       200,
		MinSpeed:       30,
		BrakeSpeed:     80,
		VelTimeout:     10,
		ShortDistance:  100,
		ParkOnShutter:  false,
		ShutterTimeout: 0,
		UseShutter:     true,
		EncoderDiv:     1, // Default encoder divisor
	}
}

func (c *Config) Validate() error {
	if c.TicksPerTurn <= 0 {
		return fmt.Errorf("ticks per turn must be greater than 0")
	}
	if c.Tolerance < 0 {
		return fmt.Errorf("tolerance must be non-negative")
	}
	if c.AzimuthTimeout <= 0 {
		return fmt.Errorf("azimuth timeout must be greater than 0")
	}
	if c.MaxSpeed <= 0 {
		return fmt.Errorf("maximum speed must be greater than 0")
	}
	if c.MinSpeed <= 0 {
		return fmt.Errorf("minimum speed must be greater than 0")
	}
	if c.BrakeSpeed <= 0 {
		return fmt.Errorf("brake speed must be greater than 0")
	}
	if c.EncoderDiv <= 0 {
		return fmt.Errorf("encoder divisor must be greater than 0")
	}
	return nil
}

// Status represents the status of the ZRO dome controller.
type Status struct {
	Position int       // Azimuth position in encoder ticks
	AtHome   bool      // True if the dome is at home position
	Slewing  bool      // True if the dome is slewing
	Dir      Direction // Direction of movement (CW or CCW)
	Target   int       // Target position in encoder ticks

	Temperature float32
	Humidity    float32

	BatteryVoltage float32
	BatteryCurrent float32

	Version string // Firmware version

	Shutter          ShutterStatus // Shutter status
	ShutterConnected bool          // True if shutter is connected
}

// telemetryMsg represents the telemetry message received periodically from the
// ZRO dome controller under the "telemetry" topic.
type telemetryMsg struct {
	AzState     int           `json:"az_state"` // State of the azimuth state machine
	ShState     ShutterStatus `json:"sh_state"`
	Position    int           `json:"pos"`
	Home        int           `json:"home"`
	Dir         int           `json:"dir"`
	Target      int           `json:"target"`
	Link        int           `json:"link"`
	Temperature float32       `json:"temp"`
	Humidity    float32       `json:"hum"`
}

// batteryMsg represents the battery message received periodically from the
// ZRO dome controller under the "battery" topic.“
type batteryMsg struct {
	Voltage float32 `json:"batt_voltage"`
	Current float32 `json:"batt_current"`
}

type Response struct {
	Code  cmdCode // The code of the command that was sent
	Value any     // The value of the response
	Error bool    // True if there was an error
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Normalize the an angle in degrees to the range [0, 360)
func normalizeAngle(angle float64) float64 {
	for angle < 0 {
		angle += 360
	}
	return math.Mod(angle+360, 360)
}

// Dome represents the ZRO dome controller.
// It is controlled via MQTT messages.
type Dome struct {
	client mqtt.Client // MQTT client

	status Status
	config Config // Configuration parameters

	responseChan chan Response // Channel for responses from the ZRO dome controller
	logger       log.FieldLogger

	// shutterLink bool   // True if the shutter is linked to the dome
}

func NewDome(client mqtt.Client, config Config, logger log.FieldLogger) (*Dome, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	dome := &Dome{
		client:       client,
		config:       config,
		responseChan: make(chan Response, 1),
		logger:       logger,
	}

	// Initialize shutter status as unknown/closed
	dome.status.Shutter = ShutterStatusClosed
	dome.status.ShutterConnected = false

	return dome, nil
}

func (d *Dome) DegreesToTicks(degrees float64) int {
	return int(normalizeAngle(degrees-d.config.HomePosition) * float64(d.config.TicksPerTurn) / 360.0)
}

func (d *Dome) TicksToDegrees(ticks int) float64 {
	return float64(ticks)*360.0/float64(d.config.TicksPerTurn) + d.config.HomePosition
}

// Run connects to the ZRO dome controller and subscribes to the necessary topics.
// When the context is cancelled, it unsubscribes from the topics and disconnects.
func (d *Dome) Run(ctx context.Context) error {
	if !d.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	root := d.config.MQTTConfig.TopicRoot

	// Subscribe to telemetry topic
	telemetryTopic := root + "/telemetry"
	if token := d.client.Subscribe(telemetryTopic, 0, d.telemetryHandler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to telemetry topic: %v", token.Error())
	}
	defer d.client.Unsubscribe(telemetryTopic)

	// Subscribe to battery topic
	batteryTopic := root + "/battery"
	if token := d.client.Subscribe(batteryTopic, 0, d.batteryHandler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to battery topic: %v", token.Error())
	}
	defer d.client.Unsubscribe(batteryTopic)

	// Subscribe to responses topic
	responseTopic := root + "/responses"
	if token := d.client.Subscribe(responseTopic, 0, d.responseHandler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to responses topic: %v", token.Error())
	}
	defer d.client.Unsubscribe(responseTopic)

	// Connect to the shutter
	if d.config.UseShutter {
		if err := d.connectShutter(); err != nil {
			return fmt.Errorf("failed to connect to shutter: %v", err)
		}
		defer d.disconnectShutter()
	}

	// Read status, firmware version and battery status
	if err := d.sendCommand(string(cmdStatus)); err != nil {
		return fmt.Errorf("failed to send status command: %v", err)
	}
	if err := d.sendCommand(string(cmdVersion)); err != nil {
		return fmt.Errorf("failed to send version command: %v", err)
	}
	if err := d.sendCommand(string(cmdBattery)); err != nil {
		return fmt.Errorf("failed to send battery command: %v", err)
	}

	// Set the configuration
	if err := d.setConfig(d.config); err != nil {
		return fmt.Errorf("failed to set configuration: %v", err)
	}

	<-ctx.Done()
	d.logger.Info("Stopping ZRO dome controller")
	return nil
}

// sendCommandWithTimeout sends a command and waits for response with custom timeout
func (d *Dome) sendCommandWithTimeout(cmd string, timeout time.Duration) error {
	if !d.client.IsConnected() {
		return ErrNotConnected
	}

	// Create the message string
	msg := "_" + cmd + ";"
	d.logger.Debugf("Sending command: %s", msg)

	// Publish the command to the ZRO dome controller
	topic := d.config.TopicRoot + "/commands"
	if token := d.client.Publish(topic, 0, false, msg); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish command: %v", token.Error())
	}

	// Wait for the response with custom timeout
	select {
	case resp := <-d.responseChan:
		if resp.Error {
			return fmt.Errorf("command failed: %c", resp.Code)
		}

		if resp.Code != cmdCode(cmd[0]) {
			return fmt.Errorf("unexpected response command: %c", resp.Code)
		}

		d.logger.Debugf("Response: %+v", resp)
		return nil

	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for response")
	}
}

func (d *Dome) sendCommand(cmd string) error {
	return d.sendCommandWithTimeout(cmd, 5*time.Second)
}

// setConfig sends the configuration to the ZRO dome controller.
// Each parameter is sent as a command with the format "_L<param>=<value>;"
// All values are integers. Example: "_LTICK=1000;"
func (d *Dome) setConfig(config Config) error {
	if !d.client.IsConnected() {
		return ErrNotConnected
	}

	cfgMap := map[string]int{
		"TICK": config.TicksPerTurn,
		"TOLE": config.Tolerance,
		"PKPO": d.DegreesToTicks(config.ParkPosition),
		"AZTO": config.AzimuthTimeout,
		"MXSP": config.MaxSpeed,
		"MNSP": config.MinSpeed,
		"BKSP": config.BrakeSpeed,
		"VLTO": config.VelTimeout,
		"SHDS": config.ShortDistance,
		"POSH": boolToInt(config.ParkOnShutter),
		"ENDV": config.EncoderDiv, // Encoder divisor for the shutter
	}

	for param, value := range cfgMap {
		if err := d.sendCommand(fmt.Sprintf("%c%s=%d", cmdLoad, param, value)); err != nil {
			return fmt.Errorf("failed to send config parameter %s: %v", param, err)
		}
	}
	return nil
}

// telemetryHandler processes the telemetry messages.
func (d *Dome) telemetryHandler(client mqtt.Client, msg mqtt.Message) {
	var telemetry telemetryMsg
	if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
		d.logger.Errorf("Failed to unmarshal telemetry message: %v", err)
		return
	}

	d.logger.Debugf("Telemetry: %+v", telemetry)

	d.status.Position = telemetry.Position
	d.status.Dir = Direction(telemetry.Dir)
	d.status.Target = telemetry.Target
	d.status.AtHome = telemetry.Home == 1

	// Determine if the dome is slewing
	d.status.Slewing = telemetry.AzState > 0 && telemetry.AzState < 5

	d.status.Temperature = telemetry.Temperature
	d.status.Humidity = telemetry.Humidity
}

// batteryHandler processes the battery messages.
func (d *Dome) batteryHandler(client mqtt.Client, msg mqtt.Message) {
	var battery batteryMsg
	if err := json.Unmarshal(msg.Payload(), &battery); err != nil {
		d.logger.Errorf("Failed to unmarshal battery message: %v", err)
		return
	}

	d.logger.Debugf("Battery: %+v", battery)

	d.status.BatteryVoltage = battery.Voltage
	d.status.BatteryCurrent = battery.Current
}

func (d *Dome) responseHandler(client mqtt.Client, msg mqtt.Message) {
	resp, err := parseResponse(string(msg.Payload()))
	if err != nil {
		d.logger.Errorf("Failed to parse response: %v", err)
		return
	}
	d.logger.Debugf("Response received: %+v", resp)

	// Handle the response based on the command
	switch resp.Code {
	case cmdStatus:
	case cmdBattery:
	case cmdVersion:
		d.status.Version = strings.Trim(resp.Value.(string), "()")
		d.logger.Infof("Dome controller firmware version: %s", d.status.Version)
	case cmdConnectShutter:
		if !resp.Error {
			d.status.ShutterConnected = true
			d.logger.Info("Shutter connected successfully")
		} else {
			d.status.ShutterConnected = false
			d.logger.Error("Failed to connect to shutter")
		}
	case cmdDisconnectShutter:
		d.status.ShutterConnected = false
		d.logger.Info("Shutter disconnected")
	}

	// Attempt to send the response to the channel with a timeout
	select {
	case d.responseChan <- resp:
		// Successfully sent the response
	case <-time.After(1 * time.Second):
		d.logger.Warn("Timeout while sending response to the channel")
	}
}

// Responses have the format:
// "_ACK_<command>;"
// "_ACK_<command>=<value>;"
// "_NACK_<command>;"
func parseResponse(msg string) (Response, error) {
	var resp Response

	fields := strings.Split(msg, "_")
	if len(fields) != 3 {
		return resp, fmt.Errorf("bad number of fields: %s", msg)
	}
	if !strings.HasSuffix(fields[2], ";") {
		return resp, fmt.Errorf("invalid response suffix: %s", msg)
	}

	// Check if the response is an acknowledgment or not
	if fields[1] == "NACK" {
		resp.Error = true
	} else if fields[1] != "ACK" {
		return resp, fmt.Errorf("invalid response format: %s", msg)
	}

	// Extract the command and value
	cmd := strings.Trim(fields[2], ";")

	parts := strings.Split(cmd, "=")
	resp.Code = cmdCode(parts[0][0])

	if len(parts) == 2 {
		resp.Value = parts[1]
	} else if len(parts) != 1 {
		return resp, fmt.Errorf("invalid response value: %s", msg)
	}

	return resp, nil
}

func (d *Dome) GetStatus() Status {
	return d.status
}

func (d *Dome) SlewToAzimuth(az float64) error {
	ticks := d.DegreesToTicks(az)
	return d.sendCommand(fmt.Sprintf("%c=%d", cmdGoto, ticks))
}

func (d *Dome) AbortSlew() error {
	return d.sendCommand(string(cmdAbort))
}

func (d *Dome) FindHome() error {
	return d.sendCommand(string(cmdHome))
}

func (d *Dome) Park() error {
	return d.sendCommand(string(cmdPark))
}

func (d *Dome) SetPark() error {
	// Get current position as the new park position
	currentTicks := d.status.Position

	// Send the park position using the load command
	return d.sendCommand(fmt.Sprintf("%c%s=%d", cmdLoad, "PKPO", currentTicks))
}

func (d *Dome) SetShutter(command ShutterCommand) error {
	if !d.config.UseShutter {
		return fmt.Errorf("shutter not supported")
	}

	var cmd cmdCode
	switch command {
	case ShutterOpen:
		cmd = cmdOpenShutter
		d.status.Shutter = ShutterStatusOpening
	case ShutterClose:
		cmd = cmdCloseShutter
		d.status.Shutter = ShutterStatusClosing
	default:
		return fmt.Errorf("invalid shutter command: %d", command)
	}

	return d.sendCommand(string(cmd))
}

// connectShutter attempts to connect to the shutter with retries
func (d *Dome) connectShutter() error {
	const maxRetries = 10
	const retryDelay = time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		d.logger.Infof("Connecting to shutter (attempt %d/%d)", attempt, maxRetries)

		// Send connect command
		if err := d.sendCommandWithTimeout(string(cmdConnectShutter), retryDelay); err != nil {
			d.logger.Warnf("Shutter connect attempt %d failed: %v", attempt, err)

			// If this was the last attempt, return the error
			if attempt == maxRetries {
				return fmt.Errorf("failed to connect to shutter after %d attempts: %v", maxRetries, err)
			}

			// Wait before retrying
			time.Sleep(retryDelay)
			continue
		}

		// Success - shutter connected
		d.logger.Infof("Shutter connected successfully on attempt %d", attempt)
		return nil
	}

	return fmt.Errorf("failed to connect to shutter after %d attempts", maxRetries)
}

// disconnectShutter disconnects from the shutter
func (d *Dome) disconnectShutter() error {
	if !d.config.UseShutter {
		return nil // Nothing to disconnect if shutter is not used
	}

	d.logger.Info("Disconnecting from shutter")

	if err := d.sendCommand(string(cmdDisconnectShutter)); err != nil {
		d.logger.Warnf("Failed to send disconnect shutter command: %v", err)
		// Don't return error, just log warning since we're disconnecting anyway
	}

	// Update status regardless of command success
	d.status.ShutterConnected = false
	d.logger.Info("Shutter disconnected")

	return nil
}
