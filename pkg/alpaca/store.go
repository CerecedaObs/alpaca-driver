package alpaca

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	bucket    = "alpaca"
	configKey = "server_config"
)

type Config struct{}

type Store struct {
	db *bolt.DB
}

func NewStore(db *bolt.DB) (*Store, error) {
	st := Store{db: db}

	if err := st.setDefaults(); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) setDefaults() error {
	if _, err := s.GetConfig(); err != nil {
		log.Infof("Setting default config")
		return s.SetConfig(Config{})
	}

	return nil
}

// SetConfig saves the configuration as a json string in the database.
func (s *Store) SetConfig(cfg Config) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		value, _ := json.Marshal(cfg)
		return b.Put([]byte(configKey), value)
	})
}

// GetConfig retrieves the configuration from the database.
func (s *Store) GetConfig() (Config, error) {
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
