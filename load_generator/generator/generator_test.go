package generator

import (
	"math/rand/v2"
	"testing"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
)

func BenchmarkGenerateRequest(b *testing.B) {
	cfg := config.CleanConfig{
		Seed:                 1,
		PoolSize:             5000,
		GrpcWorkers:          1,
		HttpMethod:           "",
		HttpHealthUrl:        "",
		TargetUrl:            "",
		Rps:                  1,
		BatchSize:            500,
		SeverityDistribution: []float64{0.1, 0.6, 0.1, 0.1, 0.1},
		ServiceNames:         []string{"testservice1", "testservice2", "testservice3"},
		BodyTokens: config.BodyTokenFormat{
			Min: 15,
			Max: 25,
			Dictionary: []string{
				"failed",
				"process",
				"transaction",
				"invalid",
				"account",
				"timeout",
				"database",
				"connection",
				"lost",
				"retry",
				"success",
				"user",
				"authenticated",
				"payload",
				"too",
				"large",
			},
		},
		ResourceAttributes: []config.KeyValue{
			{Key: "host.name", Value: "prod-payment-02"},
			{Key: "environment", Value: "production"},
		},
		LogAttributes: []config.KeyValue{
			{Key: "http.method", Value: "POST"},
		},
	}
	customRand := rand.New(rand.NewPCG(uint64(cfg.Seed), 2))
	pool, err := GenerateLogRecordPool(customRand, &cfg)

	if err != nil {
		b.Fatalf("failed to generate record pool: %v", err)
	}

	for b.Loop() {
		GenerateRequest(customRand, &cfg, pool)
	}
}
