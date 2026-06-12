package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/Eddrick-23/Logarithm/load_generator/files"
)

type KeyValue struct {
	Key   string
	Value string
}

type BodyTokenFormat struct {
	Min        int      `json:"min"`
	Max        int      `json:"max"`
	Dictionary []string `json:"dictionary"`
}

type RawConfig struct {
	Seed                 int             `json:"seed"`
	PoolSize             int             `json:"poolSize"`
	HealthUrl            string          `json:"healthUrl"`
	TargetUrl            string          `json:"targetUrl"`
	Method               string          `json:"method"`
	Encoding             string          `json:"encoding"`
	Rps                  int             `json:"rps"`
	BatchSize            int             `json:"batchSize"`
	SeverityDistribution []float64       `json:"severityDistribution"`
	ServiceNames         []string        `json:"serviceNames"`
	BodyTokens           BodyTokenFormat `json:"bodyTokens"`
	LogAttributes        []KeyValue      `json:"logAttributes"`
	ResourceAttributes   []KeyValue      `json:"resourceAttributes"`
}

type CleanConfig struct {
	Seed                 int             `json:"seed"`
	PoolSize             int             `json:"poolSize"`
	HealthUrl            string          `json:"healthUrl"`
	TargetUrl            string          `json:"targetUrl"`
	Method               string          `json:"method"`
	Encoding             string          `json:"encoding"`
	Rps                  int             `json:"rps"`
	BatchSize            int             `json:"batchSize"`
	SeverityDistribution []float64       `json:"severityDistribution"`
	ServiceNames         []string        `json:"serviceNames"`
	BodyTokens           BodyTokenFormat `json:"bodyTokens"`
	LogAttributes        []KeyValue      `json:"logAttributes"`
	ResourceAttributes   []KeyValue      `json:"resourceAttributes"`
}

func floatEquals(f1 float64, f2 float64) bool {
	const epsilon = 1e-9
	return math.Abs(f1-f2) < epsilon
}

func cleanPoolSize(rawCfg *RawConfig) int {
	const poolSizeLimit = 1_000_000
	const poolSizeDefault = 1000

	poolSize := rawCfg.PoolSize
	if rawCfg.PoolSize < 0 {
		fmt.Printf("negative pool size not allowed, defaulting to: %v\n", poolSizeDefault)
		poolSize = poolSizeDefault
	}
	if rawCfg.PoolSize > poolSizeLimit {
		fmt.Printf("poolSize over limit, capping to: %v\n", poolSizeLimit)
		poolSize = poolSizeLimit
	}

	return poolSize
}

func cleanSeverityDistribution(rawCfg *RawConfig) []float64 {
	dist := rawCfg.SeverityDistribution
	sumProb := 0.0
	for _, p := range rawCfg.SeverityDistribution {
		sumProb += p
	}
	if len(rawCfg.SeverityDistribution) != 5 || !floatEquals(sumProb, 1.0) {
		fmt.Println("invalid severity distribution, defaulting to [0.1, 0.6, 0.1, 0.1, 0.1] for [DEBUG, INFO, WARNING, ERROR, FATAL]")
		dist = []float64{0.1, 0.6, 0.1, 0.1, 0.1}
	}
	return dist
}

func cleanEncoding(rawCfg *RawConfig) string {
	encoding := strings.ToLower(rawCfg.Encoding)

	switch encoding {
	case "":
		return "none"
	case "none":
		return encoding
	case "gzip":
		return encoding
	case "zstd":
		return encoding
	default:
		fmt.Printf("encoding %v not supported, defaulting to no encoding.\n", encoding)
		return "none"
	}
}

func validateAndCleanConfig(rawCfg RawConfig) CleanConfig {
	dist := cleanSeverityDistribution(&rawCfg)
	poolSize := cleanPoolSize(&rawCfg)
	encoding := cleanEncoding(&rawCfg)
	return CleanConfig{
		Seed:                 rawCfg.Seed,
		PoolSize:             poolSize,
		HealthUrl:            rawCfg.HealthUrl,
		TargetUrl:            rawCfg.TargetUrl,
		Method:               rawCfg.Method,
		Encoding:             encoding,
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
