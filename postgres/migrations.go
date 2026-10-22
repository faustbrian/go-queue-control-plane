// Package postgres provides command/desired-state transactions and tenant audit
// history for explicit control-plane composition. NewRuntime borrows a
// go-postgres/v3 pool. The deploying application applies the hand-maintained
// embedded MigrationSource with its own runner before use. Constructors
// do not install/verify schema or own pool shutdown. Driver diagnostics may be
// wrapped in Go errors; the administrative HTTP layer selects public codes.
package postgres

import (
	"embed"

	migrations "github.com/faustbrian/go-migrations/v4"
)

// migrationFiles is hand-maintained schema history, not generated output.
// Embedded bytes and migration authority must be changed through their owner.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// MigrationSource returns the immutable embedded control-plane schema history.
func MigrationSource() (migrations.Source, error) {
	return migrations.NewFSSource(embeddedMigrationSource{}, "migrations")
}
