# Logarithm

Logarithm is a high-performance, log ingestion and self-hosted observability pipeline designed for modern microservice architectures.

**Full documentation** : [https://eddrick-23.github.io/Logarithm/](https://eddrick-23.github.io/Logarithm/)

## Tech Stack

- **Backend:** Go (Golang)
- **Message Broker:** NATS
- **Database / Storage:** ClickHouse
- **Infrastructure:** Docker & Docker Compose
- **Frontend:** React

## Project Structure

```text
.
├── backend/                    # Go microservices (ingester, worker, dashboard)
│   ├── cmd/                    # Entrypoints for the microservices
│   ├── internal/               # Core logic, transport, storage, and routing
│   └── tests/                  # Integration tests
├── frontend/                   # User interface and client-side code
│   └── src/
│       ├── api/                # API client functions for backend requests
│       ├── components/         # Reusable UI components shared across pages
│       ├── config/             # Environment variables initialisation
│       ├── hooks/              # TanStack Query hooks for data fetching and caching
│       ├── pages/              # Top-level page components mapped to routes
│       ├── routes/             # Route definitions and navigation configuration
│       ├── tests/              # Frontend tests
│       │   └── integration/    # Integration tests
│       │   └── unit/           # Unit tests
│       ├── theme/              # MUI theme configurations and style overrides
│       ├── types/              # Typescript type and interface definitions
│       ├── utils/              # Shared helper functions and utilities
│       └── workers/            # Web Workers for background processing
├── deploy/                     # Configuration files for infrastructure
│   ├── clickhouse/             # ClickHouse init scripts (e.g., init.sql)
│   └── nats/                   # NATS server configurations
├── docs/                       # documentation files
├── docker-compose.yaml         # Local development environment orchestration
└── README.md
```

## Prerequisites

Before you begin, ensure you have met the following requirements:

- **Docker**: Install the latest version of [Docker Desktop](https://docker.com).

## Getting Started

### Clone the repository

```bash
git clone https://github.com/Eddrick-23/Logarithm.git
cd Logarithm
```

### Running with Docker

Build and start all services (frontend + backend) with a single command:

```bash
docker compose up --build
```

This will spin up:

- **React** - accessible at <http://localhost:5173>
- **Go Dashboard API** - accessible at <http://localhost:8091>
- **Live Tail Logs Websocket** - accessible at ws://localhost:8091/ws/logs/tail
- **ClickHouse DB** - accessible at <http://localhost:8123>
- **Ingester API** - accessible at <http://localhost:8090>
- **NATS** - accessible at <http://localhost:8222>

To run in detached mode (background) mode:

```bash
docker compose up --build -d
```

To watch services

```bash
docker compose up --watch
```

To stop all services:

```bash
docker compose down
```

To force a rebuild from prod to dev

```bash
docker compose down
docker compose build --no-cache
docker compose up --watch
```

To force a rebuild from dev to prod

```bash
docker compose down
docker compose -f docker-compose.yml build --no-cache
docker compose -f docker-compose.yml up
```

### Running tests

To run unit tests, use

```bash
cd backend
go test ./...
```

Integration tests for the backend are available at backend/tests/integration and are tagged under `integration`
To run them, use

```bash
cd backend
go test ./... -tags=integration
```

To run frontend tests

```bash
cd frontend
npm test
```

To run frontend tests with Vitest UI

```bash
cd frontend
npm run test:ui
```

To run frontend tests with watch mode

```bash
cd frontend
npm run test:watch
```

### [Load generator and benchmarks](benchmarks/benchmarks.md)

Benchmarking logarithm uses a seeded randomised load generator using go and the vegeta library

#### Usage

```bash
cd load_generator

go build -o bin/main
bin/main --config config.json --duration 10 --warmup 5
```

To view flags use `bin/main --help` or `go run main.go --help`
Results after the test will be in the results folder labelled results.bin, at any optionally passed in directory. To convert to other formats, you can use the [vegeta cli](https://github.com/tsenart/vegeta) to generate reports.

#### Benchmark with profiling

Instead of running the load generator by itself, a profile.sh script has been set up that will start the load generator and profiling with pprof alog with tracking nats queue depth. Run with

```bash
cd load_generator
bash profile.sh 30 10 5 config.json <optional IP: default localhost>
```

specify IP only if hosting logarithm and running load generator on different machines.
The profile.sh script will write all results in the `benchmark_results` directory under a timestamped folder.
