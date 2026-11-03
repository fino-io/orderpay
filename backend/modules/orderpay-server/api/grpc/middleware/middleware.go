package middleware

import (
	"github.com/fino-io/gokit/tracing"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func ServerOptions(tracer trace.Tracer, instrument grpc.UnaryServerInterceptor) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(UnaryServerInterceptors(tracer, instrument)...),
	}
}

func UnaryServerInterceptors(tracer trace.Tracer, instrument grpc.UnaryServerInterceptor) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		tracing.GRPCUnaryServerTracingInterceptor(tracer),
		instrument,
		accessLogUnaryServerInterceptor(),
		errorMappingUnaryServerInterceptor(),
		recoveryUnaryServerInterceptor(),
	}
}
