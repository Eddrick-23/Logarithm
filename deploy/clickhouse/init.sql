CREATE DATABASE IF NOT EXISTS logarithm;

CREATE TABLE IF NOT EXISTS logarithm.logs (
    Timestamp DateTime64(9, 'UTC') CODEC(DoubleDelta, LZ4),
    InsertedAt DateTime DEFAULT now() CODEC(DoubleDelta, LZ4),
    
    -- OTLP Tracing Data
    TraceId FixedString(32) CODEC(ZSTD(1)),
    SpanId FixedString(16) CODEC(ZSTD(1)),
    
    -- OTLP Log Record Data
    SeverityText LowCardinality(String),
    SeverityNumber UInt8,
    ServiceName LowCardinality(String),
    Body String CODEC(ZSTD(3)),
    
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
