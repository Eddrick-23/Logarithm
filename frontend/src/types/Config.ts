export type Config = {
    liveTailRefreshInterval: number;
    liveTailMaxBatch: number;
    natsStreamMaxAge: string;
    natsDLQMaxAge: string;
    natsMaxDeliver: number;
    natsBackoff: string;
    natsLogStreamMaxBytes: string;
    natsDLQMaxBytes: string;
    natsConsumerMaxAckPending: number;
    workerLogLevel: string;
    workerMaxBatch: number;
    workerBackoff: string;
    workerRowsPerBatch: number;
};

export type ConfigItem = {
    key: keyof Config;
    label: string;
    description: string;
    value: string | number;
};
