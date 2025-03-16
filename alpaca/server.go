// Documentation: https://ascom-standards.org/api/?urls.primaryName=ASCOM+Alpaca+Management+API

package alpaca

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
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

	db   *store
	tmpl *template.Template
}

// NewServer creates a new ManagementServer instance.
func NewServer(description ServerDescription, devices []Device, db *store, tmpl *template.Template) *Server {
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
	r.HandleFunc("GET /management/apiversions", s.handleAPIVersions)
	r.HandleFunc("GET /management/v1/description", s.handleDescription)
	r.HandleFunc("GET /management/v1/configureddevices", s.handleConfiguredDevices)
	r.HandleFunc("/setup", s.handleSetup)

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

		apiPrefix := fmt.Sprintf("/api/v1/%s/%d", devType, devNumber)
		r.Handle(apiPrefix+"/", http.StripPrefix(apiPrefix, mux))

		setupPrefix := fmt.Sprintf("/setup/v1/%s/%d", devType, devNumber)
		r.Handle(setupPrefix+"/", http.StripPrefix(setupPrefix, mux))
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
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.db.GetMQTTConfig()
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

		log.Infof("Setting MQTT config: %+v", cfg)
		if err := s.db.SetMQTTConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.renderSetupForm(w, cfg, true, "")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

	}
}

func (s *Server) renderSetupForm(w http.ResponseWriter, cfg MQTTConfig, success bool, err string) {
	data := struct {
		MQTTConfig
		Success bool
		Error   string
	}{cfg, success, err}

	if err := s.tmpl.ExecuteTemplate(w, "setup.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseSetupForm(r *http.Request) (MQTTConfig, error) {
	if err := r.ParseForm(); err != nil {
		return MQTTConfig{}, fmt.Errorf("error parsing form: %v", err)
	}

	port := r.FormValue("port")
	intPort, err := strconv.Atoi(port)
	if err != nil {
		return MQTTConfig{}, fmt.Errorf("invalid port: %v", err)
	}

	return MQTTConfig{
		Host:     r.FormValue("host"),
		Port:     intPort,
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}, nil
}
