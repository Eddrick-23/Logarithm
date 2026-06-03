package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

func printJson(obj interface{}) {
	bytes, _ := json.MarshalIndent(obj, "", "\t")
	fmt.Println(string(bytes))
}

func run(ctx context.Context, configPath string, interval int) error {
	// TODO work on config parsing from config.json
	// need to standardise payload randomisation schema
	// TODO, find how to send POST requests and whethere we can randomise payloads
	cfg, err := parseConfig(configPath)
	if err != nil {
		return err
	}
	printJson(*cfg)
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resultsFile, err := createResultFile()
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()

	var metrics vegeta.Metrics
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
	fmt.Println(metrics.Latencies.P99)
	fmt.Println(metrics.Throughput)
	fmt.Println(metrics.Requests)
	fmt.Println(metrics.StatusCodes)
	return nil
}

func main() {
	pathToConfigPtr := flag.String("config", "", "path to config.json file")
	intervalPtr := flag.Int("interval", 1, "how often to log attack progress (seconds)")

	flag.Parse()

	if *pathToConfigPtr == "" {
		fmt.Println("no config provided")
		os.Exit(1)
	}
	ctx := context.Background()
	if err := run(ctx, *pathToConfigPtr, *intervalPtr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
