package config

import (
	"fmt"
	"math"
	"time"

	"github.com/Eddrick-23/Logarithm/api/schemas"
)

type BodyTokenFormat struct {
	Min        int      `json:"min"`
	Max        int      `json:"max"`
	Dictionary []string `json:"dictionary"`
}

type RawConfig struct {
	Seed                 int                `json:"seed"`
	PoolSize             int                `json:"poolSize"`
	HealthUrl            string             `json:"healthUrl"`
	TargetUrl            string             `json:"targetUrl"`
	Method               string             `json:"method"`
	Rps                  int                `json:"rps"`
	DurationStr          string             `json:"duration"`
	BatchSize            int                `json:"batchSize"`
	SeverityDistribution []float64          `json:"severityDistribution"`
	ServiceNames         []string           `json:"serviceNames"`
	BodyTokens           BodyTokenFormat    `json:"bodyTokens"`
	LogAttributes        []schemas.KeyValue `json:"logAttributes"`
	ResourceAttributes   []schemas.KeyValue `json:"resourceAttributes"`
}

type CleanConfig struct {
	Seed                 int                `json:"seed"`
	PoolSize             int                `json:"poolSize"`
	HealthUrl            string             `json:"healthUrl"`
	TargetUrl            string             `json:"targetUrl"`
	Method               string             `json:"method"`
	Rps                  int                `json:"rps"`
	Duration             time.Duration      `json:"duration"`
	BatchSize            int                `json:"batchSize"`
	SeverityDistribution []float64          `json:"severityDistribution"`
	ServiceNames         []string           `json:"serviceNames"`
	BodyTokens           BodyTokenFormat    `json:"bodyTokens"`
	LogAttributes        []schemas.KeyValue `json:"logAttributes"`
	ResourceAttributes   []schemas.KeyValue `json:"resourceAttributes"`
}

func floatEquals(f1 float64, f2 float64) bool {
	const epsilon = 1e-9
	return math.Abs(f1-f2) < epsilon
}

func ValidateAndCleanConfig(rawCfg RawConfig) CleanConfig {
	const poolSizeLimit = 10000
	const poolSizeDefault = 1000
	duration, err := time.ParseDuration(rawCfg.DurationStr)
	if err != nil {
		fmt.Println("error parsing duration, defaulting to 10s")
		duration = 10 * time.Second
	}

	dist := rawCfg.SeverityDistribution
	sumProb := 0.0
	for _, p := range rawCfg.SeverityDistribution {
		sumProb += p
	}
	if len(rawCfg.SeverityDistribution) != 5 || !floatEquals(sumProb, 1.0) {
		fmt.Println("invalid severity distribution, defaulting to [0.1, 0.6, 0.1, 0.1, 0.1] for [DEBUG, INFO, WARNING, ERROR, FATAL]")
		dist = []float64{0.1, 0.6, 0.1, 0.1, 0.1}
	}

	var poolSize int
	if rawCfg.PoolSize < 0 {
		fmt.Printf("negative pool size not allowed, setting to default size: %v\n", poolSizeDefault)
		poolSize = poolSizeDefault
	}
	if rawCfg.PoolSize > poolSizeLimit {
		fmt.Printf("poolSize over limit, setting to limit of: %v\n", poolSizeLimit)
		poolSize = poolSizeLimit
	}

	return CleanConfig{
		Seed:                 rawCfg.Seed,
		PoolSize:             poolSize,
		HealthUrl:            rawCfg.HealthUrl,
		TargetUrl:            rawCfg.TargetUrl,
		Method:               rawCfg.Method,
		Rps:                  rawCfg.Rps,
		Duration:             duration,
		BatchSize:            rawCfg.BatchSize,
		SeverityDistribution: dist,
		ServiceNames:         rawCfg.ServiceNames,
		BodyTokens:           rawCfg.BodyTokens,
		LogAttributes:        rawCfg.LogAttributes,
		ResourceAttributes:   rawCfg.ResourceAttributes,
	}
}
