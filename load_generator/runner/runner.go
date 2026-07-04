package runner

import (
	"context"
	"time"
)

// Runner defines the lifecycle of a load test execution
type Runner interface {
	CheckHealth() error
	Warmup(ctx context.Context, duration time.Duration) error
	Run(ctx context.Context, duration time.Duration, logInterval time.Duration) error
}
