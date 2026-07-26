import axios from "axios";
import type {
    ErrorRateMetrics,
    TopServiceErrorsStats,
    GetTopServiceErrorsStatsResponse,
    IngestionGraphData,
    LogRateStatistics,
    NatsQueueDepthGraphMetrics,
    NatsDLQMetrics,
} from "../types/Metric";
import type { StorageInfoMetrics } from "../types/Storage";

export const LOG_RATE_METRICS_REFETCH_INTERVAL_MS = 5_000; // 5 seconds
export const INGESTION_GRAPH_REFETCH_INTERVAL_MS = 5_000; // 5 seconds
export const ERROR_RATE_METRICS_REFETCH_INTERVAL_MS = 15_000; // 15 seconds
export const TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS = 30_000; // 30 seconds
export const NATS_DLQ_INFO_REFETCH_INTERVAL_MS = 1 * 60 * 1000; // 1 minute
export const STORAGE_INFO_METRICS_REFETCH_INTERVAL_MS = 15 * 60 * 1000; // 15 minutes

export const fetchIngestionGraphMetrics = async (): Promise<IngestionGraphData> => {
    // response is of the type { metrics: IngestionMetricsMap, timestamps: number[] }
    const response = await axios.get<IngestionGraphData>("/api/ingestion-graph-metrics");
    return response.data;
};

export const fetchLogRateStats = async (): Promise<LogRateStatistics> => {
    const response = await axios.get<LogRateStatistics>("/api/log-rate-stats");
    return response.data;
};

export const fetchErrorRateMetrics = async (): Promise<ErrorRateMetrics> => {
    const response = await axios.get<ErrorRateMetrics>("/api/error-rate-metrics");
    return response.data;
};

export const fetchTopServiceErrorsStats = async (): Promise<TopServiceErrorsStats[]> => {
    // response is of the type { data: TopServiceErrorsStats[] }
    const response = await axios.get<GetTopServiceErrorsStatsResponse>("/api/top-service-errors");
    return response.data.data;
};

export const fetchStorageInfoMetrics = async (): Promise<StorageInfoMetrics> => {
    const response = await axios.get<StorageInfoMetrics>("/api/storage-info");
    return response.data;
};

export const fetchNatsQueueDepthGraphMetrics = async (): Promise<NatsQueueDepthGraphMetrics> => {
    const response = await axios.get<NatsQueueDepthGraphMetrics>("/api/nats-queue-depth-metrics");
    return response.data;
};

export const fetchNatsDLQMetrics = async(): Promise<NatsDLQMetrics> => {
    const response = await axios.get<NatsDLQMetrics>("/api/nats-dlq-info");
    return response.data;
}
