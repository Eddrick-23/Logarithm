export type LogRecord = {
    timestamp: Date;
    traceId: string;
    spanId: string;
    severityText: string;
    severityNumber: number;
    body: string;
    logAttributes: Object[];
};
