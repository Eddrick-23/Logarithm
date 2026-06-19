import axios from "axios";
import type {
    ErrorRateMetrics,
    GetIngestionMetricsResponse,
    TopServiceErrorsStats,
    GetTopServiceErrorsStatsResponse,
} from "../types/Metric";

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
