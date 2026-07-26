# `backend/.env.example`

Reference template for all backend environment variables (Ingester, Dashboard API, ClickHouse, Worker, NATS). See the [Environment Variables](../environment.md) page for descriptions, defaults, and required/optional status of each variable.

!!! tip "Usage"

    ```bash
    cp backend/.env.example backend/.env
    ```

    Then fill in real values for `DB_PASSWORD` and any other secrets. **Never commit `.env` itself**, only `.env.example`.

!!! warning "Constraints to double-check"
    - `DB_BATCH_POOL_MAX_ROWS` must be **>=** `WORKER_ROWS_PER_BATCH`
    - `NATS_CONSUMER_MAX_ACK_PENDING` must be **>=** `WORKER_MAX_BATCH`
    - `WORKER_LIVE_TAIL_PRESENCE_TIMEOUT` should be **>=** `LIVE_TAIL_PRESENCE_INTERVAL`

```env title="backend/.env.example"
--8<-- "backend/.env.example"
```
