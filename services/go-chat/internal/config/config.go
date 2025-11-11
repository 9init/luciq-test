package config

import (
	"fmt"
	"os"
	"path"
	"syscall"
)

type Config struct {
	AppName          string
	ListenAddr       string
	ListenPort       int
	LogPath          string
	RedisURL         string
	AmqpURL          string
	MySqlDsn         string
	ElasticsearchURL string
}

func NewConfig() (*Config, error) {
	// Build MySQL DSN from individual env vars (same as Rails uses)
	dbHost := getEnv("DB_HOST", "localhost")
	dbUsername := getEnv("DB_USERNAME", "root")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "rails_api_development")
	dbPort := getEnv("DB_PORT", "3306")

	mysqlDsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUsername, dbPassword, dbHost, dbPort, dbName)

	return &Config{
		AppName:          getEnv("APP_NAME", path.Base(os.Args[0])),
		ListenAddr:       getEnv("LISTEN_ADDR", "localhost"),
		ListenPort:       atoiEnv("LISTEN_PORT", 8080),
		LogPath:          getEnv("LOG_DIR", "logs"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AmqpURL:          getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
		MySqlDsn:         mysqlDsn,
		ElasticsearchURL: getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
	}, nil
}

func atoiEnv(key string, defaultValue int) int {
	if valueStr, exists := lookupEnv(key); exists {
		var value int
		_, err := fmt.Sscanf(valueStr, "%d", &value)
		if err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value, exists := lookupEnv(key); exists {
		return value
	}
	return defaultValue
}

var lookupEnv = func(key string) (string, bool) {
	envValue, exists := syscall.Getenv(key)
	return envValue, exists
}
