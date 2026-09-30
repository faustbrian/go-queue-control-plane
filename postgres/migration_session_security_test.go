package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	controlpostgres "github.com/faustbrian/go-queue-control-plane/v2/postgres"
)

func TestPublicMigrationRunnerDiscardsUncertainPhysicalSession(t *testing.T) {
	for _, test := range []struct {
		name        string
		failAcquire bool
		failUnlock  bool
	}{
		{name: "uncertain acquisition", failAcquire: true},
		{name: "uncertain unlock", failUnlock: true},
		{name: "healthy session"},
	} {
		t.Run(test.name, func(t *testing.T) {
			connector := &migrationSessionConnector{failAcquire: test.failAcquire, failUnlock: test.failUnlock}
			database := sql.OpenDB(connector)
			database.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = database.Close() })
			runner, err := controlpostgres.NewMigrationRunner(database)
			if err != nil {
				t.Fatal("public migration runner construction failed")
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			plan, err := runner.Plan(ctx)
			uncertain := test.failAcquire || test.failUnlock
			if (err != nil) != uncertain {
				t.Fatalf("operation error present=%t, want %t", err != nil, uncertain)
			}
			if !uncertain && len(plan.Steps()) != 9 {
				t.Fatalf("healthy plan has %d steps, want complete embedded history", len(plan.Steps()))
			}
			if err != nil {
				for _, format := range []string{"%v", "%+v", "%#v", "%q"} {
					if strings.Contains(fmt.Sprintf(format, err), "private_migration_marker") {
						t.Errorf("public runner disclosed simulated private driver data in %s", format)
					}
				}
			}
			original := connector.firstConnection(t)
			borrowed, err := database.Conn(ctx)
			if err != nil {
				t.Fatal("caller-owned pool could not serve its next borrower")
			}
			defer func() { _ = borrowed.Close() }()
			var reused, locked bool
			if err := borrowed.Raw(func(raw any) error {
				connection := raw.(*migrationSessionConnection)
				reused = connection == original
				connection.mu.Lock()
				locked = connection.locked
				connection.mu.Unlock()
				return nil
			}); err != nil {
				t.Fatal("next physical borrower inspection failed")
			}
			original.mu.Lock()
			closed := original.closed
			original.mu.Unlock()
			if uncertain && (reused || !closed || locked) {
				t.Errorf("uncertain physical session reused=%t closed=%t next borrower locked=%t", reused, closed, locked)
			}
			if !uncertain && (!reused || closed || locked) {
				t.Errorf("healthy physical session reused=%t closed=%t next borrower locked=%t", reused, closed, locked)
			}
		})
	}
}

// This driver models a PostgreSQL session lock surviving a failed reply and
// ordinary pool return, but disappearing when the physical connection closes.
// The public helper builds the actual producer runner, reads the real embedded
// source, and uses real database/sql pooling; only the driver seam is simulated.
type migrationSessionConnector struct {
	mu          sync.Mutex
	connections []*migrationSessionConnection
	failAcquire bool
	failUnlock  bool
}

func (connector *migrationSessionConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	connection := &migrationSessionConnection{connector: connector}
	connector.mu.Lock()
	connector.connections = append(connector.connections, connection)
	connector.mu.Unlock()
	return connection, nil
}

func (connector *migrationSessionConnector) Driver() driver.Driver { return migrationSessionDriver{} }

func (connector *migrationSessionConnector) firstConnection(t *testing.T) *migrationSessionConnection {
	t.Helper()
	connector.mu.Lock()
	defer connector.mu.Unlock()
	if len(connector.connections) == 0 {
		t.Fatal("public runner never acquired a physical database session")
	}
	return connector.connections[0]
}

type migrationSessionDriver struct{}

func (migrationSessionDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use the test-owned connector")
}

type migrationSessionConnection struct {
	mu        sync.Mutex
	connector *migrationSessionConnector
	locked    bool
	closed    bool
}

func (*migrationSessionConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("context-aware execution required")
}

func (*migrationSessionConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("planning does not begin a transaction")
}

func (connection *migrationSessionConnection) Close() error {
	connection.mu.Lock()
	defer connection.mu.Unlock()
	connection.closed = true
	connection.locked = false
	return nil
}

func (*migrationSessionConnection) ExecContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(query, "CREATE TABLE IF NOT EXISTS public.go_schema_migrations") {
		return nil, errors.New("unexpected planning SQL execution")
	}
	return driver.RowsAffected(0), nil
}

func (connection *migrationSessionConnection) QueryContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	connection.mu.Lock()
	defer connection.mu.Unlock()
	switch {
	case strings.Contains(query, "pg_try_advisory_lock"):
		connection.locked = true
		if connection.connector.failAcquire {
			return &migrationSessionRows{columns: []string{"locked"}, values: []driver.Value{"private_migration_marker"}}, nil
		}
		return &migrationSessionRows{columns: []string{"locked"}, values: []driver.Value{true}}, nil
	case strings.Contains(query, "pg_advisory_unlock"):
		if connection.connector.failUnlock {
			return nil, errors.New("private_migration_marker")
		}
		connection.locked = false
		return &migrationSessionRows{columns: []string{"unlocked"}, values: []driver.Value{true}}, nil
	case strings.HasPrefix(query, "SELECT kind, version, name, checksum,"):
		return &migrationSessionRows{columns: []string{"kind", "version", "name", "checksum", "started_at", "finished_at", "execution_time_ms", "dirty"}}, nil
	default:
		return nil, errors.New("unexpected planning query")
	}
}

type migrationSessionRows struct {
	columns []string
	values  []driver.Value
}

func (rows *migrationSessionRows) Columns() []string { return rows.columns }
func (*migrationSessionRows) Close() error           { return nil }
func (rows *migrationSessionRows) Next(destination []driver.Value) error {
	if rows.values == nil {
		return io.EOF
	}
	copy(destination, rows.values)
	rows.values = nil
	return nil
}
