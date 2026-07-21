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

// NewGRPCServer constructs the ingester's gRPC server. It forces all RPCs
// through the raw-bytes codec so that arbitrary client payloads are proxied
// to NATS without proto schema validation, and it registers the standard
// gRPC health checking service (grpc.health.v1.Health) which is reachable
// via the ordinary proto codec fallback - see rawCodec.Marshal/Unmarshal.
func NewGRPCServer(logger *slog.Logger, producer transport.Producer, presize int64, bufferLimit int64) *grpc.Server {
	const system = "" // means overall server status

	proxyHandler := newProxyHandler(logger, producer, transport.LogStreamSubject)

	grpcServer := grpc.NewServer(
		grpc.ForceServerCodecV2(encoding.GetCodecV2(CodecName)),
		grpc.UnknownServiceHandler(proxyHandler.NewStreamHandler(presize, bufferLimit)),
	)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)
	healthcheck.SetServingStatus(system, healthpb.HealthCheckResponse_SERVING)

	return grpcServer
}
