package ingester

import (
	"log/slog"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func NewGRPCServer(logger *slog.Logger, producer transport.Producer) *grpc.Server {
	const system = "" // means overall server status

	proxyHandler := NewProxyHandler(logger, producer, transport.LogStreamSubject)

	grpcServer := grpc.NewServer(
		grpc.ForceServerCodecV2(encoding.GetCodecV2(CodecName)),
		grpc.UnknownServiceHandler(proxyHandler.StreamHandler),
	)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)
	healthcheck.SetServingStatus(system, healthpb.HealthCheckResponse_SERVING)

	return grpcServer
}
