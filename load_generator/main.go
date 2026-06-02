package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func createResultFile() (*os.File, error) {
	// check results directory
	// check num files
	// create bin file as results<num>.bin
	// os.Create()
	return nil, nil
}

func run(ctx context.Context, w io.Writer, args []string) error {
	rate := vegeta.Rate{Freq: 100, Per: time.Second}
	duration := 20 * time.Second

	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:8090/health",
	})

	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "logarithm load gen") {
		metrics.Add(res)

	}
	metrics.Close()
	fmt.Println(metrics.Latencies.P99)
	fmt.Println(metrics.Throughput)
	fmt.Println(metrics.Requests)
	fmt.Println(metrics.StatusCodes)
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
