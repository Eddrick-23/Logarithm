package attack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/files"
	"github.com/Eddrick-23/Logarithm/load_generator/runner"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const method string = "opentelemetry.proto.collector.logs.v1.LogsService/Export"
const workers int = 5 // TODO make configurable via config.json

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

func (g *grpcRunner) dial() (*grpc.ClientConn, error) {
	return grpc.NewClient(
		g.cfg.TargetUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func (g *grpcRunner) CheckHealth() error {
	fmt.Println("checking grpc server health at: " + g.cfg.TargetUrl)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := g.dial()

	if err != nil {
		return err
	}

	defer conn.Close()

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
	conn, err := g.dial()

	if err != nil {
		return err
	}

	defer conn.Close()
	resultsChan := g.attack(ctx, conn, duration)

	for range resultsChan {
		// discard results for warmup
	}

	return nil
}

func (g *grpcRunner) Run(ctx context.Context, duration time.Duration, logInterval time.Duration) error {
	conn, err := g.dial()

	if err != nil {
		return err
	}

	defer conn.Close()

	// setup output file
	resultsFile, err := files.CreateResultFile(g.outDir, "results.bin") // create output file
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()
	enc := vegeta.NewEncoder(resultsFile)

	// run main test loop
	var metrics vegeta.Metrics //setup controllers and attackers

	ticker := time.NewTicker(logInterval)
	defer ticker.Stop()

	resultsChan := g.attack(ctx, conn, duration)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var lastLatency time.Duration
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[%s] Reqests sent: %-6d | Last Latency: %v\n",
					time.Now().Format("15:04:05"),
					metrics.Requests,
					lastLatency.String(),
				)
			case res, ok := <-resultsChan:
				if !ok {
					fmt.Println("testing done. Cleaning up...")
					return
				}
				metrics.Add(res)
				lastLatency = res.Latency

				if err := enc.Encode(res); err != nil {
					fmt.Printf("failed to write result bytes to results file: %v", err)
				}

				// not affected by ctx.Done. Let results chan drain fully so that workers can exit.
			}

		}
	}()
	wg.Wait()
	metrics.Close()

	// print metrics summary
	fmt.Printf("Test Summary:\n P99 Latency: %v\n Throughput: %vrps\n Requests Sent: %v\n SuccessRate: %v\n TotalLogsSent: %v\n Logs/sec: %v\n AveragePayloadSize: %.2fKB\n",
		metrics.Latencies.P99,
		metrics.Throughput,
		metrics.Requests,
		metrics.Success*100,
		g.cfg.BatchSize*int(metrics.Requests),
		(g.cfg.BatchSize*int(metrics.Requests))/int(duration.Seconds()),
		metrics.BytesOut.Mean/1000,
	)

	return nil
}

func (g *grpcRunner) attack(ctx context.Context, conn *grpc.ClientConn, duration time.Duration) <-chan *vegeta.Result {
	pacer := vegeta.Rate{Freq: g.cfg.Rps, Per: time.Second}

	jobs := make(chan struct{})
	results := make(chan *vegeta.Result)

	// start a worker pool that reads signals from jobs
	// then sends an invoke call, passing result to the results channel
	var wg sync.WaitGroup
	wg.Add(workers) // assume 5 workers
	for range workers {
		go func() {
			defer wg.Done()
			for range jobs {
				results <- g.invoke(ctx, conn)
			}
		}()
	}

	// dispatch job signals to worker pool
	// closes job duration at the end so workers drain and exit
	go func() {
		defer close(jobs)
		start := time.Now()
		for hits := uint64(0); ; hits++ {
			wait, stop := pacer.Pace(time.Since(start), hits)
			if stop || time.Since(start) >= duration {
				return // return and force close jobs chan
			}

			select { // sleep for specified duration, cancellable
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}

			select {
			case jobs <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// invoke sends one encoded payload, and times a single invoke call,
// so we are able to time request-response latency.
// Results are attatched in *vegeta.Result
func (g *grpcRunner) invoke(ctx context.Context, conn *grpc.ClientConn) *vegeta.Result {
	payload, err := g.payloadFactory.GenerateEncodedPayload()
	if err != nil {
		return &vegeta.Result{
			Timestamp: time.Now(),
			Error:     fmt.Sprintf("failed to generate payload: %v", err),
		}
	}

	req := &rawFrame{RawBytes: payload}
	reply := &rawFrame{}

	begin := time.Now()
	err = conn.Invoke(ctx, method, req, reply, grpc.CallContentSubtype(codecName))
	latency := time.Since(begin)

	res := &vegeta.Result{
		Timestamp: begin,
		Latency:   latency,
		BytesOut:  uint64(len(payload)),
		BytesIn:   uint64(len(reply.RawBytes)),
	}

	if err != nil {
		st, _ := status.FromError(err)
		res.Code = uint16(st.Code())
		res.Error = st.Message()
	} else {
		res.Code = uint16(0) //codes.OK
	}

	return res
}
