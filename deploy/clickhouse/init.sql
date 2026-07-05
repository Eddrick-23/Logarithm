CREATE DATABASE IF NOT EXISTS logarithm;

CREATE TABLE IF NOT EXISTS logarithm.logs (
    Timestamp DateTime64(9, 'UTC') CODEC(DoubleDelta, LZ4),
    InsertedAt DateTime DEFAULT now() CODEC(DoubleDelta, LZ4),

    ScopeName  LowCardinality(String),
    ScopeVersion LowCardinality(String),
    
    -- OTLP Tracing Data
    TraceId FixedString(32) CODEC(ZSTD(1)),
    SpanId FixedString(16) CODEC(ZSTD(1)),
    
    -- OTLP Log Record Data
    ObservedTimestamp DateTime64(9, 'UTC') CODEC(DoubleDelta, LZ4),
    SeverityText LowCardinality(String),
    SeverityNumber UInt8,
    ServiceName LowCardinality(String),
    Body String CODEC(ZSTD(3)),
    BodyType LowCardinality(String),
    
    -- Flattened flexible attributes
    LogAttrKeys Array(String),
    LogAttrValues Array(String),
    ResAttrKeys Array(String),
    ResAttrValues Array(String),

    INDEX idx_inserted_at InsertedAt TYPE minmax GRANULARITY 2
) 
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(Timestamp)
ORDER BY (ServiceName, Timestamp, SeverityNumber)

-- Auto-delete old logs to save disk space
TTL Timestamp + INTERVAL 30 DAY;

-- metrics stored in 1 second buckets
CREATE TABLE IF NOT EXISTS logarithm.metrics (
    Timestamp DateTime('UTC'),
    ServiceName LowCardinality(String),
    LogsCount UInt64, -- stores total number of logs received in 1 second
    ErrorsCount UInt64 -- stores total number of errors received in 1 second
)
ENGINE = SummingMergeTree()
PARTITION BY toStartOfHour(Timestamp)
ORDER BY (ServiceName, Timestamp)

-- Auto-delete old log metrics to save disk space
TTL Timestamp + INTERVAL 1 HOUR;

CREATE MATERIALIZED VIEW IF NOT EXISTS logarithm.metrics_mv 
TO logarithm.metrics -- stores data into metrics
AS
SELECT
    -- get columns required for metrics
    -- all logs with the same start second will be grouped together
    toStartOfSecond(Timestamp) AS Timestamp,
    ServiceName,
    count() AS LogsCount,
    countIf(SeverityNumber >= 17) AS ErrorsCount -- ErrorsCount includes ERROR (17-20) and FATAL (21-24)
FROM logarithm.logs
GROUP BY ServiceName, Timestamp;

-- metrics stored in 1 minute buckets
CREATE TABLE IF NOT EXISTS logarithm.metrics_1m (
    Timestamp DateTime('UTC'),
    ServiceName LowCardinality(String),
    LogsCount UInt64,
    ErrorsCount UInt64,
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(Timestamp)
ORDER BY (ServiceName, Timestamp)

-- Auto-delete old log metrics to save disk space
TTL Timestamp + INTERVAL 1 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS logarithm.metrics_1m_mv
TO logarithm.metrics_1m
AS
SELECT
    -- all logs with the same start minute will be grouped together
    toStartOfMinute(Timestamp) AS Timestamp,
    ServiceName,
    sum(LogsCount) AS LogsCount,
    sum(ErrorsCount) AS ErrorsCount
FROM logarithm.metrics
GROUP BY ServiceName, Timestamp;

CREATE TABLE IF NOT EXISTS logarithm.service_registry (
    ServiceName LowCardinality(String),
    FirstSeen DateTime('UTC'),
    LastSeen DateTime('UTC') -- records last seen to see when was the service last active
)
ENGINE = ReplacingMergeTree(LastSeen)
ORDER BY ServiceName;

CREATE MATERIALIZED VIEW IF NOT EXISTS logarithm.service_registry_mv
TO logarithm.service_registry
AS
SELECT
    ServiceName,
    min(Timestamp) AS FirstSeen,
    max(Timestamp) AS LastSeen
FROM logarithm.logs
GROUP BY ServiceName;

CREATE TABLE IF NOT EXISTS logarithm.jetstream_consumer_metrics (
    Timestamp DateTime('UTC'),
    ConsumerName LowCardinality(String),
    StreamName LowCardinality(String),
    NumAckPending UInt64,
    NumRedelivered UInt64,
    NumPending UInt64,
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(Timestamp)
ORDER BY (StreamName, ConsumerName, Timestamp)

-- Auto-delete old jetstream consumer metrics to save disk space
TTL Timestamp + INTERVAL 1 HOUR;