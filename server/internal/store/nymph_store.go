// Package store implements data access for PostgreSQL and Redis.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/cicada-hunt/server/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NymphStore implements the service.NymphStore interface with PostgreSQL.
type NymphStore struct {
	pool *pgxpool.Pool
}

// NewNymphStore creates a new nymph store backed by PostgreSQL.
func NewNymphStore(pool *pgxpool.Pool) *NymphStore {
	return &NymphStore{pool: pool}
}

// GetNymphsByCell retrieves all active nymphs in a given H3 cell.
func (s *NymphStore) GetNymphsByCell(ctx context.Context, cellID string) ([]*models.NymphSpawn, error) {
	query := `
		SELECT id, lat, lng, depth_cm, h3_cell_lv9, h3_cell_lv11,
		       species, species_name, size_cm, weight_g, quality, is_rare,
		       estimated_value, status, consumed_by, consumed_at,
		       created_at, refreshed_at
		FROM nymph_spawns
		WHERE h3_cell_lv9 = $1 AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, cellID)
	if err != nil {
		return nil, fmt.Errorf("query nymphs by cell: %w", err)
	}
	defer rows.Close()

	return scanNymphs(rows)
}

// SaveNymphs batch-inserts nymph spawn records.
func (s *NymphStore) SaveNymphs(ctx context.Context, nymphs []*models.NymphSpawn) error {
	if len(nymphs) == 0 {
		return nil
	}

	// Use COPY protocol for batch insert
	batch := &pgx.Batch{}

	query := `
		INSERT INTO nymph_spawns (
			id, lat, lng, depth_cm, h3_cell_lv9, h3_cell_lv11,
			species, species_name, size_cm, weight_g, quality, is_rare,
			estimated_value, status, location, created_at, refreshed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14,
			ST_SetSRID(ST_MakePoint($3, $2), 4326),
			$15, $16
		)
		ON CONFLICT (id) DO NOTHING
	`

	for _, n := range nymphs {
		batch.Queue(query,
			n.ID, n.Lat, n.Lng, n.DepthCm, n.H3CellLv9, n.H3CellLv11,
			string(n.Species), n.SpeciesName, n.SizeCm, n.WeightG, n.Quality, n.IsRare,
			n.ValueEst, string(n.Status),
			n.CreatedAt, n.RefreshedAt,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	// Execute all inserts
	for i := 0; i < len(nymphs); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert nymph %d: %w", i, err)
		}
	}

	return nil
}

// MarkNymphConsumed marks a nymph as consumed by a player.
func (s *NymphStore) MarkNymphConsumed(ctx context.Context, nymphID, playerID string) error {
	now := time.Now()

	query := `
		UPDATE nymph_spawns
		SET status = 'consumed',
		    consumed_by = $2,
		    consumed_at = $3
		WHERE id = $1 AND status = 'active'
	`

	tag, err := s.pool.Exec(ctx, query, nymphID, playerID, now)
	if err != nil {
		return fmt.Errorf("mark nymph consumed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("nymph %s not found or already consumed", nymphID)
	}

	return nil
}

// GetNymphByID retrieves a single nymph by its ID.
func (s *NymphStore) GetNymphByID(ctx context.Context, nymphID string) (*models.NymphSpawn, error) {
	query := `
		SELECT id, lat, lng, depth_cm, h3_cell_lv9, h3_cell_lv11,
		       species, species_name, size_cm, weight_g, quality, is_rare,
		       estimated_value, status, COALESCE(consumed_by, ''), consumed_at,
		       created_at, refreshed_at
		FROM nymph_spawns
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, query, nymphID)
	return scanNymph(row)
}

// scanNymphs converts a pgx Rows result into NymphSpawn slices.
func scanNymphs(rows pgx.Rows) ([]*models.NymphSpawn, error) {
	var nymphs []*models.NymphSpawn

	for rows.Next() {
		n, err := scanNymph(rows)
		if err != nil {
			return nil, err
		}
		nymphs = append(nymphs, n)
	}

	return nymphs, nil
}

// scanNymph converts a single pgx Row into a NymphSpawn.
func scanNymph(row pgx.Row) (*models.NymphSpawn, error) {
	var n models.NymphSpawn
	var species, status string
	var consumedBy string
	var consumedAt *time.Time

	err := row.Scan(
		&n.ID, &n.Lat, &n.Lng, &n.DepthCm, &n.H3CellLv9, &n.H3CellLv11,
		&species, &n.SpeciesName, &n.SizeCm, &n.WeightG, &n.Quality, &n.IsRare,
		&n.ValueEst, &status, &consumedBy, &consumedAt,
		&n.CreatedAt, &n.RefreshedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan nymph: %w", err)
	}

	n.Species = models.NymphSpecies(species)
	n.Status = models.NymphStatus(status)
	n.ConsumedBy = consumedBy
	n.ConsumedAt = consumedAt

	return &n, nil
}
