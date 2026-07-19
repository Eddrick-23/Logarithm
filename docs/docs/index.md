# Welcome to MkDocs

For full documentation visit [mkdocs.org](https://www.mkdocs.org).

## Commands

* `mkdocs new [dir-name]` - Create a new project.
* `mkdocs serve` - Start the live-reloading docs server.
* `mkdocs build` - Build the documentation site.
* `mkdocs -h` - Print help message and exit.

## Project layout

```text
mkdocs.yml    # The configuration file.
docs/
    index.md  # The documentation homepage.
    ...       # Other markdown pages, images and other files.
```

**leave everything above as reference for now**

## Project Overview

Logarithm is a high-throughput, OpenTelemetry compliant, self-hosted observability pipeline capable of ingesting thousands of logs per second. The system will decouple data ingestion, transport and permanent storage using a Golang based engine, NATS Jetstream as a durable queue and ClickHouse for persistent storage, along with a React based dashboard for real-time live tailing and analytics.

## Installation guide

**Prerequisites**: You must have [Docker](https://www.docker.com/get-started/) installed.

* Copy the 3 files below into the same directory

??? "docker-compose.yml"

    ```yaml title="docker-compose.yml"
    --8<-- "deploy/docker-compose.yml"
    ```

??? "nats.conf"

    ```ini title="nats.conf"
    --8<-- "deploy/nats/nats.conf"
    ```

??? "init.sql"

    ```sql title="init.sql"
    --8<-- "deploy/clickhouse/init.sql"
    ```

* Build and run using

```bash
docker compose up -d
```

* Access the dashboard at [http://localhost:5173](http://localhost:5173).

## QuickStart

This is a short tutorial on connecting a simple FastAPI service, instrumented with OTLP sdk, to logarithm.

### Prerequisites

* You must have Logarithm running locally on your machine via Docker. See [installation-guide](#installation-guide).
* Python3 installed on your machine

### Guide

??? "main.py"

    ```python
    import time
    import logging
    from fastapi import FastAPI
    from opentelemetry import trace
    from opentelemetry._logs import set_logger_provider
    from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
    from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
    from opentelemetry.exporter.otlp.proto.http.import Compression
    from opentelemetry.sdk.resources import Resource, SERVICE_NAME
    from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

    # 1) Set up OTel logging. Sets service name for logs coming from this api.
    resource = Resource.create({SERVICE_NAME: "my-fastapi-service"})

    # 2) TracerProvider without exporter. This allows FastAPIInstrumentor to
    # stamp trace_id/span_id onto each log.
    tracer_provider = TracerProvider(resource=resource)
    trace.set_tracer_provider(tracer_provider)

    logger_provider = LoggerProvider(resource=resource)
    set_logger_provider(logger_provider)

    # 3) point exported logs to logarithm's ingester
    exporter = OTLPLogExporter(endpoint="http://localhost:8089/v1/logs", compression=Compression.Gzip)
    logger_provider.add_log_record_processor(BatchLogRecordProcessor(exporter))

    # 4) Attatch OpenTelemetry Handler to Python's Root Logger
    logging.getLogger().addHandler(LoggingHandler(logger_provider=logger_provider))
    logging.getLogger().setLevel(logging.INFO)

    logger = logging.getLogger(__name__)

    app = FastAPI()
    FastAPIInstrumentor.instrument_app(app)

    @app.get("/")
    async def read_root():
        logger.info("from root!")
        return {"message": "Hello, FastAPI with OpenTelemetry"}

    @app.get("/warn")
    async def read_warning():
        logger.warning("fake warning!")
        return {"message": "Hello, FastAPI with OpenTelemetry!"}

    @app.get("/info")
    async def read_info():
        logger.info("fake info!")
        return {"message": "Hello, FastAPI with OpenTelemetry!"}

    @app.get("/error")
    async def read_error():
        logger.error("fake error!")
        return {"message": "Hello, FastAPI with OpenTelemetry!"}
    ```

* Copy the above python file into a directory.
* Create a venv using `python3 -m venv venv`
* Activate the venv

    === "macOS/Linux"
        ```bash
        python3 -m venv venv
        source venv/bin/activate
        ```
    === "Windows (PowerShell)"
        ```powershell
        python -m venv venv
        venv\Scripts\Activate.ps1
        ```
    === "Windows (cmd)"
        ```bat
        python -m venv venv
        venv\Scripts\activate.bat
        ```

* Install libraries using

    ```bash
    pip install fastapi uvicorn opentelemetry-api opentelemetry-sdk \
        opentelemetry-exporter-otlp-proto-http opentelemetry-instrumentation-fastapi 
    ```

* Start the service with

    ```bash
    uvicorn main:app --port 8080
    ```

* send curl requests to trigger log events and watch the dashboard at [localhost:5173](http://localhost:5173)

    ```bash
    curl http://localhost:8080/

    curl http://localhost:8080/warn

    curl http://localhost:8080/error
    ```

!!! info
    For advanced configuration using an Otel Collector or to set up with different languages, see [Integration With OpenTelemetry](otel.md)
