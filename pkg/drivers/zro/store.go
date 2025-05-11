package zro

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	bucket    = "alpaca"
	configKey = "zro_config"
)

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
	if _, err := s.GetConfig(); err != nil {
		log.Infof("Setting default MQTT config")
		s.SetConfig(defaultConfig)
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
		return b.Put([]byte(configKey), value)
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

		value := b.Get([]byte(configKey))
		if value == nil {
			return fmt.Errorf("key config not found")
		}

		return json.Unmarshal(value, &cfg)
	})

	return cfg, err
}
