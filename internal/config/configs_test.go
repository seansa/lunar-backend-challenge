package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadUsesLocalDefaults(t *testing.T) {
	for _, key := range []string{
		"HTTP_PORT", "ADMINER_PORT", "MYSQL_HOST", "MYSQL_PORT",
		"MYSQL_DATABASE", "MYSQL_USER", "MYSQL_PASSWORD",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
		"DB_CONNECT_RETRIES", "DB_CONNECT_RETRY_DELAY",
	} {
		value, exists := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPPort != 8088 || cfg.AdminerPort != 8080 {
		t.Fatalf("unexpected ports: %+v", cfg)
	}
	if cfg.MySQLDSN != "lunar:lunar@tcp(localhost:3306)/lunar?parseTime=true&loc=UTC" {
		t.Fatalf("unexpected DSN: %q", cfg.MySQLDSN)
	}
	if cfg.DBConnMaxLifetime != time.Hour || cfg.DBConnectRetryDelay != time.Second {
		t.Fatalf("unexpected database timings: %+v", cfg)
	}
	if cfg.ConsumerWorkers != 1 || cfg.ConsumerQueueSize != 100 ||
		cfg.ConsumerProcessTimeout != 10*time.Second {
		t.Fatalf("unexpected consumer pool defaults: %+v", cfg)
	}
}

func TestLoadRejectsInvalidInteger(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-port")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid port error")
	}
}

func TestLoadRejectsNegativeConsumerQueueSize(t *testing.T) {
	t.Setenv("CONSUMER_QUEUE_SIZE", "-1")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid queue size error")
	}
}
