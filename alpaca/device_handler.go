package alpaca

import (
	"encoding/json"
	"net/http"
)

type DeviceHandler struct {
	dev Device
}

func NewDeviceHandler(dev Device) *DeviceHandler {
	return &DeviceHandler{dev}
}

func (h *DeviceHandler) RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// mux.HandleFunc("GET /setup", h.handleSetup)
	mux.HandleFunc("GET /name", h.handleName)
	mux.HandleFunc("GET /description", h.handleDescription)
	mux.HandleFunc("GET /driverinfo", h.handleDriverInfo)
	mux.HandleFunc("GET /driverversion", h.handleDriverVersion)
	mux.HandleFunc("GET /interfaceversion", h.handleInterfaceVersion)
	mux.HandleFunc("GET /devicestate", h.handleState)

	mux.HandleFunc("GET /connected", h.handleConnected)
	mux.HandleFunc("GET /connecting", h.handleConnecting)
	mux.HandleFunc("PUT /connect", h.handleConnect)
	mux.HandleFunc("PUT /disconnect", h.handleDisconnect)

	return mux
}

func (h *DeviceHandler) handleResponse(w http.ResponseWriter, value interface{}) {
	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		Value:               value,
	}
	json.NewEncoder(w).Encode(response)
}

func (h *DeviceHandler) handleError(w http.ResponseWriter, code int, message string) {
	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		ErrorNumber:         code,
		ErrorMessage:        message,
	}
	json.NewEncoder(w).Encode(response)
}

func (h *DeviceHandler) handleName(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.DeviceInfo().Name)
}

func (h *DeviceHandler) handleDescription(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.DeviceInfo().Description)
}

func (h *DeviceHandler) handleDriverInfo(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.DriverInfo())
}

func (h *DeviceHandler) handleDriverVersion(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.DriverInfo().Version)
}

func (h *DeviceHandler) handleInterfaceVersion(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.DriverInfo().InterfaceVersion)
}

func (h *DeviceHandler) handleState(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.GetState())
}

func (h *DeviceHandler) handleConnected(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.Connected())
}

func (h *DeviceHandler) handleConnecting(w http.ResponseWriter, r *http.Request) {
	h.handleResponse(w, h.dev.Connecting())
}

func (h *DeviceHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	if err := h.dev.Connect(); err != nil {
		h.handleError(w, 500, err.Error())
		return
	}
	h.handleResponse(w, true)
}

func (h *DeviceHandler) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.dev.Disconnect(); err != nil {
		h.handleError(w, 500, err.Error())
		return
	}
	h.handleResponse(w, true)
}
