# OpenTelemetry Integration

This page covers how to send logs from your service to the ingester using
OpenTelemetry. Two paths are documented:

- **Quickstart By Language**: export logs directly to the ingester. No extra
  infrastructure to run. Good default for a single service.
- **Using the OpenTelemetry Collector**: front the ingester with a collector.
  Reach for this once you're running multiple services, want retry/buffering
  during ingester downtime, or need compression the language SDKs don't
  support natively (e.g. zstd).

!!! Info
    Replace `localhost:PORT` in every example below with your actual ingester address.
    The given addresses are based on provided defaults.

---

## Quickstart By Language

=== "Python"

    Example in `FastAPI`. See [Using a different framework](#python-framework) below for Flask/Django.

    **Install:**

    ```bash
    pip install fastapi uvicorn opentelemetry-api opentelemetry-sdk \
    opentelemetry-exporter-otlp-proto-http opentelemetry-instrumentation-fastapi
    ```

    **`main.py`:**

    ```python
    import logging
    from fastapi import FastAPI
    from opentelemetry import trace
    from opentelemetry._logs import set_logger_provider
    from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
    from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
    from opentelemetry.exporter.otlp.proto.http import Compression
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
    exporter = OTLPLogExporter(endpoint="http://localhost:8090/v1/logs", compression=Compression.Gzip)
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

    **Run:**

    ```bash
    uvicorn main:app --reload
    ```

    **Using a different framework (Python)**
    {: #python-framework }

    Everything above stays the same — only the instrumentor call changes:

    | Framework | Instrumentor call |
    |---|---|
    | FastAPI | `FastAPIInstrumentor.instrument_app(app)` |
    | Flask | `FlaskInstrumentor().instrument_app(app)` |
    | Django | `DjangoInstrumentor().instrument()` (call in settings.py, no `app` object needed) |

=== "JavaScript / Node.js"

    Example in `Express`. See [Using a different framework](#javascript-framework) below for Fastify/Next.js.

    **Install:**

    ```bash
    npm install express @opentelemetry/sdk-node @opentelemetry/sdk-logs \
    @opentelemetry/api-logs @opentelemetry/exporter-logs-otlp-http \
    @opentelemetry/instrumentation-http @opentelemetry/instrumentation-express \
    @opentelemetry/resources
    ```

    **`tracing.js`** (must be required before your app code):

    ```javascript
    const { NodeSDK } = require('@opentelemetry/sdk-node');
    const { BatchLogRecordProcessor } = require('@opentelemetry/sdk-logs');
    const { OTLPLogExporter } = require('@opentelemetry/exporter-logs-otlp-http');
    const { HttpInstrumentation } = require('@opentelemetry/instrumentation-http');
    const { ExpressInstrumentation } = require('@opentelemetry/instrumentation-express');
    const { resourceFromAttributes } = require('@opentelemetry/resources');

    const sdk = new NodeSDK({
    resource: resourceFromAttributes({ 'service.name': 'my-express-service' }),
    logRecordProcessor: new BatchLogRecordProcessor(
        new OTLPLogExporter({
            url: 'http://localhost:8090/v1/logs',
        })
    ),
    // HttpInstrumentation creates the per-request span; ExpressInstrumentation
    // adds route-level detail on top of it. Together they're what let
    // logger.emit() below carry a trace_id/span_id automatically.
    instrumentations: [new HttpInstrumentation(), new ExpressInstrumentation()],
    });

    sdk.start();
    ```

    **`server.js`:**

    ```javascript
    require('./tracing');
    const express = require('express');
    const { logs } = require('@opentelemetry/api-logs');

    const logger = logs.getLogger('my-express-service');
    const app = express();

    app.get('/', (req, res) => {
    logger.emit({ severityText: 'INFO', body: 'from root!' });
    res.json({ message: 'Hello, Express with OpenTelemetry' });
    });

    app.get('/warn', (req, res) => {
    logger.emit({ severityText: 'WARN', body: 'fake warning!' });
    res.json({ message: 'Hello, Express with OpenTelemetry!' });
    });

    app.get('/error', (req, res) => {
    logger.emit({ severityText: 'ERROR', body: 'fake error!' });
    res.json({ message: 'Hello, Express with OpenTelemetry!' });
    });

    app.listen(3000, () => console.log('listening on :3000'));
    ```

    **Run:**

    ```bash
    node server.js
    ```

    **Using a different framework (JavaScript)**
    {: #javascript-framework }

    Swap the instrumentation package (and drop `ExpressInstrumentation` from the `instrumentations` array in favor of the matching one):

    | Framework | Instrumentation package |
    |---|---|
    | Express | `@opentelemetry/instrumentation-express` |
    | Fastify | `@opentelemetry/instrumentation-fastify` |
    | Next.js | `@opentelemetry/instrumentation-http` is usually sufficient — Next.js has [built-in OTel support](https://nextjs.org/docs/app/building-your-application/optimizing/open-telemetry) via `instrumentation.ts` |

    Or skip picking individual packages entirely and use `@opentelemetry/auto-instrumentations-node`, which detects installed libraries (Express, Fastify, pg, redis, etc.) and instruments them automatically.

=== "Go"

    Example in `net/http`. See [Using a different framework](#golang-framework) below for Gin/Echo.

    **`main.go`:**

    ```go
    package main

    import (
            "context"
            "fmt"
            "net/http"

            "go.opentelemetry.io/contrib/bridges/otelslog"
            "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
            "go.opentelemetry.io/otel"
            "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
            "go.opentelemetry.io/otel/log/global"
            sdklog "go.opentelemetry.io/otel/sdk/log"
            "go.opentelemetry.io/otel/sdk/resource"
            sdktrace "go.opentelemetry.io/otel/sdk/trace"
            semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
    )

    func main() {                                                                                         
            ctx := context.Background()
            // 1. Define your service name 
            res, err := resource.Merge(
                resource.Default(),
                resource.NewWithAttributes(
                    resource.Default().SchemaURL(), // Dynamically matches your SDK's schema version (e.g., 1.41.0 or 1.44.0)
                    semconv.ServiceName("my-go-service"),
                ),
            )

            if err != nil {
                    panic(err)
            }

            // 2. Initialize a local TracerProvider to generate IDs
            tracerProvider := sdktrace.NewTracerProvider(
                    sdktrace.WithResource(res),
                    // Note: We are not attaching a trace exporter here,
                    // so traces are generated for context but not sent over the network.
            )
            defer tracerProvider.Shutdown(ctx)
            // Register it globally so otelhttp can find it
            otel.SetTracerProvider(tracerProvider)

            // 3. Initialize your Log Exporter
            exporter, err := otlploghttp.New(ctx,
                    otlploghttp.WithEndpoint("localhost:8090"),
                    otlploghttp.WithInsecure(),
            )
            if err != nil {
                    panic(err)
            }

            // 4. Initialize LoggerProvider with the Resource
            provider := sdklog.NewLoggerProvider(
                    sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
                    sdklog.WithResource(res), // Attach the resource to logs
            )
            defer provider.Shutdown(ctx)
            global.SetLoggerProvider(provider)

            // otelslog bridges Go's stdlib slog to the OTel logs SDK
            logger := otelslog.NewLogger("my-go-service")

            mux := http.NewServeMux()
            mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                    logger.InfoContext(r.Context(), "from root!")
                    w.Write([]byte(`{"message": "Hello, Go with OpenTelemetry"}`))
            })
            mux.HandleFunc("/warn", func(w http.ResponseWriter, r *http.Request) {
                    logger.WarnContext(r.Context(), "fake warning!")
                    w.Write([]byte(`{"message": "Hello, Go with OpenTelemetry!"}`))
            })
            mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
                    logger.ErrorContext(r.Context(), "fake error!")
                    w.Write([]byte(`{"message": "Hello, Go with OpenTelemetry!"}`))
            })

            // otelhttp will now use the global TracerProvider to create real spans
            handler := otelhttp.NewHandler(mux, "http-server")
            fmt.Println("listening on http://localhost:8080")
            http.ListenAndServe(":8080", handler)
    }
    ```

    **Install:**

    ```bash
    go mod init example.com/m

    go mod tidy
    ```

    **Run:**

    ```bash
    go run main.go
    ```

    **Using a different framework (Go)**
    {: #golang-framework }

    Swap the handler-wrapping middleware; the logging/exporter setup above stays identical:

    | Framework | Middleware |
    |---|---|
    | `net/http` | `otelhttp.NewHandler(mux, "http-server")` |
    | `Gin` | [`otelgin.Middleware("service-name")`](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gin-gonic/gin/otelgin) |
    | `Echo` | [`otelecho.Middleware("service-name")`](https://github.com/labstack/echo-contrib/tree/master/otelecho) |

---

## Using the OpenTelemetry Collector

Reach for this once any of the following apply:

- You're running more than one service and don't want to configure/rotate
  the ingester address and credentials in every one of them individually.
- You want a retry/buffering layer so logs survive a brief ingester outage.
- You need compression the language SDKs don't support directly. For example, **zstd**, which the collector supports but the Python/JS/Go OTLP exporters currently do not (they support gzip).

??? "docker-compose.yml"

    ```yaml
    services:
    otel-collector:
        image: otel/opentelemetry-collector
        volumes:
        - ./otel-collector-config.yaml:/etc/otelcol/config.yaml
        ports:
        - 4317:4317 # OTLP gRPC receiver
        - 4318:4318 # OTLP HTTP receiver
    ```

??? "otel-collector-config.yaml"

    ```yaml
    receivers:
    otlp:
        protocols:
        grpc:
            endpoint: 0.0.0.0:4317
        http:
            endpoint: 0.0.0.0:4318

    exporters:
    otlphttp/logarithm_http:
        endpoint: "http://localhost:8090" #(1)!
        encoding: proto #(2)!
        compression: zstd
    otlp/logarithm_grpc:
        endpoint: "http://localhost:8089" #(3)!
        compression: zstd
        tls:
        insecure: true

    service:
    pipelines:
        logs:
        receivers: [otlp]
        exporters: [otlp/logarithm_grpc] #(4)!
    ```

    1.  must match port if reconfigured from defaults.
    2.  swap to `json` to send json encoded payloads instead.
    3.  must match port if reconfigured from defaults.
    4.  swap to `otlp/logarithm_http` to use http transport.

After copying above files to the same directory run:

```bash
docker compose up -d
```

### Pointing your app at the collector instead

In every quickstart example above, change the exporter endpoint from the
ingester address to the collector's, running locally:

| Language | Change |
| --- | --- |
| Python | `endpoint="http://localhost:4318/v1/logs"` |
| JavaScript | `url: 'http://localhost:4318/v1/logs'` |
| Go | `otlploghttp.WithEndpoint("localhost:4318")` |

Everything else in each example - resource, batch processor, framework
instrumentation, stays the same.

## Payload Compression

Payload compression is useful in reducing network traffic especially when log traffic is high. Logarithm supports `no compression`, `gzip` or `zstd`.

Most SDKs only support gzip compression, to utilise zstd, it is recommended to use the OpenTelemetryCollector. See [OpenTelemetry Integration](#opentelemetry-integration)
