package main

import (
	"context"
	"time"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
	gokitmetrics "github.com/fino-io/gokit/metrics"
	"github.com/fino-io/gokit/middleware/ratelimit"
	nserver "github.com/fino-io/gokit/server"
	"github.com/fino-io/gokit/tracing"
	"github.com/fino-io/orderpay/backend/infra/nacos"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/api"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/app"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/service"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/auth"
	infrarepo "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo"
	pb "github.com/fino-io/orderpay/go/orderpay/v1"
)

func loadConfig() (nserver.Config, error) {
	if err := nacos.LoadServiceConfig("orderpay-server"); err != nil {
		return nserver.Config{}, err
	}

	var cfg nserver.Config
	if err := config.ScanFrom(&cfg, "orderpayServer", "server"); err != nil {
		return nserver.Config{}, err
	}

	if err := logs.NewConfig(); err != nil {
		return nserver.Config{}, err
	}

	return cfg, nil
}

func serve(ctx context.Context, cfg nserver.Config) error {
	tracer, shutdownTrace, err := tracing.New("orderpay-server")
	if err != nil {
		logs.Warnw("failed to initialize tracing", "error", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), nserver.DefaultShutdownTimeout)
		defer cancel()
		if err := shutdownTrace(shutdownCtx); err != nil {
			logs.Warnw("failed to shut down tracing", "error", err)
		}
	}()

	instrumentation, err := gokitmetrics.New("orderpay")
	if err != nil {
		logs.Warnw("failed to initialize metrics", "error", err)
		instrumentation = gokitmetrics.Disabled()
	}

	infra, err := newInfra()
	if err != nil {
		return err
	}
	defer func() {
		if err := infra.close(); err != nil {
			logs.Errorw("failed to close infrastructure", "error", err)
		}
	}()

	server, err := buildServer(infra)
	if err != nil {
		return err
	}

	return api.RunContext(ctx, cfg, server, tracer, instrumentation)
}

func buildServer(deps infra) (pb.OrderpayServiceServer, error) {
	credentials, err := auth.Load()
	if err != nil {
		return nil, err
	}

	var limits struct {
		ReadEvery  time.Duration `json:"readEvery"`
		WriteEvery time.Duration `json:"writeEvery"`
		Burst      int           `json:"burst"`
		MaxKeys    int           `json:"maxKeys"`
		TTL        time.Duration `json:"ttl"`
	}
	if err := config.ScanFrom(&limits, "business"); err != nil {
		return nil, err
	}
	readLimiter, err := ratelimit.NewLocalKeyedRateLimiter(limits.ReadEvery, limits.Burst, limits.MaxKeys, limits.TTL)
	if err != nil {
		return nil, err
	}
	writeLimiter, err := ratelimit.NewLocalKeyedRateLimiter(limits.WriteEvery, limits.Burst, limits.MaxKeys, limits.TTL)
	if err != nil {
		return nil, err
	}

	if err := migrateSchema(deps.DB); err != nil {
		return nil, err
	}

	store := infrarepo.NewWorkflowStore(deps.DB.NewSession(context.Background()), deps.Pagination)
	if err := store.Initialize(); err != nil {
		return nil, err
	}

	return app.NewService(app.ServiceDeps{
		ReadLimiter: readLimiter,
		Workflow:    service.NewWorkflow(store, nil, writeLimiter),
		Credentials: credentials,
	}), nil
}
