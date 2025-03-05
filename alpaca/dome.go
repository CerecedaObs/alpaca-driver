// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package alpaca

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const domeUID = "621ca2e0-399a-43f6-b9e7-e6575d953507"

type DomeCapabilities struct {
	CanFindHome    bool `json:"CanFindHome"`
	CanPark        bool `json:"CanPark"`
	CanSetAltitude bool `json:"CanSetAltitude"`
	CanSetAzimuth  bool `json:"CanSetAzimuth"`
	CanSetPark     bool `json:"CanSetPark"`
	CanSetShutter  bool `json:"CanSetShutter"`
	CanSlave       bool `json:"CanSlave"`
	CanSyncAzimuth bool `json:"CanSyncAzimuth"`
}

type Dome struct {
	info         DeviceInfo
	driver       DriverInfo
	capabilities DomeCapabilities

	// Dome properties
	atHome        bool
	atPark        bool
	azimuth       int
	connected     bool
	connecting    bool
	slewing       bool
	shutterStatus int
}

// Dome specific routes:
// GET /dome/{device_number}/altitude
// GET /dome/{device_number}/athome
// GET /dome/{device_number}/atpark
// GET /dome/{device_number}/azimuth

// GET /dome/{device_number}/canfindhome
// GET /dome/{device_number}/canpark
// GET /dome/{device_number}/cansetaltitude
// GET /dome/{device_number}/cansetazimuth
// GET /dome/{device_number}/cansetpark
// GET /dome/{device_number}/cansetshutter
// GET /dome/{device_number}/canslave
// GET /dome/{device_number}/cansyncazimuth

// GET /dome/{device_number}/shutterstatus
// GET /dome/{device_number}/slaved
// PUT /dome/{device_number}/slaved
// GET /dome/{device_number}/slewing
// PUT /dome/{device_number}/abortslew
// PUT /dome/{device_number}/closeshutter
// PUT /dome/{device_number}/findhome
// PUT /dome/{device_number}/openshutter
// PUT /dome/{device_number}/park
// PUT /dome/{device_number}/setpark
// PUT /dome/{device_number}/slewtoaltitude
// PUT /dome/{device_number}/slewtoazimuth
// PUT /dome/{device_number}/synctoazimuth

func NewDome() *Dome {
	return &Dome{
		info: DeviceInfo{
			Name:     "ZRO Dome",
			Type:     "Dome",
			Number:   0,
			UniqueID: domeUID,
		},
		driver: DriverInfo{
			Name:             "ZRO Dome Driver",
			Version:          "1.0",
			InterfaceVersion: 1,
		},
	}
}

func (d *Dome) DeviceInfo() DeviceInfo {
	return d.info
}

func (d *Dome) DriverInfo() DriverInfo {
	return d.driver
}

func (d *Dome) GetState() []StateProperty {
	props := []StateProperty{
		{
			Name:  "TimeStamp",
			Value: time.Now().Format(time.RFC3339),
		},
	}

	if d.connected {
		props = append(props, []StateProperty{
			{
				Name:  "Altitude",
				Value: 0,
			},
			{
				Name:  "AtHome",
				Value: d.atHome,
			},
			{
				Name:  "AtPark",
				Value: d.atPark,
			},
			{
				Name:  "Azimuth",
				Value: d.azimuth,
			},
			{
				Name:  "ShutterStatus",
				Value: d.shutterStatus,
			},
			{
				Name:  "Slewing",
				Value: d.slewing,
			},
		}...)
	}

	return props
}

func (d *Dome) Connected() bool {
	return d.connected
}

func (d *Dome) Connecting() bool {
	return d.connecting
}

func (d *Dome) Connect() error {
	d.connected = true
	log.Infof("%s connected", d.info.Name)
	return nil
}

func (d *Dome) Disconnect() error {
	d.connected = false
	log.Infof("%s disconnected", d.info.Name)
	return nil
}
