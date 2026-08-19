-- 001_init.sql — Initial schema for Cicada Hunt
-- PostgreSQL migration

-- ================================================================
-- Extension: PostGIS for spatial queries
-- ================================================================
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ================================================================
-- Table: cell_environment
-- Stores pre-computed environmental factors per H3 Lv9 cell
-- ================================================================
CREATE TABLE cell_environment (
    h3_cell_lv9     VARCHAR(16) PRIMARY KEY,
    tree_score      REAL NOT NULL DEFAULT 0.0,
    soil_score      REAL NOT NULL DEFAULT 0.0,
    soil_type       VARCHAR(32) DEFAULT 'unknown',
    elevation_m     REAL DEFAULT 0.0,
    slope_deg       REAL DEFAULT 0.0,
    impervious_pct  REAL DEFAULT 0.0,
    ndvi            REAL DEFAULT 0.0,
    tree_density_idx REAL DEFAULT 0.0,
    is_urban        BOOLEAN DEFAULT FALSE,
    water_proximity_m REAL DEFAULT 500.0,
    dominant_trees  VARCHAR(64) DEFAULT 'unknown',
    base_density    REAL DEFAULT 0.0,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cell_env_tree ON cell_environment(tree_score) WHERE tree_score > 0.3;
CREATE INDEX idx_cell_env_updated ON cell_environment(updated_at);

-- ================================================================
-- Table: nymph_spawns
-- Individual cicada nymph spawn points
-- ================================================================
CREATE TABLE nymph_spawns (
    id              VARCHAR(32) PRIMARY KEY,
    h3_cell_lv9     VARCHAR(16) NOT NULL,
    h3_cell_lv11    VARCHAR(16),
    location        GEOGRAPHY(POINT, 4326),
    lat             DOUBLE PRECISION NOT NULL,
    lng             DOUBLE PRECISION NOT NULL,
    depth_cm        REAL NOT NULL DEFAULT 20.0,

    -- Attributes
    species         VARCHAR(32) NOT NULL DEFAULT 'black_cicada',
    species_name    VARCHAR(64) DEFAULT '',
    size_cm         REAL NOT NULL DEFAULT 3.0,
    weight_g        REAL NOT NULL DEFAULT 5.0,
    quality         SMALLINT DEFAULT 3 CHECK (quality >= 1 AND quality <= 5),
    is_rare         BOOLEAN DEFAULT FALSE,
    estimated_value REAL DEFAULT 0.0,

    -- Lifecycle
    status          VARCHAR(16) DEFAULT 'active' CHECK (status IN ('active', 'consumed', 'expired')),
    consumed_by     VARCHAR(64),
    consumed_at     TIMESTAMPTZ,

    -- Metadata
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    refreshed_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_nymph_cell ON nymph_spawns(h3_cell_lv9) WHERE status = 'active';
CREATE INDEX idx_nymph_location ON nymph_spawns USING GIST(location);
CREATE INDEX idx_nymph_status ON nymph_spawns(status);
CREATE INDEX idx_nymph_consumed_by ON nymph_spawns(consumed_by, consumed_at) WHERE status = 'consumed';
CREATE INDEX idx_nymph_species ON nymph_spawns(species) WHERE status = 'active';

-- ================================================================
-- Table: cicada_spawns
-- Adult cicada spawn points for the catching game mode
-- ================================================================
CREATE TABLE cicada_spawns (
    id              VARCHAR(32) PRIMARY KEY,
    location        GEOGRAPHY(POINT, 4326),
    lat             DOUBLE PRECISION NOT NULL,
    lng             DOUBLE PRECISION NOT NULL,
    altitude_m      REAL DEFAULT 5.0,

    -- Habitat
    tree_id         VARCHAR(64),
    tree_species    VARCHAR(64),
    height_m        REAL DEFAULT 3.0,
    foliage_density REAL DEFAULT 0.5,

    -- Attributes
    species         VARCHAR(32) NOT NULL,
    species_name    VARCHAR(64),
    size_cm         REAL DEFAULT 3.5,
    is_male         BOOLEAN DEFAULT TRUE,
    rarity          SMALLINT DEFAULT 1 CHECK (rarity >= 1 AND rarity <= 5),

    -- Behavior
    current_state   VARCHAR(16) DEFAULT 'resting',
    alert_distance_m  REAL DEFAULT 4.0,
    flee_distance_m   REAL DEFAULT 15.0,
    flight_speed_ms   REAL DEFAULT 3.0,
    agility           REAL DEFAULT 0.3,

    -- Lifecycle
    status          VARCHAR(16) DEFAULT 'active',
    captured_by     VARCHAR(64),
    captured_at     TIMESTAMPTZ,
    startled_until  TIMESTAMPTZ,

    -- Economic
    estimated_value REAL DEFAULT 5.0,

    -- Metadata
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    spawned_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cicada_location ON cicada_spawns USING GIST(location);
CREATE INDEX idx_cicada_status ON cicada_spawns(status) WHERE status = 'active';
CREATE INDEX idx_cicada_rarity ON cicada_spawns(rarity) WHERE status = 'active';

-- ================================================================
-- Table: dig_history
-- Audit log of all digging actions
-- ================================================================
CREATE TABLE dig_history (
    id              BIGSERIAL PRIMARY KEY,
    nymph_id        VARCHAR(32) NOT NULL,
    player_id       VARCHAR(64) NOT NULL,
    location        GEOGRAPHY(POINT, 4326),
    lat             DOUBLE PRECISION,
    lng             DOUBLE PRECISION,
    distance_m      REAL DEFAULT 0.0,
    deviation_cm    REAL DEFAULT 0.0,
    angle_deg       REAL DEFAULT 0.0,
    tool_used       VARCHAR(32) DEFAULT 'bare_hand',
    success         BOOLEAN DEFAULT FALSE,
    success_rate    REAL DEFAULT 0.0,
    dig_time        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dig_player_date ON dig_history(player_id, dig_time);
CREATE INDEX idx_dig_nymph ON dig_history(nymph_id);
CREATE INDEX idx_dig_success ON dig_history(success, dig_time);

-- ================================================================
-- Table: catch_history
-- Audit log of all cicada catching actions
-- ================================================================
CREATE TABLE catch_history (
    id              BIGSERIAL PRIMARY KEY,
    cicada_id       VARCHAR(32) NOT NULL,
    player_id       VARCHAR(64) NOT NULL,
    location        GEOGRAPHY(POINT, 4326),
    lat             DOUBLE PRECISION,
    lng             DOUBLE PRECISION,
    distance_m      REAL DEFAULT 0.0,
    angle_deg       REAL DEFAULT 0.0,
    net_used        VARCHAR(32) DEFAULT 'basic_net',
    swing_speed_ms  REAL DEFAULT 5.0,
    success         BOOLEAN DEFAULT FALSE,
    cicada_evaded   BOOLEAN DEFAULT FALSE,
    catch_time      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_catch_player_date ON catch_history(player_id, catch_time);

-- ================================================================
-- Table: players
-- Player profiles
-- ================================================================
CREATE TABLE players (
    id              VARCHAR(64) PRIMARY KEY,
    nickname        VARCHAR(64) NOT NULL DEFAULT '',
    level           INT DEFAULT 1,
    exp             BIGINT DEFAULT 0,
    gold_coins      BIGINT DEFAULT 0,
    diamonds        INT DEFAULT 0,

    -- Stats
    total_digs      INT DEFAULT 0,
    total_nymphs    INT DEFAULT 0,
    total_catches   INT DEFAULT 0,
    rare_nymphs     INT DEFAULT 0,
    legendary_captures INT DEFAULT 0,

    -- Equipment
    current_shovel  VARCHAR(32) DEFAULT 'bare_hand',
    current_net     VARCHAR(32) DEFAULT 'basic_net',
    current_radar   VARCHAR(32) DEFAULT 'basic_radar',

    -- Metadata
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ================================================================
-- Table: player_inventory
-- Items owned by players
-- ================================================================
CREATE TABLE player_inventory (
    id              BIGSERIAL PRIMARY KEY,
    player_id       VARCHAR(64) NOT NULL,
    item_type       VARCHAR(32) NOT NULL,
    item_id         VARCHAR(64) NOT NULL,
    quantity        INT DEFAULT 1,
    acquired_at     TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(player_id, item_id)
);

CREATE INDEX idx_inventory_player ON player_inventory(player_id);
