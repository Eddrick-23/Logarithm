package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/files"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func createResultFile() (*os.File, error) {
	const resultsDir = "results"
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v", err)
		return nil, err
	}
	exists, err := files.FolderExists(filepath.Join(dir, resultsDir))
	if err != nil {
		fmt.Printf("Could not verify if results folder exists: %v", err)
		return nil, err
	}

	if !exists {
		// 0755 give read/write/execute to owner
		if err := os.Mkdir(filepath.Join(dir, resultsDir), 0755); err != nil {
			fmt.Printf("failed to create results directory: %v", err)
			return nil, err
		}
	}

	count, err := files.NumFilesInFolder(filepath.Join(dir, resultsDir))

	filename := "results"
	if count > 0 {
		filename = filename + strconv.Itoa(count)
	}
	filename += ".bin"

	file, err := os.Create(filepath.Join(dir, "results", filename))
	if err != nil {
		fmt.Printf("error creating results file: %v", err)
		return nil, err
	}
	return file, nil
}

func setupAttack(rps int, duration time.Duration) func() <-chan *vegeta.Result {
	return func() <-chan *vegeta.Result {
		rate := vegeta.Rate{Freq: rps, Per: time.Second}

		targeter := vegeta.NewStaticTargeter(vegeta.Target{
			Method: "GET",
			URL:    "http://localhost:8090/health",
		})

		attacker := vegeta.NewAttacker()
		return attacker.Attack(targeter, rate, duration, "logarithm load generator")
	}
}

func parseConfig(configPath string) (*config.CleanConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	exists, err := files.FileExists(filepath.Join(dir, configPath))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("config file does not exist")
	}

	content, err := os.ReadFile(filepath.Join(dir, configPath))
	if err != nil {
		return nil, err
	}

	var rawCfg config.RawConfig
	if err = json.Unmarshal(content, &rawCfg); err != nil {
		return nil, err
	}

	cfg := config.ValidateAndCleanConfig(rawCfg)

	return &cfg, nil
}

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
	// TODO, find how to send POST requests and whethere we can randomise payloads
	cfg, err := parseConfig(configPath) // load config
	if err != nil {
		return err
	}

	if showConfig {
		printJson(*cfg)
	} else {
		fmt.Printf("Test configs: RPS: %v, BatchSize:%v, Duration, %v\n",
			cfg.Rps,
			cfg.BatchSize,
			cfg.Duration,
		)
	}

	if err := checkHealth(cfg.HealthUrl); err != nil { // check endpoint health
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resultsFile, err := createResultFile() // create output file
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()

	var metrics vegeta.Metrics //setup controllers and attackers
	startAttack := setupAttack(10, 10*time.Second)
	enc := vegeta.NewEncoder(resultsFile)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	resultsChan := startAttack()
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
	fmt.Printf("Test Summary: P99 Latency: %v | Throughput: %vrps | Requests Sent: %v | successRate: %v",
		metrics.Latencies.P99,
		metrics.Throughput,
		metrics.Requests,
		metrics.Success*100,
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
