package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/newrelic/go-agent/v3/integrations/nrpq" // Registers "nrpostgres" driver
	"github.com/newrelic/go-agent/v3/newrelic"

	"ride/internal/config"
)

// NewDatabase creates a new PostgreSQL connection with optimized settings.
// If nrApp is provided, it uses New Relic instrumented driver for automatic SQL tracing.
func NewDatabase(ctx context.Context, cfg config.DatabaseConfig, nrApp *newrelic.Application) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	var db *sql.DB
	var err error

	// Use New Relic instrumented driver if New Relic is enabled
	// The "nrpostgres" driver is automatically registered by the nrpq import
	if nrApp != nil {
		db, err = sql.Open("nrpostgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open database with nrpq: %w", err)
		}
	} else {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
	}

	// ============================================
	// OPTIMIZED CONNECTION POOL SETTINGS
	// ============================================
	//
	// MaxOpenConns: Maximum number of open connections to the database.
	// - Set based on: (CPU cores * 2) + effective_spindle_count
	// - For cloud DBs, typically 50-100 is a good starting point
	// - Too high can overwhelm the DB; too low causes connection wait
	db.SetMaxOpenConns(50)

	// MaxIdleConns: Maximum number of idle connections.
	// - Should be less than or equal to MaxOpenConns
	// - Set to ~25% of MaxOpenConns for typical workloads
	// - Higher values reduce connection establishment overhead
	db.SetMaxIdleConns(25)

	// ConnMaxLifetime: Maximum time a connection can be reused.
	// - Helps rotate connections and handle DB failovers
	// - Should be less than any firewall/proxy timeout
	// - 30 minutes is a good default
	db.SetConnMaxLifetime(30 * time.Minute)

	// ConnMaxIdleTime: Maximum time a connection can be idle before being closed.
	// - Prevents stale connections from accumulating
	// - 5 minutes is a good default
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection.
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
