package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPPort            = 8088
	defaultAdminerPort         = 8080
	defaultMySQLHost           = "localhost"
	defaultMySQLPort           = 3306
	defaultMySQLDatabase       = "lunar"
	defaultMySQLUser           = "lunar"
	defaultMySQLPassword       = "lunar"
	defaultDBMaxOpenConns      = 10
	defaultDBMaxIdleConns      = 10
	defaultDBConnMaxLifetime   = time.Hour
	defaultDBConnectRetries    = 30
	defaultDBConnectRetryDelay = time.Second
)

// Config contains the runtime settings shared by the API and local database
// services.
type Config struct {
	HTTPPort    int
	AdminerPort int

	MySQLHost     string
	MySQLPort     int
	MySQLDatabase string
	MySQLUser     string
	MySQLPassword string
	MySQLDSN      string

	DBMaxOpenConns      int
	DBMaxIdleConns      int
	DBConnMaxLifetime   time.Duration
	DBConnectRetries    int
	DBConnectRetryDelay time.Duration
}

// Load reads configuration from environment variables, falling back to the
// values used by docker-compose.yml for local development.
func Load() (Config, error) {
	cfg := Config{
		HTTPPort:            intEnv("HTTP_PORT", defaultHTTPPort),
		AdminerPort:         intEnv("ADMINER_PORT", defaultAdminerPort),
		MySQLHost:           stringEnv("MYSQL_HOST", defaultMySQLHost),
		MySQLPort:           intEnv("MYSQL_PORT", defaultMySQLPort),
		MySQLDatabase:       stringEnv("MYSQL_DATABASE", defaultMySQLDatabase),
		MySQLUser:           stringEnv("MYSQL_USER", defaultMySQLUser),
		MySQLPassword:       stringEnv("MYSQL_PASSWORD", defaultMySQLPassword),
		DBMaxOpenConns:      intEnv("DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns),
		DBMaxIdleConns:      intEnv("DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns),
		DBConnMaxLifetime:   durationEnv("DB_CONN_MAX_LIFETIME", defaultDBConnMaxLifetime),
		DBConnectRetries:    intEnv("DB_CONNECT_RETRIES", defaultDBConnectRetries),
		DBConnectRetryDelay: durationEnv("DB_CONNECT_RETRY_DELAY", defaultDBConnectRetryDelay),
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	cfg.MySQLDSN = fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC",
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLDatabase,
	)
	return cfg, nil
}

func stringEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return -1
	}
	return parsed
}

func validate(cfg Config) error {
	positive := []struct {
		name  string
		value int
	}{
		{"HTTP_PORT", cfg.HTTPPort},
		{"ADMINER_PORT", cfg.AdminerPort},
		{"MYSQL_PORT", cfg.MySQLPort},
		{"DB_MAX_OPEN_CONNS", cfg.DBMaxOpenConns},
		{"DB_MAX_IDLE_CONNS", cfg.DBMaxIdleConns},
		{"DB_CONNECT_RETRIES", cfg.DBConnectRetries},
	}
	for _, field := range positive {
		if field.value < 1 {
			return fmt.Errorf("%s must be a positive integer", field.name)
		}
	}
	if cfg.DBConnMaxLifetime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME must be positive")
	}
	if cfg.DBConnectRetryDelay <= 0 {
		return fmt.Errorf("DB_CONNECT_RETRY_DELAY must be positive")
	}
	return nil
}
