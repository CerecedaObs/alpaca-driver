package alpaca

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	bucket          = "alpaca"
	defaultMQTTHost = "localhost"
	defaultMQTTPort = 1883

	mqttConfigKey = "mqtt_config"
)

type MQTTConfig struct {
	Host     string
	Port     int
	Username string
	Password string
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
	if _, err := s.GetMQTTConfig(); err != nil {
		log.Infof("Setting default MQTT config")
		s.SetMQTTConfig(MQTTConfig{
			Host:     defaultMQTTHost,
			Port:     defaultMQTTPort,
			Username: "admin",
		})
	}

	return nil
}

// SetMQTTConfig saves the MQTT configuration as a json string in the database.
func (s *store) SetMQTTConfig(cfg MQTTConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if cfg.Port < 1000 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		value, _ := json.Marshal(cfg)
		return b.Put([]byte(mqttConfigKey), value)
	})
}

// GetMQTTConfig retrieves the MQTT configuration from the database.
func (s *store) GetMQTTConfig() (MQTTConfig, error) {
	var cfg MQTTConfig

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		value := b.Get([]byte(mqttConfigKey))
		if value == nil {
			return fmt.Errorf("key config not found")
		}

		return json.Unmarshal(value, &cfg)
	})

	return cfg, err
}
