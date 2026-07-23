# Usage of NATS

Nats is used as a durable message queue. </br>

By default, a simple web UI is accessible at [localhost:8222](http://localhost:8123).
Alternatively, it is recommended to use the [NATS CLI](https://github.com/nats-io/natscli#readme).

## Dead Letter Queue

Two streams are used to manage the log pipeline. `LOGS` for the main log stream of incoming logs and `LOGS_DLQ` to store logs that failed to be sent to the database after a fixed number of retries. To manage and view the DLQ stream. You can use the nats cli directly. E.g.

```bash
nats stream info LOGS_DLQ
```

Use the cli help to see check supported operations.

```bash
nats --help
```
