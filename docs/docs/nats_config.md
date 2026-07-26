# Configuring NATS

NATS is configured via a `nats.conf` file that is passed into the `docker-compose.yml` when spinning up NATS

``` yaml
listen: 0.0.0.0:4222
http_port: 8222

# Client connection settings
max_connections: 100000
max_payload: 1MB #(1)!
max_pending: 256MB
write_deadline: 10s

#jetstream config
jetstream {
store_dir: /data/jetstream

max_memory_store: 4GB
max_file_store: 100GB

sync_interval: "2m" #(2)!
}

# Logging
debug: false
trace: false
logtime: true
```

1. Increase if incoming payloads are too large. Alternatively, look into [compressing payloads](otel.md#payload-compression) being sent to the ingester.
2. Sync interval for file storage. Lower values increase durability but reduce throughput.

Nats jetstream is used as a durable queue to buffer logs in transition to the database. Logs are persisted between container startup and shutdown. Logs that fail to be sent to the database after a fixed number of retries are sent to a [**dead letter queue**](nats.md#dead-letter-queue), where they have to be manually managed.
