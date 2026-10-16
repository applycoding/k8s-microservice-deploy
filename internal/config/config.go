package config

import (
	"net"
	"net/url"
	"os"
)

// Config is process configuration loaded from the environment.
type Config struct {
	Addr     string
	LogLevel string
	Database Database
}

// Database holds Postgres connection settings.
type Database struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// DSN returns a URL Postgres drivers accept.
func (d Database) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   net.JoinHostPort(d.Host, d.Port),
		Path:   "/" + d.Name,
	}
	q := u.Query()
	if d.SSLMode != "" {
		q.Set("sslmode", d.SSLMode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// FromEnv reads configuration. Missing values fall back to local demo defaults.
func FromEnv() Config {
	return Config{
		Addr:     env("ADDR", ":8080"),
		LogLevel: env("LOG_LEVEL", "info"),
		Database: Database{
			Host:     env("DATABASE_HOST", "localhost"),
			Port:     env("DATABASE_PORT", "5432"),
			Name:     env("DATABASE_NAME", "trips"),
			User:     env("DATABASE_USER", "trips"),
			Password: env("DATABASE_PASSWORD", "trips"),
			SSLMode:  env("DATABASE_SSLMODE", "disable"),
		},
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
