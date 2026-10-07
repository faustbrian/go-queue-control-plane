package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestTLSMaterialReaderRejectsZeroBudgetForEmptyFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "empty.pem")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	reader := tlsMaterialReader{}
	if contents, err := reader.ReadFile(context.Background(), path, 0); contents != nil || err != errTLSMaterial {
		t.Fatalf("zero budget returned contents=%v error=%v", contents != nil, err)
	}
	if contents, err := reader.ReadFile(context.Background(), path, 1); err != nil || len(contents) != 0 {
		t.Fatalf("positive budget rejected an empty file: %v", err)
	}
}

func TestTLSMaterialReaderAcceptsInclusiveGlobalBudget(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bounded.pem")
	want := strings.Repeat("x", maxTLSMaterialBytes)
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	contents, err := (tlsMaterialReader{}).ReadFile(context.Background(), path, maxTLSMaterialBytes)
	if err != nil || string(contents) != want {
		t.Fatalf("inclusive budget returned bytes=%d error=%v", len(contents), err)
	}
}

func TestTLSMaterialReaderRedactsUncancelledStatFailure(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bounded.pem")
	if err := os.WriteFile(path, []byte("material"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := tlsMaterialReader{stat: func(*os.File) (os.FileInfo, error) {
		return nil, errors.New("private stat diagnostic")
	}}
	if contents, err := reader.ReadFile(context.Background(), path, 8); contents != nil || err != errTLSMaterial {
		t.Fatalf("stat failure returned contents=%v error=%v", contents != nil, err)
	}
}

func TestTLSMaterialReaderRequestsNonblockingReadOnlyOpen(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bounded.pem")
	if err := os.WriteFile(path, []byte("material"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := tlsMaterialReader{openFile: func(gotPath string, flags int, mode os.FileMode) (*os.File, error) {
		// Open flags are the OS-boundary contract: configured nonregular paths
		// must not block before admission can inspect and reject their type.
		if gotPath != path || flags != os.O_RDONLY|syscall.O_NONBLOCK || mode != 0 {
			t.Errorf("open request did not retain nonblocking read-only admission")
		}
		// #nosec G304 -- The test owns this regular-file path and opens it without blocking flags.
		return os.Open(path)
	}}
	if contents, err := reader.ReadFile(context.Background(), path, 8); err != nil || string(contents) != "material" {
		t.Fatalf("bounded material read returned bytes=%d error=%v", len(contents), err)
	}
}
