// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package alpaca

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

type baseResponse struct {
	ClientTransactionID int    `json:"ClientTransactionID"`
	ServerTransactionID int    `json:"ServerTransactionID"`
	ErrorNumber         int    `json:"ErrorNumber"`
	ErrorMessage        string `json:"ErrorMessage"`
	Value               any    `json:"Value,omitempty"`
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

type DeviceHTTPHandler interface {
	RegisterRoutes(mux *http.ServeMux)
}

func (s *Server) AddRoutes() *http.ServeMux {
	r := http.NewServeMux()
	r.HandleFunc("GET /management/apiversions", s.handleAPIVersions)
	r.HandleFunc("GET /management/v1/description", s.handleDescription)
	r.HandleFunc("GET /management/v1/configureddevices", s.handleConfiguredDevices)
	r.HandleFunc("GET /setup", s.handleSetup)

	// Create handlers for each device
	for _, dev := range s.devices {
		mux := http.NewServeMux()
		var handler DeviceHTTPHandler

		switch d := dev.(type) {
		case Dome:
			handler = NewDomeHandler(d)
			handler.RegisterRoutes(mux)
		default:
			log.Errorf("Unknown device type: %T", dev)
			handler = &DeviceHandler{dev: dev}
			handler.RegisterRoutes(mux)
		}

		devType := strings.ToLower(dev.DeviceInfo().Type.String())
		devNumber := dev.DeviceInfo().Number

		prefix := fmt.Sprintf("/api/v1/%s/%d", devType, devNumber)
		r.Handle(prefix+"/", http.StripPrefix(prefix, mux))
	}

	return r
}

func (s *Server) handleAPIVersions(w http.ResponseWriter, r *http.Request) {
	handleResponse(w, []int{1})
}

func (s *Server) handleDescription(w http.ResponseWriter, r *http.Request) {
	handleResponse(w, s.description)
}

func (s *Server) handleConfiguredDevices(w http.ResponseWriter, r *http.Request) {
	deviceInfo := make([]DeviceInfo, 0, len(s.devices))
	for _, device := range s.devices {
		deviceInfo = append(deviceInfo, device.DeviceInfo())
	}

	handleResponse(w, deviceInfo)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement setup user interface
	handleResponse(w, "Not Implemented")
}
