# Configuration

This document describes all environment variables used to configure the frontend and backend services. Variables are grouped into three categories:

- **Basic** — core connectivity settings (hosts, ports, addresses) needed to get the system running.
- **Logging Levels** — variables that control log verbosity of each service.
- **Advanced** — infrastructure tuning (timeouts, batch sizes, intervals, retention, etc.) for performance and reliability

## Basic

### Frontend (`frontend/.env`)

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `VITE_API_URL` | Base URL of backend REST API | `http://dashboard-api:8091` | Yes |
| `VITE_WEBSOCKET_URL` | WebSocket endpoint for real-time updates | `ws://dashboard-api:8091` | Yes |

> **Note:** All frontend env vars must be prefixed with `VITE_` to be exposed to client-side code. Anything without the prefix won't be injected at build time.  Do not put secrets here, since these are bundled into public Javascript.

### Backend (`backend/.env`)

This backend is composed of several services (Ingester, Dashboard API, Worker) sharing a ClickHouse database and NATS message broker.

#### Ingester

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `INGESTER_HOST` | Bind address for the ingester service | `0.0.0.0` | Yes |
| `INGESTER_PORT_GRPC` | Port for gRPC ingestion endpoint | `8089` | Yes |
| `INGESTER_PORT_HTTP` | Port for HTTP ingestion endpoint | `8090` | Yes |

---

#### Dashboard API

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `APP_HOST` | Bind address / service name for the dashboard API | `dashboard-api` | Yes |
| `APP_PORT` | Port the dashboard API listens on | `8091` | Yes |

> Corresponds to frontend `VITE_API_URL` / `VITE_WEBSOCKET_URL`, which should point at `APP_HOST:APP_PORT`.

---

#### ClickHouse (Database)

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `DB_ADDRESS` | ClickHouse connection address | `clickhouse:9000` | Yes |
| `DB_USER` | ClickHouse username | `admin` | Yes |
| `DB_PASSWORD` | ClickHouse password | `strongpassword` | Yes |
| `DB_NAME` | Database name | `logarithm` | Yes |

---

#### NATS (Message Broker)

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `NATS_URL` | NATS server connection URL | `nats://nats:4222` | Yes |

## Logging Levels

Variables that control log verbosity.

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `INGESTER_LOG_LEVEL` | Log verbosity for ingester | `INFO` | Yes |
| `DASHBOARD_LOG_LEVEL` | Log verbosity for dashboard | `INFO` | Yes |
| `WORKER_LOG_LEVEL` | Log verbosity for worker | `INFO` | Yes |

## Live Tail Streaming

Logarithm uses smart streaming under the hood. It avoids publishing to live tail if there are no existing live tail connections on the frontend. This is controlled via a heartbeat checking pipeline betweeen the dashbord api and the worker.

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `LIVE_TAIL_PRESENCE_INTERVAL` | How often to send ping messages to nats to signify active subscriptions. | `500ms` | No(default `1s`) |
| `WORKER_LIVE_TAIL_PRESENCE_TIMEOUT` | How new the last ping message has to be for it to be considered an active subscriber. | `1s` | No(default `1s`) |

!!! tip "Note on `LIVE_TAIL_PRESENCE_INTERVAL`"
    Lowering this value increases responsiveness but results in higher NATS traffic

!!! warning "Timeout should be kept larger than interval"
    It is best to keep timeout larger than interval to avoid skipping intermediate logs due to toggling between active and no-active subscribers when publishing rate is too low.

## Advanced

Infrastructure tuning variables such as timeouts, batch sizes, intervals and retention settings. Misconfiguring these variables can affect throughput, latency, or cause dropped / stalled data, so changes should be tested before rolling out to production.

### Backend (`backend/.env`)

#### Ingester

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `INGESTER_READ_HEADER_TIMEOUT` | Max time to read request headers | `2s` | No (default: `2s`) |
| `INGESTER_READ_TIMEOUT` | Max time to read full request | `5s` | No (default: `5s`) |
| `INGESTER_WRITE_TIMEOUT` | Max time to write response | `10s` | No (default: `10s`) |
| `INGESTER_IDLE_TIMEOUT` | Max idle time for keep-alive connections | `60s` | No (default: `60s`) |
| `INGESTER_PRESIZE_BUFFER` | Presize buffer to expected payload sizes to avoid expensive slice growths. | `4kb` | No (default: `4kb`) |
| `INGESTER_BUFFER_LIMIT` | Restrict maximum buffer size in the pool. Buffers that grow beyond this size are not returned to pool. | `20kb` | No (default: `20kb`) |

!!! Info
    - The ingester handlers use go's sync.Pool internally to minimise allocations on the hot path. `INGESTER_PRESIZE_BUFFER` can be raised to if expected incoming payloads are much higher than the default.
    - If payload sizes vary largely, setting an upper limit via `INGESTER_BUFFER_LIMIT` will help prevent holding on to large allocated buffers that are not used fully.

!!! warning "`INGESTER_BUFFER_LIMIT` must be >= `INGESTER_PRESIZE_BUFFER`"
    This is required for efficient buffer reuse. Else the allocated buffer will never be returned to pool resulting in a fresh allocation per request.

---

#### Dashboard API

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `LIVE_TAIL_REFRESH_INTERVAL` | Live tail poll/refresh interval (ms) | `500` | Yes |
| `LIVE_TAIL_MAX_BATCH` | Max number of log lines sent per live tail batch | `100` | Yes |

---

#### ClickHouse (Database)

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `DB_BATCH_POOL_SIZE` | Number of concurrent batch insert workers | `5` | Yes |
| `DB_BATCH_POOL_MAX_ROWS` | Max rows per batch insert | `2000` | Yes |

!!! warning "`DB_BATCH_POOL_MAX_ROWS` must be >= `WORKER_ROWS_PER_BATCH`"
    Not abiding by this may cause batches to be split/dropped unexpectedly

---

#### Worker

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `WORKER_LOG_LEVEL` | Log verbosity | `INFO` | Yes |
| `WORKER_MAX_BATCH` | Max messages pulled per batch from NATS | `1000` | Yes |
| `WORKER_MAX_WAIT` | Max time to wait before flushing a partial batch | `2s` | Yes |
| `WORKER_BACKOFF` | Retry backoff schedule on failure (comma-separated durations) | `5s,30s,60s,300s,3600s` | Yes |
| `WORKER_COUNT` | Number of worker instances/goroutines to run | `1` | Yes |
| `WORKER_LIVE_TAIL_COUNT` | Number of concurrent live tail subscriptions supported | `3` | Yes |
| `WORKER_LIVE_TAIL_QUEUE_SIZE` | Max buffered messages per live tail subscription | `10000` | Yes |
| `WORKER_ROWS_PER_BATCH` | Rows written to ClickHouse per insert batch | `1000` | Yes |

---

#### NATS (Message Broker)

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `NATS_STREAM_MAX_AGE` | Max retention age for the main log stream | `12h` | Yes |
| `NATS_DLQ_MAX_AGE` | Max retention age for dead-letter queue messages | `24h` | Yes |
| `NATS_MAX_DELIVER` | Max delivery attempts before a message is sent to DLQ | `20` | Yes |
| `NATS_BACKOFF` | Redelivery backoff schedule (comma-separated durations) | `5s,30s,60s,300s,3600s` | Yes |
| `NATS_LOG_STREAM_MAX_BYTES` | Max size of the main log stream before eviction | `50GB` | Yes |
| `NATS_DLQ_MAX_BYTES` | Max size of the dead-letter queue stream | `10GB` | Yes |
| `NATS_CONSUMER_MAX_ACK_PENDING` | Max unacknowledged messages per consumer | `2000` | Yes |

!!! warning "`NATS_CONSUMER_MAX_ACK_PENDING` must be >= `WORKER_MAX_BATCH`"
    This is to prevent the worker from stalling while waiting for more messages despite ack headroom being reached.
  

---

#### Profiling & Benchmarking

| Variable | Description | Example | Required |
| --- | --- | --- | --- |
| `ENABLE_PPROF` | Enables Go pprof profiling endpoint | `false` | No (default: `false`) |
| `PPROF_HOST` | Bind address for the pprof server | `0.0.0.0` | No (default: `0.0.0.0`) |

!!! warning
    Do not enable `ENABLE_PPROF` in production without restricting access since pprof endpoints can leak memory contents and are a security risk if publicly exposed.

## Basic

- basic environment variables.
- these include host and ports

## Logging Levels

- env variables to tune logging levels for each service
- Currently only worker has tunable log level
- Will eventually add logging levels for ingester and dashboard api

## Advanced

- advanced variables. These are for infrastructure tuning
- timeouts
- batch sizes
- intervals etc
