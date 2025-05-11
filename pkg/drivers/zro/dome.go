package zro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

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

type cmdCode uint8

// Dome commands
const (
	// Configuration commands
	cmdLoad    cmdCode = 'L' // Load dome configuration parameters
	cmdSetPark cmdCode = 'P' // Set park coordinates and policy
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

	cmdUnknown cmdCode = '?' // Unknown command
)

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

	// Shutter  ShutterStatus
	// ShutterConnected bool
}

// telemetryMsg represents the telemetry message received periodically from the
// ZRO dome controller under the "telemetry" topic.
type telemetryMsg struct {
	AzState     int     `json:"az_state"` // State of the azimuth state machine
	Position    int     `json:"pos"`
	Home        int     `json:"home"`
	Dir         int     `json:"dir"`
	Target      int     `json:"target"`
	Link        int     `json:"link"`
	Temperature float32 `json:"temp"`
	Humidity    float32 `json:"hum"`
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

// ZRO represents the ZRO dome controller.
// The dome is controlled by MQTT messages.
type ZRO struct {
	client mqtt.Client // MQTT client

	status Status
	config Config // Configuration parameters

	responseChan chan Response // Channel for responses from the ZRO dome controller
	logger       log.FieldLogger

	// shutterLink bool   // True if the shutter is linked to the dome
}

func NewZRO(client mqtt.Client, config Config, logger log.FieldLogger) *ZRO {
	return &ZRO{
		client:       client,
		config:       config,
		responseChan: make(chan Response, 1),
		logger:       logger.WithField("component", "ZRO"),
	}
}

func (d *ZRO) degreesToTicks(degrees float64) int {
	return int(normalizeAngle(degrees) * float64(d.config.TicksPerTurn) / 360.0)
}

func (d *ZRO) ticksToDegrees(ticks int) float64 {
	return float64(ticks) * 360.0 / float64(d.config.TicksPerTurn)
}

// Run connects to the ZRO dome controller and subscribes to the necessary topics.
// When the context is cancelled, it unsubscribes from the topics and disconnects.
func (d *ZRO) Run(ctx context.Context) {
	if !d.client.IsConnected() {
		d.logger.Error("MQTT client is not connected")
		return
	}

	root := d.config.MQTTConfig.TopicRoot

	// Subscribe to telemetry topic
	telemetryTopic := root + "/telemetry"
	if token := d.client.Subscribe(telemetryTopic, 0, d.telemetryHandler); token.Wait() && token.Error() != nil {
		d.logger.Errorf("Failed to subscribe to telemetry topic: %v", token.Error())
		return
	}
	defer d.client.Unsubscribe(telemetryTopic)

	// Subscribe to battery topic
	batteryTopic := root + "/battery"
	if token := d.client.Subscribe(batteryTopic, 0, d.batteryHandler); token.Wait() && token.Error() != nil {
		d.logger.Errorf("Failed to subscribe to battery topic: %v", token.Error())
		return
	}
	defer d.client.Unsubscribe(batteryTopic)

	// Subscribe to responses topic
	responseTopic := root + "/responses"
	if token := d.client.Subscribe(responseTopic, 0, d.responseHandler); token.Wait() && token.Error() != nil {
		d.logger.Errorf("Failed to subscribe to responses topic: %v", token.Error())
		return
	}
	defer d.client.Unsubscribe(responseTopic)

	// Connect to the shutter
	if d.config.UseShutter {
		if err := d.sendCommand(string(cmdConnectShutter)); err != nil {
			d.logger.Errorf("Failed to send connect shutter command: %v", err)
			return
		}
		defer d.sendCommand(string(cmdDisconnectShutter))
	}

	// Read status, firmware version and battery status
	if err := d.sendCommand(string(cmdStatus)); err != nil {
		d.logger.Errorf("Failed to send status command: %v", err)
		return
	}
	if err := d.sendCommand(string(cmdVersion)); err != nil {
		d.logger.Errorf("Failed to send version command: %v", err)
		return
	}
	if err := d.sendCommand(string(cmdBattery)); err != nil {
		d.logger.Errorf("Failed to send battery command: %v", err)
		return
	}

	<-ctx.Done()
}

func (d *ZRO) sendCommand(cmd string) error {
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

	// Wait for the response
	select {
	case resp := <-d.responseChan:
		if resp.Error {
			return fmt.Errorf("command failed: %c", resp.Code)
		}

		if resp.Code != cmdCode(cmd[0]) {
			return fmt.Errorf("unexpected response command: %c", resp.Code)
		}

		d.logger.Debugf("Response: %+v", resp)
		// TODO: Check if the response value is valid

	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for response")
	}

	return nil
}

// telemetryHandler processes the telemetry messages.
func (d *ZRO) telemetryHandler(client mqtt.Client, msg mqtt.Message) {
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
func (d *ZRO) batteryHandler(client mqtt.Client, msg mqtt.Message) {
	var battery batteryMsg
	if err := json.Unmarshal(msg.Payload(), &battery); err != nil {
		d.logger.Errorf("Failed to unmarshal battery message: %v", err)
		return
	}

	d.logger.Debugf("Battery: %+v", battery)

	d.status.BatteryVoltage = battery.Voltage
	d.status.BatteryCurrent = battery.Current
}

func (d *ZRO) responseHandler(client mqtt.Client, msg mqtt.Message) {
	resp, err := parseResponse(string(msg.Payload()))
	if err != nil {
		d.logger.Errorf("Failed to parse response: %v", err)
		return
	}

	// Handle the response based on the command
	switch resp.Code {
	case cmdStatus:
	case cmdBattery:
		// Ignore those responses
	case cmdVersion:
		d.status.Version = strings.Trim(resp.Value.(string), "()")
		d.logger.Infof("Dome controller firmware version: %s", d.status.Version)
	default:
		d.logger.Warnf("Unknown response command: %c", resp.Code)
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
	if len(parts[0]) != 1 {
		return resp, fmt.Errorf("invalid command format: %s", msg)
	}
	resp.Code = cmdCode(parts[0][0])

	if len(parts) == 2 {
		resp.Value = parts[1]
	} else if len(parts) != 1 {
		return resp, fmt.Errorf("invalid response value: %s", msg)
	}

	return resp, nil
}

// SetConfig sends the configuration to the ZRO dome controller.
// Each parameter is sent as a command with the format "_L<param>=<value>;"
// All values are integers. Example: "_LTICK=1000;"
func (d *ZRO) SetConfig(config Config) error {
	if !d.client.IsConnected() {
		return ErrNotConnected
	}

	cfgMap := map[string]int{
		"TICK": config.TicksPerTurn,
		"TOLE": config.Tolerance,
		"PKPO": d.degreesToTicks(config.ParkPosition),
		"POSH": d.degreesToTicks(config.HomePosition),
		"AZTO": config.AzimuthTimeout,
		"MXSP": config.MaxSpeed,
		"MNSP": config.MinSpeed,
		"BKSP": config.BrakeSpeed,
		"VLTO": config.VelTimeout,
		"SHDS": config.ShortDistance,
		"ENDV": boolToInt(config.ParkOnShutter),
	}

	for param, value := range cfgMap {
		if err := d.sendCommand(fmt.Sprintf("L%s=%d", param, value)); err != nil {
			return fmt.Errorf("failed to send config parameter %s: %v", param, err)
		}
	}
	return nil
}

func (d *ZRO) GetStatus() Status {
	return d.status
}

func (d *ZRO) SlewToAzimuth(az float64) error {
	ticks := d.degreesToTicks(az)
	return d.sendCommand(fmt.Sprintf("G=%d", ticks))
}

func (d *ZRO) AbortSlew() error {
	return d.sendCommand(string(cmdAbort))
}

func (d *ZRO) FindHome() error {
	return d.sendCommand(string(cmdHome))
}

func (d *ZRO) Park() error {
	return d.sendCommand(string(cmdPark))
}

func (d *ZRO) SetPark() error {
	return d.sendCommand(string(cmdSetPark))
}

func (d *ZRO) SetShutter(command ShutterCommand) error {
	var cmd string
	switch command {
	case ShutterOpen:
		cmd = string(cmdOpenShutter)
	case ShutterClose:
		cmd = string(cmdCloseShutter)
	default:
		return fmt.Errorf("invalid shutter command: %d", command)
	}

	return d.sendCommand(cmd)
}
