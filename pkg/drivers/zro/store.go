package zro

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	bucket                = "alpaca"
	defaultHomePosition   = 0
	defaultParkPosition   = 90
	defaultShutterTimeout = 60
	defaultTicksPerRev    = 1470

	domeConfigKey = "zro_config"
)

type MQTTConfig struct {
	Host      string
	Username  string
	Password  string
	TopicRoot string // Root topic for the ZRO dome controller
}

type Config struct {
	MQTTConfig

	TicksPerTurn   int     // Encoder ticks per dome revolution
	Tolerance      int     // Tolerance in encoder ticks
	HomePosition   float64 // Home position in degrees
	ParkPosition   float64 // Park position in degrees
	AzimuthTimeout int     // Azimuth timeout in seconds
	MaxSpeed       int     // Maximum speed in encoder ticks per second
	MinSpeed       int     // Minimum speed in encoder ticks per second
	BrakeSpeed     int     // Brake speed in encoder ticks per second
	VelTimeout     int     // Velocity timeout in seconds
	ShortDistance  int     // Short distance in encoder ticks
	ParkOnShutter  bool    // True if the dome should park on shutter
	ShutterTimeout int     // Shutter timeout in seconds
	UseShutter     bool    // True if the shutter is used
}

var defaultConfig = Config{
	MQTTConfig: MQTTConfig{
		Host:      "tcp://localhost:1883",
		Username:  "",
		Password:  "",
		TopicRoot: "/ZRO",
	},
	TicksPerTurn:   10476,
	Tolerance:      4,
	HomePosition:   0,
	ParkPosition:   0,
	AzimuthTimeout: 20000,
	MaxSpeed:       200,
	MinSpeed:       30,
	BrakeSpeed:     80,
	VelTimeout:     10,
	ShortDistance:  100,
	ParkOnShutter:  false,
	ShutterTimeout: 0,
	UseShutter:     true,
}

type store struct {
	db *bolt.DB
}

// NewStore creates a new store instance and sets default values if they are not already set.
func NewStore(db *bolt.DB) (*store, error) {
	st := store{db: db}

	if err := st.setDefaults(); err != nil {
		return nil, err
	}
	return &st, nil
}

// setDefaults sets the default configuration values if they are not already set in the database.
func (s *store) setDefaults() error {
	if _, err := s.GetDomeConfig(); err != nil {
		log.Infof("Setting default MQTT config")
		s.SetDomeConfig(defaultConfig)
	}

	return nil
}

// SetDomeConfig saves the dome configuration as a json string in the database.
func (s *store) SetDomeConfig(cfg Config) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		value, _ := json.Marshal(cfg)
		return b.Put([]byte(domeConfigKey), value)
	})
}

// GetDomeConfig retrieves the dome configuration from the database.
func (s *store) GetDomeConfig() (Config, error) {
	var cfg Config

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		value := b.Get([]byte(domeConfigKey))
		if value == nil {
			return fmt.Errorf("key config not found")
		}

		return json.Unmarshal(value, &cfg)
	})

	return cfg, err
}
