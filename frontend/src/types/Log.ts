export type LogType = "debug" | "info" | "warning" | "error";

export type KeyValue = {
    key: string;
    value: string;
};

export type LogRecord = {
    timestamp: Date;
    traceId: string;
    spanId: string;
    severityText: string;
    severityNumber: number;
    body: string;
    logAttributes: KeyValue[];
    resourceAttributes: KeyValue[];
};

export type LogRecordDTO = {
    timestamp: string;
    traceId: string;
    spanId: string;
    severityText: LogType;
    severityNumber: number;
    body: string;
    logAttributes: KeyValue[];
};
export type LogIngestRequest = {
    serviceName: string;
    resourceAttributes: KeyValue[];
    records: LogRecordDTO[];
};

export type FlatLogEntry = {
    serviceName: string;
    log: LogRecordDTO;
};
