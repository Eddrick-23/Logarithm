import axios from "axios";
import type { GetIngestionMetricsResponse, ErrorMetrics, GetErrorMetricsResponse } from "../types/Metric";

export const fetchIngestionMetrics = async (): Promise<GetIngestionMetricsResponse> => {
    // response is of the type { metrics: IngestionMetricsMap, timestamps: number[] }
    const response = await axios.get<GetIngestionMetricsResponse>("/api/ingestion-metrics");
    return response.data;
};

export const fetchErrorMetrics = async (): Promise<ErrorMetrics[]> => {
    // response is of the type { data: ErrorMetrics[] }
    const response = await axios.get<GetErrorMetricsResponse>("/api/error-metrics");
    return response.data.data;
};
