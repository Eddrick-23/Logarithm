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
