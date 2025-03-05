// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package alpaca

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type baseResponse struct {
	ClientTransactionID int         `json:"ClientTransactionID"`
	ServerTransactionID int         `json:"ServerTransactionID"`
	ErrorNumber         int         `json:"ErrorNumber"`
	ErrorMessage        string      `json:"ErrorMessage"`
	Value               interface{} `json:"Value"`
}

type ServerDescription struct {
	Name                string `json:"ServerName"`
	Manufacturer        string `json:"Manufacturer"`
	ManufacturerVersion string `json:"ManufacturerVersion"`
	Location            string `json:"Location"`
}

// Global transaction counter
var txCounter atomic.Int32

// Server is an Alpaca management server that provides information
// about the server and the devices it manages.
type Server struct {
	description ServerDescription
	devices     []Device
}

// NewServer creates a new ManagementServer instance.
func NewServer(description ServerDescription, devices []Device) *Server {
	server := Server{
		description: description,
		devices:     devices,
	}

	return &server
}

func (s *Server) AddRoutes() *http.ServeMux {
	r := http.NewServeMux()
	r.HandleFunc("GET /management/apiversions", s.handleAPIVersions)
	r.HandleFunc("GET /management/v1/description", s.handleDescription)
	r.HandleFunc("GET /management/v1/configureddevices", s.handleConfiguredDevices)
	r.HandleFunc("GET /setup", s.handleSetup)

	// Add device specific routes
	for _, dev := range s.devices {
		handler := NewDeviceHandler(dev)
		mux := handler.RegisterRoutes()

		devType := strings.ToLower(handler.dev.DeviceInfo().Type)
		prefix := fmt.Sprintf("/api/v1/%s/%d", devType, handler.dev.DeviceInfo().Number)
		r.Handle(prefix+"/", http.StripPrefix(prefix, mux))
	}

	return r
}

func (s *Server) handleResponse(w http.ResponseWriter, value interface{}) {
	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		Value:               value,
	}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIVersions(w http.ResponseWriter, r *http.Request) {
	s.handleResponse(w, []int{1})
}

func (s *Server) handleDescription(w http.ResponseWriter, r *http.Request) {
	s.handleResponse(w, s.description)
}

func (s *Server) handleConfiguredDevices(w http.ResponseWriter, r *http.Request) {
	deviceInfo := make([]DeviceInfo, 0, len(s.devices))
	for _, device := range s.devices {
		deviceInfo = append(deviceInfo, device.DeviceInfo())
	}

	s.handleResponse(w, deviceInfo)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement setup user interface
	s.handleResponse(w, "Not Implemented")
}
