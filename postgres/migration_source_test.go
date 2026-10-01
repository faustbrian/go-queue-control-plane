package postgres

import (
	"bytes"
	"context"
	"errors"
	"path"
	"testing"

	migrations "github.com/faustbrian/go-migrations/v2"
)

func TestEmbeddedMigrationProviderHonorsInventoryBudgets(t *testing.T) {
	provider := embeddedMigrationSource{}
	entries, err := provider.ReadDir(context.Background(), "migrations", migrations.SourceDirectoryLimits{
		MaxEntries: 9, MaxNameBytes: 512, MaxTotalNameBytes: 4096,
	})
	if err != nil || len(entries) != 9 {
		t.Fatalf("complete embedded inventory count=%d error=%v", len(entries), err)
	}
	total, longest := 0, 0
	for _, entry := range entries {
		total += len(entry.Name)
		longest = max(longest, len(entry.Name))
	}
	exact := migrations.SourceDirectoryLimits{MaxEntries: len(entries), MaxNameBytes: longest, MaxTotalNameBytes: total}
	if accepted, err := provider.ReadDir(context.Background(), "migrations", exact); err != nil || len(accepted) != len(entries) {
		t.Fatal("exact inclusive inventory budget rejected")
	}
	for _, limits := range []migrations.SourceDirectoryLimits{
		{MaxEntries: 8, MaxNameBytes: longest, MaxTotalNameBytes: total},
		{MaxEntries: 9, MaxNameBytes: longest - 1, MaxTotalNameBytes: total},
		{MaxEntries: 9, MaxNameBytes: longest, MaxTotalNameBytes: total - 1},
		{MaxEntries: -1, MaxNameBytes: longest, MaxTotalNameBytes: total},
	} {
		if rejected, err := provider.ReadDir(context.Background(), "migrations", limits); !errors.Is(err, migrations.ErrSourceLimit) || rejected != nil {
			t.Fatal("over-budget inventory returned partial metadata or incorrect error")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if rejected, err := provider.ReadDir(ctx, "migrations", exact); !errors.Is(err, context.Canceled) || rejected != nil {
		t.Fatal("canceled inventory returned data or incorrect error")
	}
}

func TestEmbeddedMigrationProviderPreservesBytesWithinBudget(t *testing.T) {
	provider := embeddedMigrationSource{}
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := path.Join("migrations", entry.Name())
		want, err := migrationFiles.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		got, err := provider.ReadFile(context.Background(), name, len(want))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("inclusive file budget did not preserve complete embedded SQL")
		}
		for _, budget := range []int{len(want) - 1, -1} {
			if rejected, err := provider.ReadFile(context.Background(), name, budget); !errors.Is(err, migrations.ErrInvalidEncoding) || rejected != nil {
				t.Fatal("over-budget file returned partial bytes or incorrect error")
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if rejected, err := provider.ReadFile(ctx, name, len(want)); !errors.Is(err, context.Canceled) || rejected != nil {
			t.Fatal("canceled file read returned data or incorrect error")
		}
	}
}
