// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package alpaca

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

const templateDir = "templates"

type ServerDescription struct {
	Name                string `json:"ServerName"`
	Manufacturer        string `json:"Manufacturer"`
	ManufacturerVersion string `json:"ManufacturerVersion"`
	Location            string `json:"Location"`
}

// Server is an Alpaca management server that provides information
// about the server and the devices it manages.
type Server struct {
	description ServerDescription
	devices     []Device

	tmpl *template.Template
}

// NewServer creates a new ManagementServer instance.
func NewServer(description ServerDescription, devices []Device) *Server {
	tmpl, err := template.ParseGlob(templateDir + "/*.html")
	if err != nil {
		log.Fatalf("Error loading setup template: %v", err)
	}

	server := Server{
		description: description,
		devices:     devices,
		tmpl:        tmpl,
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
	r.HandleFunc("GET /setup/v1/dome/0/setup", s.handleDomeSetup)

	// Create handlers for each device
	for _, dev := range s.devices {
		mux := http.NewServeMux()
		var handler DeviceHTTPHandler

		switch d := dev.(type) {
		case Dome:
			logger := log.WithField("device", d.DeviceInfo().Name)
			handler = NewDomeHandler(d, logger)
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
	handleResponse(w, r, []int{1})
}

func (s *Server) handleDescription(w http.ResponseWriter, r *http.Request) {
	handleResponse(w, r, s.description)
}

func (s *Server) handleConfiguredDevices(w http.ResponseWriter, r *http.Request) {
	deviceInfo := make([]DeviceInfo, 0, len(s.devices))
	for _, device := range s.devices {
		deviceInfo = append(deviceInfo, device.DeviceInfo())
	}

	handleResponse(w, r, deviceInfo)
}

// handleSetup returns a user interface for setting up the server.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	// Use the pre-parsed /home/jme/go/alpaca-driver/alpaca/templates/setup.html")
	err := s.tmpl.ExecuteTemplate(w, "setup.html", s)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		log.Errorf("Error rendering template: %v", err)
	}
}

func (s *Server) handleDomeSetup(w http.ResponseWriter, r *http.Request) {
	err := s.tmpl.ExecuteTemplate(w, "dome_setup.html", s)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		log.Errorf("Error rendering template: %v", err)
	}
}
