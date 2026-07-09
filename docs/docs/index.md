# Welcome to MkDocs

For full documentation visit [mkdocs.org](https://www.mkdocs.org).

## Commands

* `mkdocs new [dir-name]` - Create a new project.
* `mkdocs serve` - Start the live-reloading docs server.
* `mkdocs build` - Build the documentation site.
* `mkdocs -h` - Print help message and exit.

## Project layout

    mkdocs.yml    # The configuration file.
    docs/
        index.md  # The documentation homepage.
        ...       # Other markdown pages, images and other files.

**leave everything above as reference for now**

## Project Overview

Brief explanation of what this project is about

## Installation guide

**Prerequisites**: You must have Docker installed.

* Copy the 3 files into the same directory
* Build and run using

```bash
docker compose up -d
```

* Access the dashboard at [http://localhost:5173](http://localhost:5173)

??? "docker-compose.yml"

    ```yaml title="docker-compose.yml"
    services:
    react:
        image: eddricklivando/logarithm:react
        ports:
        - "5173:80"
        environment:
        - VITE_API_URL=http://dashboard-api:8091
        depends_on:
        dashboard-api:
            condition: service_healthy

    clickhouse:
        image: clickhouse/clickhouse-server:26.3
        ports:
        - "8123:8123"
        - "9000:9000"
        volumes:
        - ./init.sql:/docker-entrypoint-initdb.d/init.sql
        - clickhouse_data:/var/lib/clickhouse
        - clickhouse_logs:/var/log/clickhouse-server
        environment:
        CLICKHOUSE_DB: default
        CLICKHOUSE_USER: admin
        CLICKHOUSE_PASSWORD: strongpassword
        CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1
        ulimits:
        nofile:
            soft: 262144
            hard: 262144
        healthcheck:
        test: ["CMD", "clickhouse-client", "--user", "admin", "--password", "strongpassword", "--query", "SELECT 1"]
        interval: 10s
        timeout: 5s
        retries: 10
        start_period: 30s

    nats:
        image: nats:2.14-alpine
        ports:
        - "4222:4222"
        - "8222:8222"
        command: ["-c", "/etc/deploy/nats.conf"]
        volumes:
        - ./nats.conf:/etc/deploy/nats.conf:ro
        - nats_storage:/data
        healthcheck:
        test: ["CMD", "wget", "-qO-", "http://localhost:8222/healthz"]
        interval: 5s
        timeout: 3s
        retries: 5
        start_period: 10s

    ingester:
        image: eddricklivando/logarithm:ingester
        ports:
        - "8090:8090"
        environment:
        - NATS_URL=nats://nats:4222
        - PORT=8090
        - DB_USER=admin
        - DB_PASSWORD=strongpassword
        - DB_ADDRESS=clickhouse:9000
        depends_on:
        nats:
            condition: service_healthy

    worker:
        image: eddricklivando/logarithm:worker
        environment:
        - NATS_URL=nats://nats:4222
        - DB_USER=admin
        - DB_PASSWORD=strongpassword
        - DB_ADDRESS=clickhouse:9000
        depends_on:
        nats:
            condition: service_healthy
        clickhouse:
            condition: service_healthy

    dashboard-api:
        image: eddricklivando/logarithm:dashboard-api
        ports:
        - "8091:8091"
        environment:
        - PORT=8091
        - NATS_URL=nats://nats:4222
        - DB_USER=admin
        - DB_PASSWORD=strongpassword
        - DB_ADDRESS=clickhouse:9000
        - SEED_SYSTEM=true
        healthcheck:
        test: ["CMD", "wget", "-qO-", "http://dashboard-api:8091/health"]
        interval: 10s
        timeout: 5s
        retries: 5
        start_period: 30s
        depends_on:
        nats:
            condition: service_healthy
        clickhouse:
            condition: service_healthy

    volumes:
    clickhouse_data:
    clickhouse_logs:
    nats_storage:
    ```

??? "nats.conf"

    ```ini title="nats.conf"

    listen: 0.0.0.0:4222
    http_port: 8222

    # Client connection settings
    max_connections: 100000
    max_payload: 1MB # default
    max_pending: 256MB
    write_deadline: 10s

    #jetstream config
    jetstream {
        # Storage directory for streams
        store_dir: /data/jetstream

        # Memory and storage limits
        max_memory_store: 4GB
        max_file_store: 100GB

        # Sync interval for file storage
        # Lower values increase durability but reduce throughput
        sync_interval: "2m"
    }

    # Logging
    debug: false
    trace: false
    logtime: true
    ```

??? "init.sql"

    ```sql title="init.sql"
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
    ```

## QuickStart

This is a short tutorial on connecting a service to logarithm itself, and sending some sample logs.

**Prerequisites**: You must have logarithm running locally on your machine via docker. See [installation-guide](#installation-guide)

* dockerised http server maybe using python fastapi
* setup otlp into loggers and point directly to logarithm or to an otel collector
* send curl requests to force log events and view dashboard
