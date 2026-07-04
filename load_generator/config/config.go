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
	Protocol             string          `json:"protocol"`
	GrpcWorkers          int             `json:"grpcWorkers"`
	HttpMethod           string          `json:"httpMethod"`
	HttpHealthUrl        string          `json:"httpHealthUrl"`
	TargetUrl            string          `json:"targetUrl"`
	ContentType          string          `json:"contentType"`
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
	Protocol             string          `json:"protocol"`
	GrpcWorkers          int             `json:"grpcWorkers"`
	HttpMethod           string          `json:"httpMethod"`
	HttpHealthUrl        string          `json:"httpHealthUrl"`
	TargetUrl            string          `json:"targetUrl"`
	ContentType          string          `json:"contentType"`
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
		return "identity"
	case "none":
		return encoding
	case "gzip":
		return encoding
	case "zstd":
		return encoding
	default:
		fmt.Printf("encoding %v not supported, defaulting to no encoding.\n", encoding)
		return "identity"
	}
}

func cleanContentType(rawCfg *RawConfig) string {
	ct := strings.ToLower(rawCfg.ContentType)

	switch ct {
	case "json":
		return ct
	case "proto":
		return ct
	default:
		fmt.Printf("content type %v not supported, defaulting to proto.\n", ct)
		return "proto"
	}
}

func cleanGrpcWorkerCount(rawCfg *RawConfig) int {
	const defaultCount = 5
	workers := rawCfg.GrpcWorkers

	if workers <= 0 && rawCfg.Protocol == "grpc" {
		fmt.Printf("grpc load generation requires positive worker count, defaulting to %d", defaultCount)
		return defaultCount
	}

	return workers
}

// cleans up config to ensure valid severity-distribution,
// poolSize, encoding and contentType. Invalid or missing fields
// are set to defaults
func cleanConfig(rawCfg *RawConfig) CleanConfig {
	dist := cleanSeverityDistribution(rawCfg)
	poolSize := cleanPoolSize(rawCfg)
	encoding := cleanEncoding(rawCfg)
	contentType := cleanContentType(rawCfg)
	grpcWorkers := cleanGrpcWorkerCount(rawCfg)
	return CleanConfig{
		Seed:                 rawCfg.Seed,
		PoolSize:             poolSize,
		Protocol:             rawCfg.Protocol,
		GrpcWorkers:          grpcWorkers,
		HttpMethod:           rawCfg.HttpMethod,
		HttpHealthUrl:        rawCfg.HttpHealthUrl,
		TargetUrl:            rawCfg.TargetUrl,
		ContentType:          contentType,
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

// Enforces that protocol provided is valid
// Currently only http and grpc are provided
func (r *RawConfig) validate() error {
	switch r.Protocol {
	case "http":
		if r.HttpMethod == "" {
			return fmt.Errorf("http protocol requires 'method' to be set (e.g., POST)")
		}
		if r.HttpHealthUrl == "" {
			return fmt.Errorf("http protocol requires 'healthUrl' to be set")
		}
		if r.GrpcWorkers != 0 {
			fmt.Println("Warning: 'grpc_workers is ignored when using http protocol")
		}
	case "grpc":
		if r.HttpMethod != "" {
			fmt.Println("Warning: 'http_method' is ignored when using grpc protocol")
		}
		if r.HttpHealthUrl != "" {
			fmt.Println("Warning: 'http_healthUrl' is ignored when using grpc protocol")
		}
	default:
		return fmt.Errorf("unsupported protocol: %s", r.Protocol)
	}

	return nil
}

// Parses given config from specified json file.
// Validates enforced fields and cleans up malformed fields to ensure
// compatibility with runners.
// CleanConfig must always be compatible with any interfaces that use it
// i.e. all fields are valid and present.
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

	if err = rawCfg.validate(); err != nil {
		return nil, err
	}

	cfg := cleanConfig(&rawCfg)

	return &cfg, nil
}
