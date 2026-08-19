package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cicada-hunt/server/internal/models"
)

// CacheManager provides Redis-based caching for nymph data and cell densities.
type CacheManager struct {
	redis RedisClient
}

// RedisClient abstracts Redis operations for testability.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// Cache key patterns
const (
	keyCellNymphs    = "cell:nymphs:%s"      // H3 cell → JSON []NymphSpawn
	keyCellDensity   = "cell:density:%s"     // H3 cell → float64 density
	keyCellCooldown  = "gen:cooldown:%s"     // H3 cell → cooldown flag
	keyPlayerDaily   = "player:daily:%s:%s"  // playerID:date → count
	keyNymphDetail   = "nymph:%s"            // nymphID → JSON NymphSpawn
)

const (
	defaultCellTTL  = 10 * time.Minute
	defaultNymphTTL = 24 * time.Hour
	dailyTTL        = 24 * time.Hour
	cooldownTTL     = 5 * time.Minute
)

// NewCacheManager creates a new cache manager.
func NewCacheManager(redis RedisClient) *CacheManager {
	return &CacheManager{redis: redis}
}

// GetCellNymphs retrieves cached nymphs for a cell.
func (cm *CacheManager) GetCellNymphs(ctx context.Context, cellID string) ([]*models.NymphSpawn, error) {
	key := fmt.Sprintf(keyCellNymphs, cellID)
	data, err := cm.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	if data == "" {
		return nil, fmt.Errorf("cache miss")
	}

	var nymphs []*models.NymphSpawn
	if err := json.Unmarshal([]byte(data), &nymphs); err != nil {
		return nil, fmt.Errorf("unmarshal nymphs: %w", err)
	}

	return nymphs, nil
}

// SetCellNymphs caches nymphs for a cell.
func (cm *CacheManager) SetCellNymphs(ctx context.Context, cellID string, nymphs []*models.NymphSpawn, ttl time.Duration) error {
	if ttl == 0 {
		ttl = defaultCellTTL
	}

	data, err := json.Marshal(nymphs)
	if err != nil {
		return fmt.Errorf("marshal nymphs: %w", err)
	}

	key := fmt.Sprintf(keyCellNymphs, cellID)
	return cm.redis.Set(ctx, key, string(data), ttl)
}

// GetCellDensity retrieves cached density for a cell.
func (cm *CacheManager) GetCellDensity(ctx context.Context, cellID string) (float64, error) {
	key := fmt.Sprintf(keyCellDensity, cellID)
	data, err := cm.redis.Get(ctx, key)
	if err != nil || data == "" {
		return 0, fmt.Errorf("cache miss")
	}

	var density float64
	if _, err := fmt.Sscanf(data, "%f", &density); err != nil {
		return 0, fmt.Errorf("parse density: %w", err)
	}

	return density, nil
}

// SetCellDensity caches the current density for a cell.
func (cm *CacheManager) SetCellDensity(ctx context.Context, cellID string, density float64, ttl time.Duration) error {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}

	key := fmt.Sprintf(keyCellDensity, cellID)
	return cm.redis.Set(ctx, key, fmt.Sprintf("%.2f", density), ttl)
}

// IncrPlayerDailyDigs increments and returns a player's daily digging count.
func (cm *CacheManager) IncrPlayerDailyDigs(ctx context.Context, playerID string) (int64, error) {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf(keyPlayerDaily, playerID, today)

	count, err := cm.redis.Incr(ctx, key)
	if err != nil {
		return 0, err
	}

	// Ensure key expires at end of day
	cm.redis.Expire(ctx, key, dailyTTL)

	return count, nil
}

// GetPlayerDailyDigs returns the player's current daily digging count.
func (cm *CacheManager) GetPlayerDailyDigs(ctx context.Context, playerID string) (int64, error) {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf(keyPlayerDaily, playerID, today)

	data, err := cm.redis.Get(ctx, key)
	if err != nil || data == "" {
		return 0, nil
	}

	var count int64
	if _, err := fmt.Sscanf(data, "%d", &count); err != nil {
		return 0, nil
	}

	return count, nil
}

// SetGenCooldown marks a cell as recently generated to prevent stampedes.
func (cm *CacheManager) SetGenCooldown(ctx context.Context, cellID string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = cooldownTTL
	}
	key := fmt.Sprintf(keyCellCooldown, cellID)
	return cm.redis.Set(ctx, key, "1", ttl)
}

// CheckGenCooldown checks if a cell is still on generation cooldown.
func (cm *CacheManager) CheckGenCooldown(ctx context.Context, cellID string) (bool, error) {
	key := fmt.Sprintf(keyCellCooldown, cellID)
	exists, err := cm.redis.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// GetNymphDetail retrieves a cached individual nymph.
func (cm *CacheManager) GetNymphDetail(ctx context.Context, nymphID string) (*models.NymphSpawn, error) {
	key := fmt.Sprintf(keyNymphDetail, nymphID)
	data, err := cm.redis.Get(ctx, key)
	if err != nil || data == "" {
		return nil, fmt.Errorf("cache miss")
	}

	var nymph models.NymphSpawn
	if err := json.Unmarshal([]byte(data), &nymph); err != nil {
		return nil, fmt.Errorf("unmarshal nymph: %w", err)
	}

	return &nymph, nil
}

// SetNymphDetail caches an individual nymph.
func (cm *CacheManager) SetNymphDetail(ctx context.Context, nymph *models.NymphSpawn, ttl time.Duration) error {
	if ttl == 0 {
		ttl = defaultNymphTTL
	}

	data, err := json.Marshal(nymph)
	if err != nil {
		return fmt.Errorf("marshal nymph: %w", err)
	}

	key := fmt.Sprintf(keyNymphDetail, nymph.ID)
	return cm.redis.Set(ctx, key, string(data), ttl)
}
