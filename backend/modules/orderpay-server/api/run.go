package api

import (
	"context"
	"errors"

	gokitmetrics "github.com/fino-io/gokit/metrics"
	"github.com/fino-io/gokit/sd"
	nserver "github.com/fino-io/gokit/server"
	"go.opentelemetry.io/otel/trace"
	_ "go.uber.org/automaxprocs"

	"github.com/fino-io/orderpay/backend/modules/orderpay-server/api/endpoint"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/api/endpoint/middleware"
	grpc2 "github.com/fino-io/orderpay/backend/modules/orderpay-server/api/grpc"
	http2 "github.com/fino-io/orderpay/backend/modules/orderpay-server/api/http"
	pb "github.com/fino-io/orderpay/go/orderpay/v1"
)

const FullServiceName = "orderpay.v1.OrderpayService"

// RunContext assembles the service transports and waits for shutdown.
func RunContext(ctx context.Context, cfg nserver.Config, service pb.OrderpayServiceServer, tracer trace.Tracer, instrumentation *gokitmetrics.Instrumentation) (err error) {
	if service == nil {
		return errors.New("nil server")
	}
	if tracer == nil {
		return errors.New("nil tracer")
	}
	if instrumentation == nil {
		instrumentation = gokitmetrics.Disabled()
	}

	endpoints := endpoint.NewEndpoints(service)
	endpoints = middleware.WrapEndpoints(endpoints, middleware.DefaultOptions(tracer))
	servers, err := nserver.StartTransportServers(
		cfg.HttpAddr, http2.NewHandler(endpoints, tracer, instrumentation.HTTPMiddleware),
		cfg.GrpcAddr, grpc2.NewServer(endpoints, tracer, instrumentation.GRPCUnaryInterceptor()),
		cfg.DebugAddr, nserver.NewDebugHandler(cfg.HttpAddr, instrumentation.Handler()),
	)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, servers.Shutdown())
	}()

	sdClient, err := sd.New()
	if err != nil {
		return err
	}
	if err := sdClient.RegisterService(FullServiceName, servers.HTTPAddr(), servers.GRPCAddr()); err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, sdClient.Deregister())
	}()

	return servers.Wait(ctx)
}
