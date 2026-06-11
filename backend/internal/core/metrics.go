package core

import "time"

type IngestionMetrics struct {
	Timestamp   time.Time `ch:"Timestamp" json:"timestamp"`
	ServiceName string    `ch:"ServiceName" json:"serviceName"`
	LogsCount   uint64    `ch:"LogsCount" json:"logsCount"`
}

type IngestionMetricsMap map[string][]IngestionMetrics

type IngestionMetricsResponse struct {
	Metrics IngestionMetricsMap `json:"metrics"`
}

func NewIngestionMetricsMap(rows []IngestionMetrics) IngestionMetricsMap {
	result := make(IngestionMetricsMap)
	for _, row := range rows {
		result[row.ServiceName] = append(result[row.ServiceName], row)
	}
	return result
}
