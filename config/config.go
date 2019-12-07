package config

import (
	"os"
	"sync"

	registry "github.com/gecosys/registry-go/config/registry"
)

var once sync.Once
var instance *Config

// Config is content of config.json
type Config struct {
	Registry registry.Config `json:"registry"`
}

// Get returns object Config (Singleton)
func Get() *Config {
	if instance != nil {
		return instance
	}

	once.Do(func() {
		instance = new(Config)
		instance.Registry.Address = os.Getenv("SERVER_REGISTRY_ADDRESS")
	})
	return instance
}
