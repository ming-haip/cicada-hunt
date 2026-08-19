package environment

import (
	"context"
	"fmt"
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

// MockWeatherClient implements WeatherClient with configurable mock data.
// In production, replace with real API client (OpenWeather / 和风天气).
type MockWeatherClient struct {
	DefaultTemp     float64
	DefaultHumidity float64
	DefaultRaining  bool
}

// NewMockWeatherClient creates a mock weather client with sensible defaults.
func NewMockWeatherClient() *MockWeatherClient {
	return &MockWeatherClient{
		DefaultTemp:     28.0,
		DefaultHumidity: 65.0,
		DefaultRaining:  false,
	}
}

// GetCurrentWeather returns the current weather context.
// In production, calls an external weather API.
func (c *MockWeatherClient) GetCurrentWeather(cellID string) *models.WeatherContext {
	return &models.WeatherContext{
		TemperatureC:  c.DefaultTemp,
		HumidityPct:   c.DefaultHumidity,
		RainLast24hMm: 0,
		IsRaining:     c.DefaultRaining,
		WindSpeedMS:   3.0,
		WeatherFactor: 1.0,
	}
}

// RealWeatherClient fetches live weather data from an external API.
type RealWeatherClient struct {
	apiKey      string
	baseURL     string
	httpTimeout time.Duration
}

// NewRealWeatherClient creates a real weather API client.
func NewRealWeatherClient(apiKey, baseURL string) *RealWeatherClient {
	return &RealWeatherClient{
		apiKey:      apiKey,
		baseURL:     baseURL,
		httpTimeout: 5 * time.Second,
	}
}

// GetCurrentWeather fetches real weather data.
// TODO: Implement actual HTTP call to OpenWeather / 和风天气 API.
func (c *RealWeatherClient) GetCurrentWeather(cellID string) *models.WeatherContext {
	ctx, cancel := context.WithTimeout(context.Background(), c.httpTimeout)
	defer cancel()

	_ = ctx
	_ = fmt.Sprintf // placeholder

	// Real implementation:
	// 1. Convert cellID to lat/lng
	// 2. Call weather API
	// 3. Parse response into WeatherContext
	// 4. Handle rate limits and caching

	// For now, return default data
	return &models.WeatherContext{
		TemperatureC:  25.0,
		HumidityPct:   60.0,
		RainLast24hMm: 0,
		IsRaining:     false,
		WindSpeedMS:   3.0,
		WeatherFactor: 1.0,
	}
}
