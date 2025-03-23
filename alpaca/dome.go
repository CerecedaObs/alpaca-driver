package alpaca

import (
	"net/http"
)

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

type ShutterStatus int

const (
	ShutterOpen ShutterStatus = iota
	ShutterClosed
	ShutterOpening
	ShutterClosing
	ShutterError
)

type DomeStatus struct {
	AtHome   bool          `json:"AtHome"`
	AtPark   bool          `json:"AtPark"`
	Slewing  bool          `json:"Slewing"`
	Slaved   bool          `json:"Slaved"`
	Altitude float64       `json:"Altitude"`
	Azimuth  float64       `json:"Azimuth"`
	Shutter  ShutterStatus `json:"ShutterStatus"`
}

func (ds DomeStatus) ToProperties() []StateProperty {
	return []StateProperty{
		{"AtHome", ds.AtHome},
		{"AtPark", ds.AtPark},
		{"Slewing", ds.Slewing},
		{"Slaved", ds.Slaved},
		{"Altitude", ds.Altitude},
		{"Azimuth", ds.Azimuth},
		{"ShutterStatus", ds.Shutter},
	}
}

type ShutterCommand bool

const (
	ShutterCommandOpen  ShutterCommand = true
	ShutterCommandClose ShutterCommand = false
)

type Dome interface {
	Device

	// Dome specific methods
	Capabilities() DomeCapabilities
	Status() DomeStatus
	SetSlaved(bool) error

	SlewToAltitude(float64) error
	SlewToAzimuth(float64) error
	SyncToAzimuth(float64) error
	AbortSlew() error

	FindHome() error
	Park() error
	SetPark() error
	SetShutter(ShutterCommand) error
}

type DomeHandler struct {
	DeviceHandler
	dev Dome
}

func NewDomeHandler(dev Dome) *DomeHandler {
	return &DomeHandler{
		DeviceHandler: DeviceHandler{dev: dev},
		dev:           dev,
	}
}

func (dh *DomeHandler) RegisterRoutes(mux *http.ServeMux) {
	dh.DeviceHandler.RegisterRoutes(mux)

	mux.Handle("GET /altitude", handleAPI(dh.handleStatus))
	mux.Handle("GET /athome", handleAPI(dh.handleStatus))
	mux.Handle("GET /atpark", handleAPI(dh.handleStatus))
	mux.Handle("GET /azimuth", handleAPI(dh.handleStatus))
	mux.Handle("GET /shutterstatus", handleAPI(dh.handleStatus))
	mux.Handle("GET /slewing", handleAPI(dh.handleStatus))

	mux.Handle("GET /canfindhome", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /canpark", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /cansetaltitude", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /cansetazimuth", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /cansetpark", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /cansetshutter", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /canslave", handleAPI(dh.handleCapabilities))
	mux.Handle("GET /cansyncazimuth", handleAPI(dh.handleCapabilities))

	mux.Handle("GET /slaved", handleAPI(func(r *http.Request) (any, error) {
		return dh.dev.Status().Slaved, nil
	}))
	mux.Handle("PUT /slaved", handleAPI(dh.handleSlaved))

	mux.Handle("PUT /slewtoaltitude", handleAPI(dh.handleSlewToAltitude))
	mux.Handle("PUT /slewtoazimuth", handleAPI(dh.handleSlewToAzimuth))
	mux.Handle("PUT /synctoazimuth", handleAPI(dh.handleSyncToAzimuth))
	mux.Handle("PUT /abortslew", handleAPI(dh.handleAbortSlew))
	mux.Handle("PUT /findhome", handleAPI(dh.handleFindHome))
	mux.Handle("PUT /park", handleAPI(dh.handlePark))
	mux.Handle("PUT /setpark", handleAPI(dh.handleSetPark))
	mux.Handle("PUT /openshutter", handleAPI(dh.handleOpenShutter))
	mux.Handle("PUT /closeshutter", handleAPI(dh.handleCloseShutter))
}

func (dh *DomeHandler) handleStatus(r *http.Request) (any, error) {
	status := dh.dev.Status()

	property := r.URL.Path[1:]
	switch property {
	case "altitude":
		return status.Altitude, nil
	case "athome":
		return status.AtHome, nil
	case "atpark":
		return status.AtPark, nil
	case "azimuth":
		return status.Azimuth, nil
	case "shutterstatus":
		return status.Shutter, nil
	case "slewing":
		return status.Slewing, nil
	case "slaved":
		return status.Slaved, nil
	default:
		return nil, errBadRequest
	}
}

func (dh *DomeHandler) handleCapabilities(r *http.Request) (any, error) {
	cap := dh.dev.Capabilities()

	property := r.URL.Path[1:]
	switch property {
	case "canfindhome":
		return cap.CanFindHome, nil
	case "canpark":
		return cap.CanPark, nil
	case "cansetaltitude":
		return cap.CanSetAltitude, nil
	case "cansetazimuth":
		return cap.CanSetAzimuth, nil
	case "cansetpark":
		return cap.CanSetPark, nil
	case "cansetshutter":
		return cap.CanSetShutter, nil
	case "canslave":
		return cap.CanSlave, nil
	case "cansyncazimuth":
		return cap.CanSyncAzimuth, nil
	default:
		return nil, errBadRequest
	}
}

func (dh *DomeHandler) handleSlaved(r *http.Request) (any, error) {
	slaved, err := getBoolParam(r, "Slaved")
	if err != nil {
		return nil, errBadRequest
	}

	if err := dh.dev.SetSlaved(slaved); err != nil {
		return nil, err
	}
	return slaved, nil
}

func (dh *DomeHandler) handleSlewToAltitude(r *http.Request) (any, error) {
	altitude, err := getFloatParam(r, "Altitude")
	if err != nil {
		return nil, errBadRequest
	}

	if err := dh.dev.SlewToAltitude(altitude); err != nil {
		return nil, err
	}
	return nil, nil
}

func (dh *DomeHandler) handleSlewToAzimuth(r *http.Request) (any, error) {
	azimuth, err := getFloatParam(r, "Azimuth")
	if err != nil {
		return nil, errBadRequest
	}

	return nil, dh.dev.SlewToAzimuth(azimuth)
}

func (dh *DomeHandler) handleSyncToAzimuth(r *http.Request) (any, error) {
	azimuth, err := getFloatParam(r, "Azimuth")
	if err != nil {
		return nil, errBadRequest
	}

	return nil, dh.dev.SyncToAzimuth(azimuth)
}

func (dh *DomeHandler) handleAbortSlew(r *http.Request) (any, error) {
	return nil, dh.dev.AbortSlew()
}

func (dh *DomeHandler) handleFindHome(r *http.Request) (any, error) {
	return nil, dh.dev.FindHome()
}

func (dh *DomeHandler) handlePark(r *http.Request) (any, error) {
	return nil, dh.dev.Park()
}

func (dh *DomeHandler) handleSetPark(r *http.Request) (any, error) {
	return nil, dh.dev.SetPark()
}

func (dh *DomeHandler) handleOpenShutter(r *http.Request) (any, error) {
	return nil, dh.dev.SetShutter(ShutterCommandOpen)
}

func (dh *DomeHandler) handleCloseShutter(r *http.Request) (any, error) {
	return nil, dh.dev.SetShutter(ShutterCommandClose)
}
