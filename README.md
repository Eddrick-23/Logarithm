# Logarithm
Logarithm is a high-performance, log ingestion and self-hosted observability pipeline designed for modern microservice architectures.

## Tech Stack

* **Backend:** Go (Golang)
* **Message Broker:** NATS
* **Database / Storage:** ClickHouse
* **Infrastructure:** Docker & Docker Compose
* **Frontend:** React 


## Project Structure
```text
.
├── backend/                # Go microservices (ingester, worker, dashboard)
│   ├── cmd/                # Entrypoints for the microservices
│   ├── internal/           # Core logic, transport, storage, and routing
│   └── tests/              # Integration tests
├── frontend/               # User interface and client-side code
├── deploy/                 # Configuration files for infrastructure
│   ├── clickhouse/         # ClickHouse init scripts (e.g., init.sql)
│   └── nats/               # NATS server configurations
├── docs/                   # documentation files
├── docker-compose.yaml     # Local development environment orchestration
└── README.md
```

### Setup and Run containers
```
docker compose up -d
```

### stop containers
```
docker compose down
```

### Running integration tests
Integration tests for the backend are available at backend/tests/integration.
To run them, use
```
cd backend/tests/integration
go test
```
