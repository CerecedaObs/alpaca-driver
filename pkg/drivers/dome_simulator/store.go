package dome_simulator

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

	domeConfigKey = "dome_config"
)

type Config struct {
	HomePosition   uint `json:"home_position"`   // degrees
	ParkPosition   uint `json:"park_position"`   // degrees
	ShutterTimeout uint `json:"shutter_timeout"` // seconds
	TicksPerRev    uint `json:"ticks_per_rev"`   // encoder ticks per revolution
}

type store struct {
	db *bolt.DB
}

func NewStore(db *bolt.DB) (*store, error) {
	st := store{db: db}

	if err := st.setDefaults(); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *store) setDefaults() error {
	if _, err := s.GetConfig(); err != nil {
		log.Infof("Setting default MQTT config")
		s.SetConfig(Config{
			HomePosition:   defaultHomePosition,
			ParkPosition:   defaultParkPosition,
			ShutterTimeout: defaultShutterTimeout,
			TicksPerRev:    defaultTicksPerRev,
		})
	}

	return nil
}

// SetConfig saves the dome configuration as a json string in the database.
func (s *store) SetConfig(cfg Config) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		value, _ := json.Marshal(cfg)
		return b.Put([]byte(domeConfigKey), value)
	})
}

// GetConfig retrieves the dome configuration from the database.
func (s *store) GetConfig() (Config, error) {
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
