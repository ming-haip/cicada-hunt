// Package config provides application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server
	ServerPort int
	Env        string // "development", "staging", "production"

	// Database
	DatabaseURL string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// External APIs
	WeatherAPIKey  string
	WeatherBaseURL string

	// Game Settings
	MaxDailyDigs      int64
	MaxNymphsPerCell  int
	MaxQueryRadiusM   float64
	MaxResultsPerQuery int

	// Rate Limiting
	RateLimitPerMin int

	// Cache TTLs
	CellNymphsTTL time.Duration
	DensityTTL    time.Duration
	CooldownTTL   time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort:         getEnvInt("SERVER_PORT", 8080),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://localhost:5432/cicada_hunt?sslmode=disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		WeatherAPIKey:      getEnv("WEATHER_API_KEY", ""),
		WeatherBaseURL:     getEnv("WEATHER_BASE_URL", "https://api.openweathermap.org/data/2.5"),
		MaxDailyDigs:       int64(getEnvInt("MAX_DAILY_DIGS", 50)),
		MaxNymphsPerCell:   getEnvInt("MAX_NYMPHS_PER_CELL", 50),
		MaxQueryRadiusM:    getEnvFloat("MAX_QUERY_RADIUS_M", 500),
		MaxResultsPerQuery: getEnvInt("MAX_RESULTS_PER_QUERY", 50),
		RateLimitPerMin:    getEnvInt("RATE_LIMIT_PER_MIN", 60),
		CellNymphsTTL:      getEnvDuration("CELL_NYMPHS_TTL", 10*time.Minute),
		DensityTTL:         getEnvDuration("DENSITY_TTL", 1*time.Hour),
		CooldownTTL:        getEnvDuration("COOLDOWN_TTL", 5*time.Minute),
	}
}

// Addr returns the server listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.ServerPort)
}

// IsProduction returns whether the environment is production.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// Helpers

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
