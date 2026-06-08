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

export type FlatLogRecord = {
    timestamp: string;
    observedTimestamp: string;
    insertedAt: string;
    traceId: string;
    spanId: string;
    severityText: string;
    severityNumber: number;
    serviceName: string;
    body: string;
    bodyType: string;
    scopeName: string;
    scopeVersion: string;
    logAttrKeys: string[];
    logAttrValues: string[];
    resAttrKeys: string[];
    resAttrValues: string[];
};
