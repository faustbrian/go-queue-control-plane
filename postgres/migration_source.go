package postgres

import (
	"context"
	"errors"
	"io"
	"io/fs"

	migrations "github.com/faustbrian/go-migrations/v2"
)

// embeddedMigrationSource serves only compiler-owned immutable files. It checks
// producer budgets before retaining directory metadata or copying file bytes;
// no caller filesystem, blocking reader, or background worker is involved.
type embeddedMigrationSource struct{}

func (embeddedMigrationSource) ReadDir(ctx context.Context, root string, limits migrations.SourceDirectoryLimits) ([]migrations.SourceEntry, error) {
	return readEmbeddedMigrationDirectory(ctx, migrationFiles, root, limits)
}

func readEmbeddedMigrationDirectory(ctx context.Context, files fs.FS, root string, limits migrations.SourceDirectoryLimits) ([]migrations.SourceEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limits.MaxEntries < 0 || limits.MaxNameBytes < 0 || limits.MaxTotalNameBytes < 0 {
		return nil, migrations.ErrSourceLimit
	}
	directory, err := files.Open(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()
	reader, ok := directory.(fs.ReadDirFile)
	if !ok {
		return nil, migrations.ErrInvalidSource
	}
	var entries []migrations.SourceEntry
	remaining := limits.MaxTotalNameBytes
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, err := reader.ReadDir(1)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		for _, entry := range batch {
			name := entry.Name()
			if len(entries) >= limits.MaxEntries || len(name) > limits.MaxNameBytes || len(name) > remaining {
				return nil, migrations.ErrSourceLimit
			}
			remaining -= len(name)
			entries = append(entries, migrations.SourceEntry{Name: name, Directory: entry.IsDir()})
		}
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
	}
}

func (embeddedMigrationSource) ReadFile(ctx context.Context, name string, maxBytes int) ([]byte, error) {
	return readEmbeddedMigrationFile(ctx, migrationFiles, name, maxBytes)
}

func readEmbeddedMigrationFile(ctx context.Context, files fs.ReadFileFS, name string, maxBytes int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxBytes < 0 {
		return nil, migrations.ErrInvalidEncoding
	}
	file, err := files.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() || info.Size() > int64(maxBytes) {
		return nil, migrations.ErrInvalidEncoding
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	contents, err := files.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return contents, nil
}
