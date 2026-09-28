package main

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
)

const maxTLSMaterialBytes = 1 << 20

var errTLSMaterial = errors.New("queue-control-plane: invalid TLS material")

// tlsMaterialReader admits only bounded regular files for the owned OTLP runtime.
type tlsMaterialReader struct {
	stat func(*os.File) (os.FileInfo, error)
}

func (reader tlsMaterialReader) ReadFile(ctx context.Context, path string, maxBytes int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxBytes <= 0 || maxBytes > maxTLSMaterialBytes {
		return nil, errTLSMaterial
	}

	// Nonblocking open prevents a configured pipe or device path from hanging
	// startup before the file can be rejected as non-regular.
	// #nosec G304 -- Operator-owned local TLS path; type and size checked below.
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errTLSMaterial
	}
	defer func() { _ = file.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = file.Close() })
	defer stop()

	stat := reader.stat
	if stat == nil {
		stat = (*os.File).Stat
	}
	info, err := stat(file)
	if contextErr := ctx.Err(); contextErr != nil {
		return nil, contextErr
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > int64(maxBytes) {
		return nil, errTLSMaterial
	}
	contents, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || len(contents) > maxBytes {
		return nil, errTLSMaterial
	}
	return contents, nil
}
