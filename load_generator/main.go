package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/attack"
	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/files"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func printJson(obj any) {
	bytes, _ := json.MarshalIndent(obj, "", "\t")
	fmt.Println(string(bytes))
}

func checkHealth(url string) error {
	fmt.Println("checking health at: " + url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned unhealthy status: %d", resp.StatusCode)
	}

	fmt.Println("endpoint healthy")
	return nil
}

func run(ctx context.Context, configPath string, interval int, showConfig bool) error {
	cfg, err := config.ParseConfig(configPath) // load config
	if err != nil {
		return err
	}

	if showConfig {
		printJson(*cfg)
	} else {
		fmt.Printf("Test configs: RPS: %v, BatchSize:%v, Duration: %v, useGzip: %v\n",
			cfg.Rps,
			cfg.BatchSize,
			cfg.Duration,
			cfg.Gzip,
		)
	}

	if err := checkHealth(cfg.HealthUrl); err != nil { // check endpoint health
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resultsFile, err := files.CreateResultFile() // create output file
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()

	var metrics vegeta.Metrics //setup controllers and attackers
	attack, err := attack.NewAttack(cfg)
	if err != nil {
		return fmt.Errorf("failed to create attacker: %w", err)
	}

	enc := vegeta.NewEncoder(resultsFile)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	resultsChan := attack.Start()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var lastLatency time.Duration
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[%s] Reqests sent: %-6d | Last Latency: %dms\n",
					time.Now().Format("15:04:05"),
					metrics.Requests,
					lastLatency.Milliseconds(),
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
			case <-ctx.Done():
				fmt.Println("Timeout or cancelled, cleaning up...")
				return
			}
		}
	}()
	wg.Wait()
	metrics.Close()
	fmt.Printf("Test Summary\n: P99 Latency: %v\n Throughput: %vrps\n Requests Sent: %v\n SuccessRate: %v\n TotalLogsSent: %v\n Logs/sec: %v\n",
		metrics.Latencies.P99,
		metrics.Throughput,
		metrics.Requests,
		metrics.Success*100,
		cfg.BatchSize*int(metrics.Requests),
		(cfg.BatchSize*int(metrics.Requests))/int(cfg.Duration.Seconds()),
	)
	return nil
}

func main() {
	pathToConfigPtr := flag.String("config", "", "path to config.json file")
	intervalPtr := flag.Int("interval", 1, "how often to log attack progress (seconds)")
	showConfigPtr := flag.Bool("showConfig", false, "display parsed config once at startup")

	flag.Parse()

	if *pathToConfigPtr == "" {
		fmt.Println("no config provided")
		os.Exit(1)
	}
	ctx := context.Background()
	if err := run(ctx, *pathToConfigPtr, *intervalPtr, *showConfigPtr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
