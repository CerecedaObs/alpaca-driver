package alpaca

import (
	"net/http"

	log "github.com/sirupsen/logrus"
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

	mux.HandleFunc("/altitude", dh.handleStatus)
	mux.HandleFunc("/athome", dh.handleStatus)
	mux.HandleFunc("/atpark", dh.handleStatus)
	mux.HandleFunc("/azimuth", dh.handleStatus)
	mux.HandleFunc("/shutterstatus", dh.handleStatus)
	mux.HandleFunc("/slewing", dh.handleStatus)
	mux.HandleFunc("/slaved", dh.handleStatus)

	mux.HandleFunc("/canfindhome", dh.handleCapabilities)
	mux.HandleFunc("/canpark", dh.handleCapabilities)
	mux.HandleFunc("/cansetaltitude", dh.handleCapabilities)
	mux.HandleFunc("/cansetazimuth", dh.handleCapabilities)
	mux.HandleFunc("/cansetpark", dh.handleCapabilities)
	mux.HandleFunc("/cansetshutter", dh.handleCapabilities)
	mux.HandleFunc("/canslave", dh.handleCapabilities)
	mux.HandleFunc("/cansyncazimuth", dh.handleCapabilities)

	mux.HandleFunc("/slewtoaltitude", dh.handleSlewToAltitude)
	mux.HandleFunc("/slewtoazimuth", dh.handleSlewToAzimuth)
	mux.HandleFunc("/synctoazimuth", dh.handleSyncToAzimuth)
	mux.HandleFunc("/abortslew", dh.handleAbortSlew)
	mux.HandleFunc("/findhome", dh.handleFindHome)
	mux.HandleFunc("/park", dh.handlePark)
	mux.HandleFunc("/setpark", dh.handleSetPark)
	mux.HandleFunc("/setshutter", dh.handleSetShutter)
}

func (dh *DomeHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	property := r.URL.Path[1:]
	log.Debugf("Dome property: %s", property)

	status := dh.dev.Status()

	switch property {
	case "altitude":
		handleResponse(w, status.Altitude)
	case "athome":
		handleResponse(w, status.AtHome)
	case "atpark":
		handleResponse(w, status.AtPark)
	case "azimuth":
		handleResponse(w, status.Azimuth)
	case "shutterstatus":
		handleResponse(w, status.Shutter)
	case "slewing":
		handleResponse(w, status.Slewing)
	case "slaved":
		handleResponse(w, status.Slaved)
	default:
		handleError(w, http.StatusNotFound, "Property not found")
	}
}

func (dh *DomeHandler) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	property := r.URL.Path[1:]
	log.Debugf("Dome property: %s", property)

	cap := dh.dev.Capabilities()

	switch property {
	case "canfindhome":
		handleResponse(w, cap.CanFindHome)
	case "canpark":
		handleResponse(w, cap.CanPark)
	case "cansetaltitude":
		handleResponse(w, cap.CanSetAltitude)
	case "cansetazimuth":
		handleResponse(w, cap.CanSetAzimuth)
	case "cansetpark":
		handleResponse(w, cap.CanSetPark)
	case "cansetshutter":
		handleResponse(w, cap.CanSetShutter)
	case "canslave":
		handleResponse(w, cap.CanSlave)
	case "cansyncazimuth":
		handleResponse(w, cap.CanSyncAzimuth)
	default:
		handleError(w, http.StatusNotFound, "Property not found")
	}
}

func (dh *DomeHandler) handleSlewToAltitude(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Altitude float64 `json:"altitude"`
	}
	if err := parseRequest(r, &req); err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SlewToAltitude(req.Altitude); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handleSlewToAzimuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Azimuth float64 `json:"azimuth"`
	}
	if err := parseRequest(r, &req); err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SlewToAzimuth(req.Azimuth); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handleSyncToAzimuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Azimuth float64 `json:"azimuth"`
	}
	if err := parseRequest(r, &req); err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SyncToAzimuth(req.Azimuth); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handleAbortSlew(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.AbortSlew(); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handleFindHome(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.FindHome(); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handlePark(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.Park(); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handleSetPark(w http.ResponseWriter, r *http.Request) {
	if err := dh.dev.SetPark(); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}

func (dh *DomeHandler) handleSetShutter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command ShutterCommand `json:"command"`
	}
	if err := parseRequest(r, &req); err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := dh.dev.SetShutter(req.Command); err != nil {
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(w, nil)
}
