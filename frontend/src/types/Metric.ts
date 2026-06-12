export type IngestionMetrics = {
    timestamp: string;
    serviceName: string;
    logsCount: number;
};

// serviceName: IngestionMetrics[]
export type IngestionMetricsMap = Record<string, IngestionMetrics[]>;

export type GetIngestionMetricsResponse = {
    metrics: IngestionMetricsMap;
};

export type ErrorMetrics = {
    serviceName: string;
    totalErrors: number;
    errorRate: number; // the API returns a float
};
