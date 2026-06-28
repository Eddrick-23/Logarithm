import axios from "axios";
import type {
    ErrorRateMetrics,
    GetIngestionMetricsResponse,
    TopServiceErrorsStats,
    GetTopServiceErrorsStatsResponse,
} from "../types/Metric";
import type { StorageInfoMetrics } from "../types/Storage";

export const LOG_RATE_METRICS_REFETCH_INTERVAL_MS = 5_000; // 5 seconds
export const INGESTION_GRAPH_REFETCH_INTERVAL_MS = 5_000; // 5 seconds
export const ERROR_RATE_METRICS_REFETCH_INTERVAL_MS = 15_000; // 15 seconds
export const TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS = 30_000; // 30 seconds
export const STORAGE_INFO_METRICS_REFETCH_INTERVAL_MS = 15 * 60 * 1000; // 15 minutes

export const fetchIngestionMetrics = async (): Promise<GetIngestionMetricsResponse> => {
    // response is of the type { metrics: IngestionMetricsMap, timestamps: number[] }
    const response = await axios.get<GetIngestionMetricsResponse>("/api/ingestion-metrics");
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
