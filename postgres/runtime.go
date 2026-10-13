package postgres

import (
	"context"
	"errors"

	gopostgres "github.com/faustbrian/go-postgres"
	gopostgresv2 "github.com/faustbrian/go-postgres/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidRuntimePool reports a missing or uninitialized PostgreSQL pool.
var ErrInvalidRuntimePool = errors.New("postgres: invalid runtime pool")

// Runtime is the complete PostgreSQL-backed control-plane service bundle.
// The caller retains ownership of the underlying pool.
type Runtime struct {
	// Journal coordinates command, desired-state and audit writes against the
	// borrowed pool.
	Journal *Journal
	// Audit provides tenant audit reads, verification and retention.
	Audit *AuditStore
	// Commands provides scoped command lookup, pagination and retention.
	Commands *CommandStore
	// Desired reads latest durable desired state; it does not drive worker
	// convergence.
	Desired *DesiredStore
	// Readiness pings the borrowed pool through its configured health-check
	// budget.
	Readiness *PoolReadiness
}

// PoolReadiness adapts postgres bounded health checks to the API contract.
type PoolReadiness struct {
	pool interface{ Ping(context.Context) error }
}

// Ready performs a bounded PostgreSQL ping through postgres.
func (readiness *PoolReadiness) Ready(ctx context.Context) error {
	return readiness.pool.Ping(ctx)
}

// NewRuntime wires all control-plane persistence services to a legacy v1 pool.
// The caller retains ownership; use NewRuntimeWithPool for go-postgres/v2.
func NewRuntime(pool *gopostgres.Pool) (*Runtime, error) {
	if pool == nil || pool.Raw() == nil {
		return nil, ErrInvalidRuntimePool
	}
	return newRuntime(pool.Raw(), pool)
}

// NewRuntimeWithPool wires persistence services to a go-postgres/v2 pool.
// It refuses nil or uninitialized pools. The caller retains pool ownership;
// transactions use the native pgxpool, and readiness uses the pool's bounded Ping.
func NewRuntimeWithPool(pool *gopostgresv2.Pool) (*Runtime, error) {
	if pool == nil || pool.Raw() == nil {
		return nil, ErrInvalidRuntimePool
	}

	return newRuntime(pool.Raw(), pool)
}

func newRuntime(raw *pgxpool.Pool, pool interface{ Ping(context.Context) error }) (*Runtime, error) {
	return &Runtime{
		Journal:   newJournal(newPostgresTransactionRunner(raw)),
		Audit:     &AuditStore{beginner: raw},
		Commands:  &CommandStore{beginner: raw},
		Desired:   &DesiredStore{queryer: raw},
		Readiness: &PoolReadiness{pool: pool},
	}, nil
}
