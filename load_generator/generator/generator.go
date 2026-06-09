package generator

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/mroth/weightedrand/v3"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type LogRecordTemplate struct {
	TraceId        string
	SpanId         string
	SeverityText   string
	SeverityNumber uint8
	Body           string
	LogAttributes  []config.KeyValue
}

func randInteger(customRand *rand.Rand, min int, max int) int {
	return customRand.IntN(max-min+1) + min
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
	if len(dictionary) == 0 {
		return ""
	}

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

func GenerateLogRecordPool(customRand *rand.Rand, cfg *config.CleanConfig) ([]LogRecordTemplate, error) {
	chooser, err := weightedrand.NewChooser(
		weightedrand.NewChoice("DEBUG", int(cfg.SeverityDistribution[0]*100)),
		weightedrand.NewChoice("INFO", int(cfg.SeverityDistribution[1]*100)),
		weightedrand.NewChoice("WARNING", int(cfg.SeverityDistribution[2]*100)),
		weightedrand.NewChoice("ERROR", int(cfg.SeverityDistribution[3]*100)),
		weightedrand.NewChoice("FATAL", int(cfg.SeverityDistribution[4]*100)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create weighted chooser: %w", err)
	}

	pool := make([]LogRecordTemplate, cfg.PoolSize)

	for i := 0; i < cfg.PoolSize; i++ {
		severityText := chooser.PickWith(customRand)
		severityNumber := severityTextRandNumber(customRand, severityText)

		pool[i] = LogRecordTemplate{
			SeverityText:   severityText,
			SeverityNumber: uint8(severityNumber),
			Body: randomisedBody(customRand,
				cfg.BodyTokens.Dictionary,
				cfg.BodyTokens.Min,
				cfg.BodyTokens.Max),
			LogAttributes: cfg.LogAttributes,
		}
	}
	return pool, nil
}

func GenerateSpanID(customRand *rand.Rand) pcommon.SpanID {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], customRand.Uint64())
	return pcommon.SpanID(b)
}

func GenerateTraceID(customRand *rand.Rand) pcommon.TraceID {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[:8], customRand.Uint64())
	binary.LittleEndian.PutUint64(b[8:], customRand.Uint64())

	return pcommon.TraceID(b)
}

func GenerateRequest(customRand *rand.Rand, cfg *config.CleanConfig, logRecordPool []LogRecordTemplate) plog.Logs {

	// timestamp for every record is heavy, init once per request
	now := pcommon.Timestamp(time.Now().UnixNano())

	// high level object to marshal before sending
	logs := plog.NewLogs()
	logRL := logs.ResourceLogs().AppendEmpty()

	// assign random service name for this batch
	idx := customRand.IntN(len(cfg.ServiceNames))
	serviceName := cfg.ServiceNames[idx]
	logRL.Resource().Attributes().PutStr("service.name", serviceName)

	// create scope logs that hold every log entry
	logSL := logRL.ScopeLogs().AppendEmpty()
	logSL.Scope().SetName("test-scope")
	logSL.Scope().SetVersion("1.0.0")
	for i := 0; i < cfg.BatchSize; i++ {
		logRecord := logSL.LogRecords().AppendEmpty()
		idx := customRand.IntN(len(logRecordPool))
		template := logRecordPool[idx]
		logRecord.SetTimestamp(now)
		logRecord.SetObservedTimestamp(now)

		logRecord.SetTraceID(GenerateTraceID(customRand))
		logRecord.SetSpanID(GenerateSpanID(customRand))

		logRecord.SetSeverityText(template.SeverityText)
		logRecord.SetSeverityNumber(plog.SeverityNumber(template.SeverityNumber))

		logRecord.Body().SetStr(template.Body)

		for _, attr := range template.LogAttributes {
			logRecord.Attributes().PutStr(attr.Key, attr.Value)
		}
	}

	return logs
}
