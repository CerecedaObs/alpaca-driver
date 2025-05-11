// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package alpaca

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

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

	db   *Store
	tmpl *template.Template
}

// NewServer creates a new ManagementServer instance.
func NewServer(description ServerDescription, devices []Device, db *Store, tmpl *template.Template) *Server {
	server := Server{
		description: description,
		devices:     devices,
		db:          db,
		tmpl:        tmpl,
	}

	return &server
}

type DeviceHTTPHandler interface {
	RegisterRoutes(mux *http.ServeMux)
}

func (s *Server) AddRoutes() *http.ServeMux {
	r := http.NewServeMux()

	// Add management routes
	r.Handle("GET /management/apiversions", handleMgm(s.handleAPIVersions))
	r.Handle("GET /management/v1/description", handleMgm(s.handleDescription))
	r.Handle("GET /management/v1/configureddevices", handleMgm(s.handleConfiguredDevices))
	r.HandleFunc("/setup", s.handleSetup)

	// Create handlers for each device
	for _, dev := range s.devices {
		mux := http.NewServeMux()
		var handler DeviceHTTPHandler

		switch d := dev.(type) {
		case Dome:
			log.Infof("Creating new DomeHandler for %s", dev.DeviceInfo().Name)
			handler = NewDomeHandler(d)
			handler.RegisterRoutes(mux)
		default:
			log.Errorf("Unknown device type: %T", dev)
			handler = &DeviceHandler{dev: dev}
			handler.RegisterRoutes(mux)
		}

		devType := strings.ToLower(dev.DeviceInfo().Type.String())
		devNumber := dev.DeviceInfo().Number

		apiPrefix := fmt.Sprintf("/api/v1/%s/%d", devType, devNumber)
		r.Handle(apiPrefix+"/", http.StripPrefix(apiPrefix, mux))

		setupPrefix := fmt.Sprintf("/setup/v1/%s/%d", devType, devNumber)
		r.Handle(setupPrefix+"/", http.StripPrefix(setupPrefix, mux))
	}

	return r
}

func (s *Server) handleAPIVersions(r *http.Request) (any, error) {
	return []int{1}, nil
}

func (s *Server) handleDescription(r *http.Request) (any, error) {
	return s.description, nil
}

func (s *Server) handleConfiguredDevices(r *http.Request) (any, error) {
	deviceInfo := make([]DeviceInfo, 0, len(s.devices))
	for _, device := range s.devices {
		deviceInfo = append(deviceInfo, device.DeviceInfo())
	}

	return deviceInfo, nil
}

// handleSetup returns a user interface for setting up the server.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.db.GetConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.renderSetupForm(w, cfg, false, "")

	case http.MethodPost:
		cfg, err := parseSetupForm(r)
		if err != nil {
			s.renderSetupForm(w, cfg, false, err.Error())
			return
		}

		log.Infof("Setting config: %+v", cfg)
		if err := s.db.SetConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.renderSetupForm(w, cfg, true, "")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

	}
}

func (s *Server) renderSetupForm(w http.ResponseWriter, cfg Config, success bool, err string) {
	data := struct {
		Config
		Success bool
		Error   string
	}{cfg, success, err}

	if err := s.tmpl.ExecuteTemplate(w, "setup.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseSetupForm(r *http.Request) (Config, error) {
	if err := r.ParseForm(); err != nil {
		return Config{}, fmt.Errorf("error parsing form: %v", err)
	}

	return Config{}, nil
}
