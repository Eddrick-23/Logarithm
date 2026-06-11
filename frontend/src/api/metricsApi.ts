import axios from "axios";
import type { IngestionMetricsMap, GetIngestionMetricsResponse } from "../types/Metric";

export const fetchMetrics = async (): Promise<IngestionMetricsMap> => {
    // response is of the type { metrics: IngestionMetricsMap }
    const response = await axios.get<GetIngestionMetricsResponse>("/api/ingestion-metrics");
    return response.data.metrics;
};
