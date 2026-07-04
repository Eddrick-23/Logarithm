// implement runner interface for http protocol
package attack

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/files"
	"github.com/Eddrick-23/Logarithm/load_generator/runner"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var _ runner.Runner = (*httpRunner)(nil)

type httpRunner struct {
	cfg            *config.CleanConfig
	payloadFactory *PayloadFactory
	outDir         string
}

func newHttpRunner(cfg *config.CleanConfig, payloadFactory *PayloadFactory, outDir string) runner.Runner {
	return &httpRunner{
		cfg:            cfg,
		payloadFactory: payloadFactory,
		outDir:         outDir,
	}
}

func (h *httpRunner) CheckHealth() error {
	fmt.Println("checking health at: " + h.cfg.HttpHealthUrl)
	resp, err := http.Get(h.cfg.HttpHealthUrl)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned unhealthy status: %d", resp.StatusCode)
	}

	fmt.Println("http server healthy")
	return nil
}

func (h *httpRunner) Warmup(ctx context.Context, duration time.Duration) error {
	// Use the same targeter logic as the main run, but discard results
	attacker := vegeta.NewAttacker()
	rate := vegeta.Rate{Freq: h.cfg.Rps, Per: time.Second}
	targeter := h.createTargeter()

	for range attacker.Attack(targeter, rate, duration, "warmup") {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// discard result
		}
	}
	return nil
}

func (h *httpRunner) Run(ctx context.Context, duration time.Duration, logInterval time.Duration) error {
	// set up output file
	resultsFile, err := files.CreateResultFile(h.outDir, "results.bin") // create output file
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()
	enc := vegeta.NewEncoder(resultsFile)

	// run main test loop
	var metrics vegeta.Metrics //setup controllers and attackers

	ticker := time.NewTicker(logInterval)
	defer ticker.Stop()

	rate := vegeta.Rate{Freq: h.cfg.Rps, Per: time.Second}
	targeter := h.createTargeter()
	attacker := vegeta.NewAttacker()
	resultsChan := attacker.Attack(targeter, rate, duration, "http-test")

	var wg sync.WaitGroup
	wg.Go(func() {
		var lastLatency time.Duration
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[%s] Requests sent: %-6d | Last Latency: %v\n",
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
	})
	wg.Wait()
	metrics.Close()

	// print metrics summary
	fmt.Printf("Test Summary:\n P99 Latency: %v\n Throughput: %vrps\n Requests Sent: %v\n SuccessRate: %v\n TotalLogsSent: %v\n Logs/sec: %v\n AveragePayloadSize: %.2fKB\n",
		metrics.Latencies.P99,
		metrics.Throughput,
		metrics.Requests,
		metrics.Success*100,
		h.cfg.BatchSize*int(metrics.Requests),
		(h.cfg.BatchSize*int(metrics.Requests))/int(duration.Seconds()),
		metrics.BytesOut.Mean/1000,
	)
	return nil
}

// helper to setup vegeta targetting and keep run clean
func (h *httpRunner) createTargeter() vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		encoded, err := h.payloadFactory.GenerateEncodedPayload()
		if err != nil {
			return err
		}

		if tgt.Header == nil {
			tgt.Header = make(http.Header)
		}

		h.writeHeaders(tgt)

		tgt.Body = encoded
		tgt.Method = h.cfg.HttpMethod
		tgt.URL = h.cfg.TargetUrl
		return nil
	}
}

func (h *httpRunner) writeHeaders(tgt *vegeta.Target) {
	if tgt.Header == nil {
		tgt.Header = make(http.Header)
	}

	switch h.cfg.ContentType {
	case "proto":
		tgt.Header.Set("Content-Type", "application/x-protobuf")
	case "json":
		tgt.Header.Set("Content-Type", "application/json")
	default:
		// don't set
	}

	switch h.cfg.Encoding {
	case "zstd":
		tgt.Header.Set("Content-Encoding", "zstd")
	case "gzip":
		tgt.Header.Set("Content-Encoding", "gzip")
	default:
		tgt.Header.Set("Content-Encoding", "identity")
	}
}
