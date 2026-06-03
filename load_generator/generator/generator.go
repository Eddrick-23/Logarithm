package generator

import (
	"encoding/binary"
	"encoding/hex"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/Eddrick-23/Logarithm/api/schemas"
	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/mroth/weightedrand/v3"
)

func randInteger(customRand *rand.Rand, min int, max int) int {
	result := customRand.IntN(max - min + 1)
	return result
}

func severityTextRandNumber(customRand *rand.Rand, text string) int {
	switch text {
	case "DEBUG":
		return randInteger(customRand, 5, 8)
	case "INFO":
		return randInteger(customRand, 9, 12)
	case "WARNING":
		return randInteger(customRand, 13, 16)
	case "ERROR":
		return randInteger(customRand, 17, 20)
	case "FATAL":
		return randInteger(customRand, 21, 24)
	}
	return 0
}

func randomisedBody(customRand *rand.Rand, dictionary []string, min int, max int) string {
	length := randInteger(customRand, min, max)
	var result strings.Builder
	for i := 0; i < length; i++ {
		idx := randInteger(customRand, 0, len(dictionary)-1)
		result.WriteString(dictionary[idx])
		if i != length-1 {
			result.WriteRune(' ')
		}
	}

	return result.String()
}

func GenerateLogRecordPool(customRand *rand.Rand, cfg config.CleanConfig) []schemas.LogRecordDTO {
	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice("DEBUG", int(cfg.SeverityDistribution[0]*100)),
		weightedrand.NewChoice("INFO", int(cfg.SeverityDistribution[1]*100)),
		weightedrand.NewChoice("WARNING", int(cfg.SeverityDistribution[2]*100)),
		weightedrand.NewChoice("ERROR", int(cfg.SeverityDistribution[3]*100)),
		weightedrand.NewChoice("FATAL", int(cfg.SeverityDistribution[4]*100)),
	)

	pool := make([]schemas.LogRecordDTO, cfg.PoolSize)

	for i := 0; i < cfg.PoolSize; i++ {
		severityText := chooser.PickWith(customRand)
		severityNumber := severityTextRandNumber(customRand, severityText)

		pool[i] = schemas.LogRecordDTO{
			SeverityText:   severityText,
			SeverityNumber: uint8(severityNumber),
			Body: randomisedBody(customRand,
				cfg.BodyTokens.Dictionary,
				cfg.BodyTokens.Min,
				cfg.BodyTokens.Max),
			LogAttributes: cfg.LogAttributes,
		}
	}
	return pool
}

func GenerateSpanID(customRand *rand.Rand) string {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], customRand.Uint64())
	id := hex.EncodeToString(b[:])

	return id
}

func GenerateTraceID(customRand *rand.Rand) string {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[:8], customRand.Uint64())
	binary.LittleEndian.PutUint64(b[8:], customRand.Uint64())
	id := hex.EncodeToString(b[:])

	return id
}

func GenerateRequest(customRand *rand.Rand, cfg config.CleanConfig, logRecordPool []schemas.LogRecordDTO) schemas.LogIngestRequest {
	records := make([]schemas.LogRecordDTO, cfg.BatchSize)
	// timestamp for every record is heavy and nanosecond granularity anyway
	timestamp := time.Now().UTC()
	for i := 0; i < cfg.BatchSize; i++ {
		idx := customRand.IntN(len(logRecordPool))
		record := logRecordPool[idx]
		record.TraceId = GenerateTraceID(customRand)
		record.SpanId = GenerateSpanID(customRand)
		record.Timestamp = timestamp
		records[i] = record
	}

	idx := customRand.IntN(len(cfg.ServiceNames))
	serviceName := cfg.ServiceNames[idx]
	return schemas.LogIngestRequest{
		ServiceName:        serviceName,
		ResourceAttributes: cfg.ResourceAttributes,
		Records:            records,
	}

}
