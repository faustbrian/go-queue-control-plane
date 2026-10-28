package main

import (
	"context"
	"net"
	"net/http"
	"time"

	gopostgres "github.com/faustbrian/go-postgres/v3"
	"github.com/faustbrian/go-queue-control-plane/v6/apihttp"
	"github.com/faustbrian/go-queue-control-plane/v6/control"
	controlpostgres "github.com/faustbrian/go-queue-control-plane/v6/postgres"
	"github.com/faustbrian/go-queue-control-plane/v6/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	productionRateLimit  uint32 = 120
	productionRateWindow        = time.Minute
	productionRateKeys          = 10_000
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildTime    string
)

// postgresPoolConfig keeps DSN acquisition in the application. Native parsing
// may synchronously consult environment and files; cancellation is cooperative
// at the boundaries, not a promise to preempt those native operations.
func postgresPoolConfig(dsn string) gopostgres.Config {
	return gopostgres.Config{
		DSN:           dsn,
		StartupPolicy: gopostgres.StartupPing,
		ResolveDSN: func(ctx context.Context, dsn string) (*gopostgres.PoolConfig, error) {
			if err := ctx.Err(); err != nil {
				return nil, context.Cause(ctx)
			}
			// PrepareConfig checks cancellation again when this resolver returns.
			return pgxpool.ParseConfig(dsn)
		},
	}
}

func productionDependencies() processDependencies {
	return processDependencies{
		buildInfo: func() (apihttp.BuildInfo, error) {
			return parseBuildInfo(buildVersion, buildCommit, buildTime)
		},
		buildTelemetry: func(
			ctx context.Context,
			config Config,
			build apihttp.BuildInfo,
		) (processTelemetry, error) {
			runtime, err := buildProductionTelemetry(ctx, config, build)
			if runtime == nil {
				return nil, err
			}

			return runtime, err
		},
		loadAccess:     server.LoadStaticAccessFile,
		loadWorkloads:  loadProductionWorkloads,
		loadManagement: loadProductionManagement,
		routeDispatchers: func(dataPlane, workloads control.Dispatcher) (control.Dispatcher, error) {
			return control.NewRoutingDispatcher(dataPlane, workloads)
		},
		retain:           executeProductionRetention,
		openPool:         gopostgres.Connect,
		buildPersistence: controlpostgres.NewRuntime,
		buildRateLimiter: func() (apihttp.RateLimiter, error) {
			return apihttp.NewFixedWindowRateLimiter(
				productionRateLimit,
				productionRateWindow,
				productionRateKeys,
				time.Now,
			)
		},
		listen: net.Listen,
		buildServer: func(
			listener net.Listener,
			handler http.Handler,
			config server.Config,
		) (processServer, error) {
			return server.New(listener, handler, config)
		},
		dispatcher: control.UnavailableDispatcher{},
	}
}
