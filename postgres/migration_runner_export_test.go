package postgres

import (
	"context"
	"database/sql"

	migrations "github.com/faustbrian/go-migrations/v4"
	migrationpostgres "github.com/faustbrian/go-migrations/v4/postgres"
	gopostgres "github.com/faustbrian/go-postgres/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewMigrationRunner composes the embedded source with a go-migrations v4
// runner the way a deploying application does, for database tests only.
func NewMigrationRunner(database *sql.DB) (*migrations.Runner, error) {
	source, err := MigrationSource()
	if err != nil {
		return nil, err
	}
	backend, err := migrationpostgres.New(database)
	if err != nil {
		return nil, err
	}

	return migrations.NewRunner(source, backend)
}

// DatabaseTestPoolConfig is the database-test pool configuration: go-postgres v3
// requires an explicit resolver, and tests want startup connectivity checked.
func DatabaseTestPoolConfig(dsn string) gopostgres.Config {
	return gopostgres.Config{
		DSN:           dsn,
		StartupPolicy: gopostgres.StartupPing,
		ResolveDSN: func(ctx context.Context, dsn string) (*gopostgres.PoolConfig, error) {
			if ctx.Err() != nil {
				return nil, context.Cause(ctx)
			}
			return pgxpool.ParseConfig(dsn)
		},
	}
}
