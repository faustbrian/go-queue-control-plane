package postgres

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path"
	"testing"

	migrations "github.com/faustbrian/go-migrations/v2"
)

func TestEmbeddedMigrationProviderRejectsMissingAndNonFilePaths(t *testing.T) {
	provider := embeddedMigrationSource{}
	limits := migrations.SourceDirectoryLimits{MaxEntries: 9, MaxNameBytes: 512, MaxTotalNameBytes: 4096}
	if entries, err := provider.ReadDir(context.Background(), "migrations/missing.sql", limits); entries != nil || !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("missing embedded directory returned metadata or lost its cause")
	}
	if entries, err := provider.ReadDir(context.Background(), "migrations/00001_control_plane.sql", limits); entries != nil || !errors.Is(err, migrations.ErrInvalidSource) {
		t.Fatal("regular embedded file was accepted as a directory")
	}
	if contents, err := provider.ReadFile(context.Background(), "migrations/missing.sql", 1024); contents != nil || !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("missing embedded file returned bytes or lost its cause")
	}
	if contents, err := provider.ReadFile(context.Background(), "migrations", 1024); contents != nil || !errors.Is(err, migrations.ErrInvalidEncoding) {
		t.Fatal("embedded directory was accepted as SQL bytes")
	}
}

type embeddedProviderCancellationContext struct {
	context.Context
	calls, at int
}

func (ctx *embeddedProviderCancellationContext) Err() error {
	ctx.calls++
	if ctx.calls >= ctx.at {
		return context.Canceled
	}
	return nil
}

func TestEmbeddedMigrationProviderDiscardsCanceledPreparation(t *testing.T) {
	provider := embeddedMigrationSource{}
	limits := migrations.SourceDirectoryLimits{MaxEntries: 9, MaxNameBytes: 512, MaxTotalNameBytes: 4096}
	for _, at := range []int{2, 3} {
		ctx := &embeddedProviderCancellationContext{Context: context.Background(), at: at}
		if entries, err := provider.ReadDir(ctx, "migrations", limits); entries != nil || !errors.Is(err, context.Canceled) {
			t.Fatal("canceled inventory preparation exposed partial metadata")
		}
		ctx = &embeddedProviderCancellationContext{Context: context.Background(), at: at}
		if contents, err := provider.ReadFile(ctx, "migrations/00001_control_plane.sql", 1<<20); contents != nil || !errors.Is(err, context.Canceled) {
			t.Fatal("canceled file preparation exposed SQL bytes")
		}
	}
}

func TestEmbeddedMigrationHelpersPreserveReadFailuresAndCloseFiles(t *testing.T) {
	for _, test := range []struct {
		name        string
		directory   bool
		statFailure bool
	}{
		{name: "directory read", directory: true},
		{name: "file stat", statFailure: true},
		{name: "file read"},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "migrations/00001_control_plane.sql"
			if test.directory {
				name = "migrations"
			}
			opened, err := migrationFiles.Open(name)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = opened.Close() })
			cause := errors.New("ordinary embedded-source read failure")
			failure := &fs.PathError{Op: test.name, Path: name, Err: cause}
			file := &migrationFailureFile{File: opened}
			files := migrationFailureFS{file: file}
			if test.directory {
				file.directoryFailure = failure
			} else if test.statFailure {
				file.statFailure = failure
			} else {
				files.readFailure = failure
			}
			if test.directory {
				entries, err := readEmbeddedMigrationDirectory(context.Background(), files, name, migrations.SourceDirectoryLimits{MaxEntries: 9, MaxNameBytes: 512, MaxTotalNameBytes: 4096})
				if entries != nil || !errors.Is(err, cause) {
					t.Fatal("directory failure lost its cause or returned partial inventory")
				}
				var pathError *fs.PathError
				if !errors.As(err, &pathError) || pathError != failure {
					t.Fatal("directory failure lost its original typed error")
				}
			} else {
				contents, err := readEmbeddedMigrationFile(context.Background(), files, name, 1<<20)
				if contents != nil || !errors.Is(err, cause) {
					t.Fatal("file failure lost its cause or returned partial SQL")
				}
				var pathError *fs.PathError
				if !errors.As(err, &pathError) || pathError != failure {
					t.Fatal("file failure lost its original typed error")
				}
			}
			if !file.closed {
				t.Fatal("failed embedded-source preparation retained an open file")
			}
		})
	}
}

type migrationFailureFS struct {
	file        *migrationFailureFile
	readFailure error
}

func (files migrationFailureFS) Open(string) (fs.File, error)    { return files.file, nil }
func (files migrationFailureFS) ReadFile(string) ([]byte, error) { return nil, files.readFailure }

type migrationFailureFile struct {
	fs.File
	statFailure      error
	directoryFailure error
	closed           bool
}

func (file *migrationFailureFile) Stat() (fs.FileInfo, error) {
	if file.statFailure != nil {
		return nil, file.statFailure
	}
	return file.File.Stat()
}

func (file *migrationFailureFile) ReadDir(int) ([]fs.DirEntry, error) {
	return nil, file.directoryFailure
}

func (file *migrationFailureFile) Close() error {
	file.closed = true
	return file.File.Close()
}

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
