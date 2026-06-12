import axios from "axios";
import type { IngestionMetricsMap, GetIngestionMetricsResponse, ErrorMetrics } from "../types/Metric";

export const fetchIngestionMetrics = async (): Promise<IngestionMetricsMap> => {
    // response is of the type { metrics: IngestionMetricsMap }
    const response = await axios.get<GetIngestionMetricsResponse>("/api/ingestion-metrics");
    return response.data.metrics;
};

export const fetchErrorMetrics = async (): Promise<ErrorMetrics[]> => {
    const response = await axios.get<ErrorMetrics[]>("/api/error-metrics");
    return response.data;
};
