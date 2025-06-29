package config

import (
	"os"
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port string
	}
	Database struct {
		URL string
	}
	Redis struct {
		URL string
	}
	NATS struct {
		URL string
	}
	Alchemyst struct {
		APIKey  string
		BaseURL string
	}
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	var config Config

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.url", "postgres://admin:password@localhost:5432/arch_search?sslmode=disable")
	viper.SetDefault("redis.url", "redis://localhost:6379")
	viper.SetDefault("nats.url", "nats://localhost:4222")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	config.Server.Port = viper.GetString("server.port")
	config.Database.URL = viper.GetString("database.url")
	config.Redis.URL = viper.GetString("redis.url")
	config.NATS.URL = viper.GetString("nats.url")
	config.Alchemyst.APIKey = os.Getenv("ALCHEMYST_API_KEY")
	config.Alchemyst.BaseURL = os.Getenv("ALCHEMYST_BASE_URL")

	return &config, nil
}

func (c *Config) ValidateAlchemyst() error {
	if c.Alchemyst.APIKey == "" {
		return fmt.Errorf("ALCHEMYST_API_KEY is required")
	}
	if c.Alchemyst.BaseURL == "" {
		return fmt.Errorf("ALCHEMYST_BASE_URL is required")
	}
	return nil
}