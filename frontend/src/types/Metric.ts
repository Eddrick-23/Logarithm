export type IngestionMetrics = {
    timestamp: string;
    serviceName: string;
    logsCount: number;
};

// serviceName: IngestionMetrics[]
export type IngestionMetricsMap = Record<string, IngestionMetrics[]>;

export type IngestionGraphData = {
    metrics: IngestionMetricsMap;
    timestamps: number[]; // API returns ISO time numbers
};

export type LogRateStatistics = {
    currentRate: number;
    avgRate: number;
    ratio: number;
};

export type ErrorRateMetrics = {
    currentRate: number;
};

export type TopServiceErrorsStats = {
    serviceName: string;
    totalErrors: number;
    errorRate: number; // the API returns a float
};

export type GetTopServiceErrorsStatsResponse = {
    data: TopServiceErrorsStats[];
};

export type NatsQueueDepthGraphMetrics = {
    timestamps: number[];
    numPending: number[];
    numAckPending: number[];
    numRedelivered: number[];
};
