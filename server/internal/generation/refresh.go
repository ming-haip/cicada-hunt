package generation

import (
	"context"
	"log"
	"time"
)

// RefreshManager handles the daily recovery of depleted nymph populations.
// It simulates natural ecosystem recovery: each day at 3am, cells recover
// 15% of the gap between current density and base density.
type RefreshManager struct {
	densityCalc *DensityCalculator
	ticker      *time.Ticker
	stopCh      chan struct{}
}

// NewRefreshManager creates a new refresh manager.
func NewRefreshManager(dc *DensityCalculator) *RefreshManager {
	return &RefreshManager{
		densityCalc: dc,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the periodic refresh cycle.
// It runs once immediately, then every hour.
func (rm *RefreshManager) Start(ctx context.Context) {
	log.Println("[RefreshManager] Starting periodic nymph population refresh")

	// Run once on startup
	rm.runRefresh(ctx)

	// Then every hour
	rm.ticker = time.NewTicker(1 * time.Hour)

	go func() {
		for {
			select {
			case <-rm.ticker.C:
				rm.runRefresh(ctx)
			case <-rm.stopCh:
				rm.ticker.Stop()
				log.Println("[RefreshManager] Stopped")
				return
			case <-ctx.Done():
				rm.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the periodic refresh.
func (rm *RefreshManager) Stop() {
	close(rm.stopCh)
}

// runRefresh executes one refresh cycle.
// Only cells needing refresh (density < base) are processed.
func (rm *RefreshManager) runRefresh(ctx context.Context) {
	now := time.Now()

	// Only run full refresh at 3am; hourly runs are lighter
	isFullRefresh := now.Hour() == 3

	log.Printf("[RefreshManager] Running refresh cycle (full=%v) at %s", isFullRefresh, now.Format(time.RFC3339))

	// In production:
	// 1. Query all active cells from database
	// 2. For each cell, calculate current vs base density
	// 3. If depleted, recover 15% of the gap
	// 4. Generate new spawn points for recovered density
	// 5. Purge expired nymphs (status=consumed, age > 7 days)

	_ = ctx
	_ = isFullRefresh
}

// RecoveryRate is the daily density recovery ratio.
const RecoveryRate = 0.15 // 15% per day

// CalculateRecoveredDensity computes the new density after one recovery cycle.
func CalculateRecoveredDensity(currDensity, baseDensity float64) float64 {
	if currDensity >= baseDensity {
		return baseDensity
	}

	gap := baseDensity - currDensity
	recovery := gap * RecoveryRate

	return currDensity + recovery
}
