package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faustbrian/go-queue-control-plane/v2/apihttp"
	telemetry "github.com/faustbrian/go-telemetry/v2"
)

func TestProductionTelemetryConfigUsesExplicitSecureRuntime(t *testing.T) {
	t.Parallel()

	config := productionTelemetryConfig(Config{
		TelemetryEndpoint:        "collector.telemetry.svc:4317",
		TelemetryProtocol:        "http/protobuf",
		TelemetryInsecure:        false,
		TelemetryEnvironment:     "production",
		TelemetryInstance:        "control-plane-1",
		TelemetryCAFile:          "/var/run/telemetry/ca.pem",
		TelemetryCertificateFile: "/var/run/telemetry/client.pem",
		TelemetryPrivateKeyFile:  "/var/run/telemetry/client-key.pem",
		TelemetryServerName:      "collector.telemetry.svc",
	}, apihttp.BuildInfo{Version: "v1.2.3"})

	if config.Service.Name != "go-queue-control-plane" ||
		config.Service.Version != "v1.2.3" ||
		config.Service.Instance != "control-plane-1" ||
		config.Environment != "production" ||
		config.RegisterGlobal ||
		!config.Traces.Enabled || !config.Metrics.Enabled ||
		config.Traces.Exporter.Endpoint != "collector.telemetry.svc:4317" ||
		config.Metrics.Exporter.Endpoint != "collector.telemetry.svc:4317" ||
		config.Traces.Exporter.Protocol != telemetry.ProtocolHTTPProtobuf ||
		config.Metrics.Exporter.Protocol != telemetry.ProtocolHTTPProtobuf ||
		config.Traces.Exporter.TLS.Insecure || config.Metrics.Exporter.TLS.Insecure ||
		config.Traces.Exporter.TLS.CAFile != "/var/run/telemetry/ca.pem" ||
		config.Traces.Exporter.TLS.FileReader == nil ||
		config.Metrics.Exporter.TLS.FileReader == nil ||
		config.Metrics.Exporter.TLS.CertificateFile != "/var/run/telemetry/client.pem" ||
		config.Traces.Exporter.TLS.PrivateKeyFile != "/var/run/telemetry/client-key.pem" ||
		config.Metrics.Exporter.TLS.ServerName != "collector.telemetry.svc" {
		t.Fatalf("productionTelemetryConfig() = %+v", config)
	}
}

func TestTLSMaterialReaderBoundsAndCancelsFileAccess(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	reader := tlsMaterialReader{}
	contents, err := reader.ReadFile(context.Background(), path, 11)
	if err != nil || string(contents) != "certificate" {
		t.Fatalf("ReadFile() = (%q, %v), want certificate", contents, err)
	}
	if _, err := reader.ReadFile(context.Background(), path, 10); err == nil {
		t.Fatal("ReadFile(over limit) returned nil error")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := reader.ReadFile(cancelled, path, 11); !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadFile(cancelled) error = %v, want context canceled", err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 1<<20+1)), 0o600); err != nil {
		t.Fatalf("WriteFile(oversize) error = %v", err)
	}
	if _, err := reader.ReadFile(context.Background(), path, 1<<20); err == nil {
		t.Fatal("ReadFile(oversize) returned nil error")
	}
}

func TestTLSMaterialReaderPreservesCancellationAfterStat(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	reader := tlsMaterialReader{stat: func(*os.File) (os.FileInfo, error) {
		cancel()
		return nil, errors.New("stat unavailable")
	}}
	if _, err := reader.ReadFile(ctx, path, 11); !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadFile(cancelled during Stat) error = %v, want context canceled", err)
	}
}

func TestTLSMaterialReaderRejectsInvalidLimitsAndMissingFiles(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	reader := tlsMaterialReader{}
	for _, maxBytes := range []int{0, -1, maxTLSMaterialBytes + 1} {
		if contents, err := reader.ReadFile(context.Background(), path, maxBytes); contents != nil || !errors.Is(err, errTLSMaterial) {
			t.Errorf("ReadFile(maxBytes=%d) = (%q, %v), want invalid TLS material", maxBytes, contents, err)
		}
	}
	if contents, err := reader.ReadFile(context.Background(), filepath.Join(t.TempDir(), "missing.pem"), 11); contents != nil || !errors.Is(err, errTLSMaterial) {
		t.Errorf("ReadFile(missing) = (%q, %v), want invalid TLS material", contents, err)
	}
}

func TestTLSMaterialReaderRejectsDirectory(t *testing.T) {
	t.Parallel()

	if contents, err := (tlsMaterialReader{}).ReadFile(context.Background(), t.TempDir(), maxTLSMaterialBytes); contents != nil || !errors.Is(err, errTLSMaterial) {
		t.Fatalf("ReadFile(directory) = (%q, %v), want invalid TLS material", contents, err)
	}
}

type cancelAfterInitialCheckContext struct {
	context.Context
	cancel  context.CancelFunc
	checked bool
}

func (ctx *cancelAfterInitialCheckContext) Err() error {
	err := ctx.Context.Err()
	if !ctx.checked {
		ctx.checked = true
		ctx.cancel()
	}
	return err
}

func TestTLSMaterialReaderPreservesCancellationAfterOpenFailure(t *testing.T) {
	t.Parallel()

	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cancelAfterInitialCheckContext{Context: base, cancel: cancel}
	path := filepath.Join(t.TempDir(), "missing.pem")
	if contents, err := (tlsMaterialReader{}).ReadFile(ctx, path, 11); contents != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadFile(cancelled during open) = (%q, %v), want context canceled", contents, err)
	}
}

type sizeOverrideFileInfo struct {
	os.FileInfo
	size int64
}

func (info sizeOverrideFileInfo) Size() int64 { return info.size }

type cancelOnSizeFileInfo struct {
	os.FileInfo
	cancel context.CancelFunc
}

func (info cancelOnSizeFileInfo) Size() int64 {
	info.cancel()
	return info.FileInfo.Size()
}

func TestTLSMaterialReaderRejectsChangedOrUnreadableFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Run("oversize after stat", func(t *testing.T) {
		reader := tlsMaterialReader{stat: func(file *os.File) (os.FileInfo, error) {
			info, err := file.Stat()
			return sizeOverrideFileInfo{FileInfo: info, size: 1}, err
		}}
		if contents, err := reader.ReadFile(context.Background(), path, 10); contents != nil || !errors.Is(err, errTLSMaterial) {
			t.Fatalf("ReadFile(oversize after stat) = (%q, %v), want invalid TLS material", contents, err)
		}
	})
	t.Run("read failure", func(t *testing.T) {
		reader := tlsMaterialReader{stat: func(file *os.File) (os.FileInfo, error) {
			info, err := file.Stat()
			_ = file.Close()
			return info, err
		}}
		if contents, err := reader.ReadFile(context.Background(), path, 11); contents != nil || !errors.Is(err, errTLSMaterial) {
			t.Fatalf("ReadFile(closed before read) = (%q, %v), want invalid TLS material", contents, err)
		}
	})
}

func TestTLSMaterialReaderPreservesCancellationAfterSizeCheck(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := tlsMaterialReader{stat: func(file *os.File) (os.FileInfo, error) {
		info, err := file.Stat()
		return cancelOnSizeFileInfo{FileInfo: info, cancel: cancel}, err
	}}
	if contents, err := reader.ReadFile(ctx, path, 11); contents != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadFile(cancelled after size check) = (%q, %v), want context canceled", contents, err)
	}
}
