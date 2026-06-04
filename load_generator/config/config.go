package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/Eddrick-23/Logarithm/api/schemas"
	"github.com/Eddrick-23/Logarithm/load_generator/files"
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
	Gzip                 bool               `json:"gzip"`
	Rps                  int                `json:"rps"`
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
	Gzip                 bool               `json:"gzip"`
	Rps                  int                `json:"rps"`
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

func validateAndCleanConfig(rawCfg RawConfig) CleanConfig {
	const poolSizeLimit = 1_000_000
	const poolSizeDefault = 1000

	dist := rawCfg.SeverityDistribution
	sumProb := 0.0
	for _, p := range rawCfg.SeverityDistribution {
		sumProb += p
	}
	if len(rawCfg.SeverityDistribution) != 5 || !floatEquals(sumProb, 1.0) {
		fmt.Println("invalid severity distribution, defaulting to [0.1, 0.6, 0.1, 0.1, 0.1] for [DEBUG, INFO, WARNING, ERROR, FATAL]")
		dist = []float64{0.1, 0.6, 0.1, 0.1, 0.1}
	}

	poolSize := rawCfg.PoolSize
	if rawCfg.PoolSize < 0 {
		fmt.Printf("negative pool size not allowed, defaulting to: %v\n", poolSizeDefault)
		poolSize = poolSizeDefault
	}
	if rawCfg.PoolSize > poolSizeLimit {
		fmt.Printf("poolSize over limit, capping to: %v\n", poolSizeLimit)
		poolSize = poolSizeLimit
	}

	return CleanConfig{
		Seed:                 rawCfg.Seed,
		PoolSize:             poolSize,
		HealthUrl:            rawCfg.HealthUrl,
		TargetUrl:            rawCfg.TargetUrl,
		Method:               rawCfg.Method,
		Gzip:                 rawCfg.Gzip,
		Rps:                  rawCfg.Rps,
		BatchSize:            rawCfg.BatchSize,
		SeverityDistribution: dist,
		ServiceNames:         rawCfg.ServiceNames,
		BodyTokens:           rawCfg.BodyTokens,
		LogAttributes:        rawCfg.LogAttributes,
		ResourceAttributes:   rawCfg.ResourceAttributes,
	}
}

func ParseConfig(configPath string) (*CleanConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, configPath)
	exists, err := files.FileExists(path)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("config file does not exist")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawCfg RawConfig
	if err = json.Unmarshal(content, &rawCfg); err != nil {
		return nil, err
	}

	cfg := validateAndCleanConfig(rawCfg)

	return &cfg, nil
}
