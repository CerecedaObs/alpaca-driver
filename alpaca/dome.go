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

	mux.HandleFunc("GET /altitude", dh.handleStatus)
	mux.HandleFunc("GET /athome", dh.handleStatus)
	mux.HandleFunc("GET /atpark", dh.handleStatus)
	mux.HandleFunc("GET /azimuth", dh.handleStatus)
	mux.HandleFunc("GET /shutterstatus", dh.handleStatus)
	mux.HandleFunc("GET /slewing", dh.handleStatus)
	mux.HandleFunc("GET /slaved", dh.handleStatus)

	mux.HandleFunc("GET /canfindhome", dh.handleCapabilities)
	mux.HandleFunc("GET /canpark", dh.handleCapabilities)
	mux.HandleFunc("GET /cansetaltitude", dh.handleCapabilities)
	mux.HandleFunc("GET /cansetazimuth", dh.handleCapabilities)
	mux.HandleFunc("GET /cansetpark", dh.handleCapabilities)
	mux.HandleFunc("GET /cansetshutter", dh.handleCapabilities)
	mux.HandleFunc("GET /canslave", dh.handleCapabilities)
	mux.HandleFunc("GET /cansyncazimuth", dh.handleCapabilities)

	mux.HandleFunc("PUT /slewtoaltitude", dh.handleSlewToAltitude)
	mux.HandleFunc("PUT /slewtoazimuth", dh.handleSlewToAzimuth)
	mux.HandleFunc("PUT /synctoazimuth", dh.handleSyncToAzimuth)
	mux.HandleFunc("PUT /abortslew", dh.handleAbortSlew)
	mux.HandleFunc("PUT /findhome", dh.handleFindHome)
	mux.HandleFunc("PUT /park", dh.handlePark)
	mux.HandleFunc("PUT /setpark", dh.handleSetPark)
	mux.HandleFunc("PUT /openshutter", dh.handleOpenShutter)
	mux.HandleFunc("PUT /closeshutter", dh.handleCloseShutter)
}

func (dh *DomeHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := dh.dev.Status()

	property := r.URL.Path[1:]
	switch property {
	case "altitude":
		handleResponse(w, r, status.Altitude)
	case "athome":
		handleResponse(w, r, status.AtHome)
	case "atpark":
		handleResponse(w, r, status.AtPark)
	case "azimuth":
		handleResponse(w, r, status.Azimuth)
	case "shutterstatus":
		handleResponse(w, r, status.Shutter)
	case "slewing":
		handleResponse(w, r, status.Slewing)
	case "slaved":
		handleResponse(w, r, status.Slaved)
	default:
		handleError(w, r, http.StatusNotFound, "Property not found")
	}
}

func (dh *DomeHandler) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	cap := dh.dev.Capabilities()

	property := r.URL.Path[1:]
	switch property {
	case "canfindhome":
		handleResponse(w, r, cap.CanFindHome)
	case "canpark":
		handleResponse(w, r, cap.CanPark)
	case "cansetaltitude":
		handleResponse(w, r, cap.CanSetAltitude)
	case "cansetazimuth":
		handleResponse(w, r, cap.CanSetAzimuth)
	case "cansetpark":
		handleResponse(w, r, cap.CanSetPark)
	case "cansetshutter":
		handleResponse(w, r, cap.CanSetShutter)
	case "canslave":
		handleResponse(w, r, cap.CanSlave)
	case "cansyncazimuth":
		handleResponse(w, r, cap.CanSyncAzimuth)
	default:
		handleError(w, r, http.StatusNotFound, "Property not found")
	}
}

func (dh *DomeHandler) handleSlewToAltitude(w http.ResponseWriter, r *http.Request) {
	altitude, err := parseFloatRequest(r, "Altitude")
	if err != nil {
		handleError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SlewToAltitude(altitude); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleSlewToAzimuth(w http.ResponseWriter, r *http.Request) {
	azimuth, err := parseFloatRequest(r, "Azimuth")
	if err != nil {
		handleError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SlewToAzimuth(azimuth); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleSyncToAzimuth(w http.ResponseWriter, r *http.Request) {
	azimuth, err := parseFloatRequest(r, "Azimuth")
	if err != nil {
		handleError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SyncToAzimuth(azimuth); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleAbortSlew(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.AbortSlew(); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleFindHome(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.FindHome(); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handlePark(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.Park(); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleSetPark(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.SetPark(); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleOpenShutter(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.SetShutter(ShutterCommandOpen); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}

func (dh *DomeHandler) handleCloseShutter(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.SetShutter(ShutterCommandClose); err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, r, nil)
}
