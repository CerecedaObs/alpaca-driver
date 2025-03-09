// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package simulators

import (
	"alpaca/alpaca"
	"time"

	log "github.com/sirupsen/logrus"
)

const domeUID = "621ca2e0-399a-43f6-b9e7-e6575d953507"

// DomeSimulator implements the alpaca.Dome interface
type DomeSimulator struct {
	info         alpaca.DeviceInfo
	driver       alpaca.DriverInfo
	capabilities alpaca.DomeCapabilities
	status       alpaca.DomeStatus

	connected  bool
	connecting bool
}

func NewDomeSimulator(number int) *DomeSimulator {
	return &DomeSimulator{
		info: alpaca.DeviceInfo{
			Name:     "Dome Simulator",
			Type:     "Dome",
			Number:   number,
			UniqueID: domeUID,
		},
		driver: alpaca.DriverInfo{
			Name:             "ZRO Dome Driver",
			Version:          "1.0",
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
	log.Infof("%s connected", d.info.Name)
	return nil
}

func (d *DomeSimulator) Disconnect() error {
	d.connected = false
	log.Infof("%s disconnected", d.info.Name)
	return nil
}

func (d *DomeSimulator) Capabilities() alpaca.DomeCapabilities {
	return d.capabilities
}

func (d *DomeSimulator) Status() alpaca.DomeStatus {
	return d.status
}

func (d *DomeSimulator) SetSlaved(slaved bool) error {
	d.status.Slaved = slaved
	return nil
}

func (d *DomeSimulator) SlewToAltitude(altitude float64) error {
	d.status.Altitude = altitude
	return nil
}

func (d *DomeSimulator) SlewToAzimuth(azimuth float64) error {
	d.status.Azimuth = azimuth
	return nil
}

func (d *DomeSimulator) SyncToAzimuth(azimuth float64) error {
	d.status.Azimuth = azimuth
	return nil
}

func (d *DomeSimulator) AbortSlew() error {
	return nil
}

func (d *DomeSimulator) FindHome() error {
	d.status.AtHome = true
	d.status.AtPark = false
	return nil
}

func (d *DomeSimulator) Park() error {
	d.status.AtHome = false
	d.status.AtPark = true
	return nil
}

func (d *DomeSimulator) SetPark() error {
	d.status.AtHome = false
	d.status.AtPark = true
	return nil
}

func (d *DomeSimulator) SetShutter(cmd alpaca.ShutterCommand) error {
	switch cmd {
	case alpaca.ShutterCommandOpen:
		d.status.Shutter = alpaca.ShutterOpen
	case alpaca.ShutterCommandClose:
		d.status.Shutter = alpaca.ShutterClosed
	}
	return nil
}
