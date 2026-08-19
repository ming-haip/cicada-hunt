package store

import (
	"context"
	"fmt"
	"time"

	"github.com/cicada-hunt/server/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvironmentStore handles persistence of environmental factor data.
type EnvironmentStore struct {
	pool *pgxpool.Pool
}

// NewEnvironmentStore creates a new environment store.
func NewEnvironmentStore(pool *pgxpool.Pool) *EnvironmentStore {
	return &EnvironmentStore{pool: pool}
}

// GetEnvironmentFactors retrieves the cached environmental factors for an H3 cell.
func (s *EnvironmentStore) GetEnvironmentFactors(ctx context.Context, cellID string) (*models.EnvironmentFactors, error) {
	query := `
		SELECT h3_cell_lv9, tree_score, soil_score, soil_type,
		       elevation_m, slope_deg, impervious_pct, ndvi,
		       tree_density_idx, is_urban, water_proximity_m,
		       dominant_trees, updated_at
		FROM cell_environment
		WHERE h3_cell_lv9 = $1
	`

	row := s.pool.QueryRow(ctx, query, cellID)

	var env models.EnvironmentFactors
	err := row.Scan(
		&env.H3CellLv9, &env.TreeScore, &env.SoilScore,
		&env.SoilType, &env.ElevationM, &env.SlopeDeg,
		&env.ImperviousPct, &env.NDVI, &env.TreeDensityIdx,
		&env.IsUrban, &env.WaterProxM, &env.DominantTrees,
		&env.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get environment factors: %w", err)
	}

	return &env, nil
}

// UpsertEnvironmentFactors inserts or updates environmental factors for a cell.
func (s *EnvironmentStore) UpsertEnvironmentFactors(ctx context.Context, env *models.EnvironmentFactors) error {
	query := `
		INSERT INTO cell_environment (
			h3_cell_lv9, tree_score, soil_score, soil_type,
			elevation_m, slope_deg, impervious_pct, ndvi,
			tree_density_idx, is_urban, water_proximity_m,
			dominant_trees, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13
		)
		ON CONFLICT (h3_cell_lv9) DO UPDATE SET
			tree_score = EXCLUDED.tree_score,
			soil_score = EXCLUDED.soil_score,
			soil_type = EXCLUDED.soil_type,
			ndvi = EXCLUDED.ndvi,
			tree_density_idx = EXCLUDED.tree_density_idx,
			updated_at = EXCLUDED.updated_at
	`

	env.UpdatedAt = time.Now()

	_, err := s.pool.Exec(ctx, query,
		env.H3CellLv9, env.TreeScore, env.SoilScore, env.SoilType,
		env.ElevationM, env.SlopeDeg, env.ImperviousPct, env.NDVI,
		env.TreeDensityIdx, env.IsUrban, env.WaterProxM,
		env.DominantTrees, env.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert environment factors: %w", err)
	}

	return nil
}
