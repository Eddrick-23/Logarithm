package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/attack"
	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/files"
)

func printJson(obj any) {
	bytes, _ := json.MarshalIndent(obj, "", "\t")
	fmt.Println(string(bytes))
}

func run(ctx context.Context, configPath string, interval int, duration time.Duration,
	warmupDuration time.Duration, restDuration time.Duration, outDir string, showConfig bool) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.ParseConfig(configPath) // load config
	if err != nil {
		return err
	}

	if showConfig {
		printJson(*cfg)
	}
	fmt.Printf("Test configs: RPS: %v, BatchSize:%v, Duration: %v, Warmup: %v, contentType: %v, Encoding: %v\n",
		cfg.Rps,
		cfg.BatchSize,
		duration,
		warmupDuration,
		cfg.ContentType,
		cfg.Encoding,
	)

	loadTestRunner, err := attack.NewRunner(cfg, outDir)
	if err != nil {
		return err
	}

	if err := loadTestRunner.CheckHealth(); err != nil {
		return err
	}

	if warmupDuration > 0 {
		fmt.Printf("Starting warmup run for %s\n", warmupDuration)
		if err := loadTestRunner.Warmup(ctx, warmupDuration); err != nil {
			return err
		}
		fmt.Printf("warmup complete, resting for %s\n", restDuration)

		select {
		case <-time.After(restDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// signal bash script that initialisation, healthchecks and warmup is done and ready to start main test
	if err := files.CreateReadyFile(outDir, ".sampling_ready"); err != nil {
		return err
	}
	defer func() {
		if err := files.DeleteFile(outDir, ".sampling_ready"); err != nil {
			fmt.Printf("failed to remove temporary signal file, %v", err)
		}
	}()

	if err := loadTestRunner.Run(ctx, duration, time.Duration(interval)*time.Second); err != nil {
		return err
	}

	return nil
}

func main() {
	pathToConfigPtr := flag.String("config", "", "path to config.json file")
	intervalPtr := flag.Int("interval", 1, "how often to log attack progress in seconds")
	durationPtr := flag.Int("duration", 1, "test duration in seconds")
	warmupPtr := flag.Int("warmup", 0, "warmup duration in seconds")
	warmupRestPtr := flag.Int("rest", 5, "resting duration after warmup in seconds")
	outdirPtr := flag.String("o", "results", "output directory of result.bin")
	showConfigPtr := flag.Bool("showConfig", false, "display parsed config once at startup")

	flag.Parse()

	if *pathToConfigPtr == "" {
		fmt.Println("no config provided")
		os.Exit(1)
	}

	if *durationPtr <= 0 {
		fmt.Println("duration must be positive")
		os.Exit(1)
	}

	warmupDuration := time.Duration(*warmupPtr) * time.Second
	restDuration := time.Duration(*warmupRestPtr) * time.Second
	testDuration := time.Duration(*durationPtr) * time.Second

	ctx := context.Background()
	if err := run(ctx, *pathToConfigPtr, *intervalPtr, testDuration, warmupDuration, restDuration, *outdirPtr, *showConfigPtr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
