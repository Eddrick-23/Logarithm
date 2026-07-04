package core

import (
	"time"
)

type IngestionMetrics struct {
	Timestamp   time.Time `ch:"Timestamp" json:"timestamp"`
	ServiceName string    `ch:"ServiceName" json:"serviceName"`
	LogsCount   uint64    `ch:"LogsCount" json:"logsCount"`
}

type IngestionMetricsMap map[string][]IngestionMetrics

type IngestionGraphMetrics struct {
	Timestamps []int64             `json:"timestamps"`
	Metrics    IngestionMetricsMap `json:"metrics"`
}

type TopServiceErrorsStats struct {
	ServiceName string  `ch:"ServiceName" json:"serviceName"`
	TotalErrors uint64  `ch:"TotalErrors" json:"totalErrors"`
	ErrorRate   float64 `ch:"ErrorRate" json:"errorRate"`
}

type TopServiceErrorsStatsResponse struct {
	Data []TopServiceErrorsStats `json:"data"`
}

type LogRateStatistics struct {
	CurrentRate float64 `ch:"CurrentRate" json:"currentRate"`
	AvgRate     float64 `ch:"AvgRate" json:"avgRate"`
	Ratio       float64 `ch:"Ratio" json:"ratio"`
}

type ErrorRateMetrics struct {
	CurrentRate float64 `ch:"CurrentRate" json:"currentRate"`
}

type NatsQueueDepthGraphMetrics struct {
	Timestamps    []int64  `json:"timestamps"`
	NumPending    []uint64 `json:"numPending"`
	NumAckPending []uint64 `json:"numAckPending"`
}

func NewIngestionMetricsMap(rows []IngestionMetrics) IngestionMetricsMap {
	result := make(IngestionMetricsMap)
	for _, row := range rows {
		if row.ServiceName != "" {
			result[row.ServiceName] = append(result[row.ServiceName], row)
		}
	}
	return result
}

func NewIngestionGraphMetrics(rows []IngestionMetrics, numTimestamps int) IngestionGraphMetrics {
	if len(rows) == 0 {
		return IngestionGraphMetrics{}
	}

	timestamps := make([]int64, numTimestamps)
	// copy over the timestamps to return to the frontend, made possible due to backfilling
	for i := range numTimestamps {
		timestamps[i] = rows[i].Timestamp.UnixMilli() // convert to numbers so MUI X time scale works
	}

	ingestionMetricsMap := NewIngestionMetricsMap(rows)
	return IngestionGraphMetrics{Timestamps: timestamps, Metrics: ingestionMetricsMap}
}
