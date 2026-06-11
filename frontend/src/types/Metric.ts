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
