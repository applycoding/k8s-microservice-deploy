package config

import (
	"net/url"
	"testing"
)

func TestDSNRoundTrip(t *testing.T) {
	d := Database{
		Host:     "db.internal",
		Port:     "5432",
		Name:     "trips",
		User:     "trips",
		Password: "p@ss word",
		SSLMode:  "disable",
	}
	u, err := url.Parse(d.DSN())
	if err != nil {
		t.Fatal(err)
	}
	if u.Hostname() != "db.internal" || u.Port() != "5432" {
		t.Fatalf("host = %s", u.Host)
	}
	if u.Path != "/trips" {
		t.Fatalf("path = %s", u.Path)
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	if user != "trips" || pass != "p@ss word" {
		t.Fatalf("userinfo = %s / %s", user, pass)
	}
	if u.Query().Get("sslmode") != "disable" {
		t.Fatalf("sslmode = %s", u.Query().Get("sslmode"))
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_HOST", "postgres")
	t.Setenv("DATABASE_PORT", "5433")
	t.Setenv("DATABASE_NAME", "fleet")
	t.Setenv("DATABASE_USER", "app")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_SSLMODE", "require")

	cfg := FromEnv()
	if cfg.Addr != ":9090" || cfg.LogLevel != "debug" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Database.Host != "postgres" || cfg.Database.Port != "5433" || cfg.Database.Name != "fleet" {
		t.Fatalf("database = %+v", cfg.Database)
	}
	if cfg.Database.User != "app" || cfg.Database.Password != "secret" || cfg.Database.SSLMode != "require" {
		t.Fatalf("database auth = %+v", cfg.Database)
	}
}
