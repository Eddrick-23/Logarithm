# Configuring ClickHouse

ClickHouse is configured via the `docker-compose.yml` file.

```yaml
clickhouse:
    image: clickhouse/clickhouse-server:26.3
    ports:
        - "8123:8123"  #(1)!
        - "9000:9000"  #(2)!
    volumes:
        - ./deploy/clickhouse/init.sql:/docker-entrypoint-initdb.d/init.sql
        - clickhouse_data:/var/lib/clickhouse
        - clickhouse_logs:/var/log/clickhouse-server
    environment: #(3)!
        CLICKHOUSE_DB: default
        CLICKHOUSE_USER: admin
        CLICKHOUSE_PASSWORD: strongpassword
        CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: 1
    ulimits: #(4)!  
        nofile:
        soft: 262144
        hard: 262144
    healthcheck:
        test: ["CMD", "clickhouse-client", "--query", "SELECT 1"]
        interval: 10s
        timeout: 5s
        retries: 10
        start_period: 60s # increased to allow more time for init.sql migrations
```

1. Web UI: Used to access the built-in HTTP visual interface via your browser.
2. Native Port: Used by ClickHouse clients and external applications to send and retrieve raw data over TCP.
3. Credentials: Sets the database name, username and password. This must match the corresponding environment variables injected to the backend services. See [Credentials](#credentials)
4. Open File Limits: ClickHouse's columnar storage model requires opening thousands of files simultaneously. Increasing this limit prevents fatal "too many open files" errors.

## Ports

ClickHouse exposes two ports:

- `8123` to access the web UI
- `9000` which is for clients to send and retrieve data.
If the second port `9000` is modified, the env variables passed in must match this change

``` text title=".env"
DB_ADDRESS=clickhouse:<modified-port>
```

## Credentials

By default, the `docker-compose.yml` file provisions the initial admin account using environment variables:

- **Username:** `admin` (configured via `CLICKHOUSE_USER`)
- **Password:** `strongpassword` (configured via `CLICKHOUSE_PASSWORD`)

> Detailed guide for environment variables is configured [here](environment.md).
