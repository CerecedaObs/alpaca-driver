package alpaca

import (
	"net/http"
)

type DeviceType string

const (
	DeviceTypeCamera      DeviceType = "Camera"
	DeviceTypeCover       DeviceType = "CoverCalibrator"
	DeviceTypeDome        DeviceType = "Dome"
	DeviceTypeFilterWheel DeviceType = "FilterWheel"
	DeviceTypeFocuser     DeviceType = "Focuser"
	DeviceTypeRotator     DeviceType = "Rotator"
	DeviceTypeSafety      DeviceType = "SafetyMonitor"
	DeviceTypeSwitch      DeviceType = "Switch"
	DeviceTypeTelescope   DeviceType = "Telescope"
)

func (dt DeviceType) String() string {
	return string(dt)
}

type DeviceInfo struct {
	Name        string     `json:"DeviceName"`
	Description string     `json:"-"`
	Type        DeviceType `json:"DeviceType"`
	UniqueID    string     `json:"UniqueID"`

	// TODO: Number should be generated by the server
	Number int `json:"DeviceNumber"`
}

type DriverInfo struct {
	Name             string
	Version          string
	InterfaceVersion int
}

type StateProperty struct {
	Name  string
	Value any
}

type Device interface {
	DeviceInfo() DeviceInfo
	DriverInfo() DriverInfo
	GetState() []StateProperty

	Connected() bool
	Connecting() bool
	Connect() error
	Disconnect() error

	HandleSetup(http.ResponseWriter, *http.Request)
}

type DeviceHandler struct {
	dev Device
}

func (h *DeviceHandler) RegisterRoutes(mux *http.ServeMux) {
	// mux.HandleFunc("GET /setup", h.handleSetup)
	mux.Handle("GET /name", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.DeviceInfo().Name, nil
	}))
	mux.Handle("GET /description", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.DeviceInfo().Description, nil
	}))
	mux.Handle("GET /driverinfo", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.DriverInfo().Name, nil
	}))
	mux.Handle("GET /driverversion", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.DriverInfo().Version, nil
	}))
	mux.Handle("GET /interfaceversion", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.DriverInfo().InterfaceVersion, nil
	}))
	mux.Handle("GET /devicestate", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.GetState(), nil
	}))
	mux.Handle("GET /supportedactions", handleAPI(func(r *http.Request) (any, error) {
		return []string{}, nil
	}))
	mux.Handle("GET /connecting", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.Connecting(), nil
	}))
	mux.Handle("GET /connected", handleAPI(func(r *http.Request) (any, error) {
		return h.dev.Connected(), nil
	}))

	mux.Handle("PUT /connected", handleAPI(h.putConnected))
	mux.Handle("PUT /connect", handleAPI(h.handleConnect))
	mux.Handle("PUT /disconnect", handleAPI(h.handleDisconnect))

	mux.HandleFunc("/setup", h.dev.HandleSetup)
}

func (h *DeviceHandler) putConnected(r *http.Request) (any, error) {
	connected, err := getBoolParam(r, "Connected")
	if err != nil {
		return nil, errBadRequest
	}

	if connected {
		return connected, h.dev.Connect()
	}
	return connected, h.dev.Disconnect()
}

func (h *DeviceHandler) handleConnect(r *http.Request) (any, error) {
	if err := h.dev.Connect(); err != nil {
		return nil, err
	}
	return true, nil
}

func (h *DeviceHandler) handleDisconnect(r *http.Request) (any, error) {
	if err := h.dev.Disconnect(); err != nil {
		return nil, err
	}
	return true, nil
}
