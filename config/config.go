package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Env string

const (
	EnvTest Env = "test"
	EnvDev  Env = "dev"
)

type Config struct {
	APIServerHost    string `env:"API_SERVER_HOST"`
	APIServerPort    string `env:"API_SERVER_PORT"`
	DatabaseName     string `env:"DB_NAME"`
	DatabaseHost     string `env:"DB_HOST"`
	DatabasePort     string `env:"DB_PORT"`
	DatabaseTestPort string `env:"TEST_DB_PORT"`
	DatabaseUser     string `env:"DB_USER"`
	DatabasePassword string `env:"DB_PASSWORD"`
	Env              Env    `env:"ENV" envDefault:"dev"`
}

func New() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &cfg, err
}

func (c *Config) DatabaseURL() string {
	port := c.DatabasePort
	if c.Env == EnvTest {
		port = c.DatabaseTestPort
	}

	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseHost,
		port,
		c.DatabaseName,
	)
}
