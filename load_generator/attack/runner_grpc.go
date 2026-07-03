package attack

import (
	"context"
	"fmt"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/runner"
	ghz "github.com/bojand/ghz/runner"
	"github.com/jhump/protoreflect/desc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const method string = "opentelemetry.proto.collector.logs.v1.LogsService/Export"

var _ runner.Runner = (*grpcRunner)(nil)

type grpcRunner struct {
	cfg            *config.CleanConfig
	payloadFactory *PayloadFactory
	outDir         string
}

func newGRPCRunner(cfg *config.CleanConfig, payloadFactory *PayloadFactory, outDir string) *grpcRunner {
	return &grpcRunner{
		cfg:            cfg,
		payloadFactory: payloadFactory,
		outDir:         outDir,
	}
}

func (g *grpcRunner) CheckHealth() error {
	fmt.Println("checking grpc server health at: " + g.cfg.TargetUrl)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(g.cfg.TargetUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return err
	}

	healthClient := healthpb.NewHealthClient(conn)
	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{Service: ""})

	if err != nil {
		return err
	}

	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("service returned unhealthy status: %v", resp.Status)
	}

	fmt.Println("grpc server healthy")
	return nil
}

func (g *grpcRunner) Warmup(ctx context.Context, duration time.Duration) error {
	dataFunc := func(mtd *desc.MethodDescriptor, callData *ghz.CallData) []byte {
		payload, err := g.payloadFactory.GenerateEncodedPayload()
		if err != nil {
			fmt.Printf("failed to generate payload: %v\n", err)
			return nil
		}
		return payload
	}
	_, err := ghz.Run(
		method,
		g.cfg.TargetUrl,
		ghz.WithInsecure(true),
		ghz.WithRPS(uint(g.cfg.Rps)),
		ghz.WithRunDuration(duration),
		ghz.WithBinaryDataFunc(dataFunc),
	)
	return err
}

func (g *grpcRunner) Run(ctx context.Context, duration time.Duration, logInterval time.Duration) error {
	return nil
}
