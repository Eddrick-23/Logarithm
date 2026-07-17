# `frontend/.env.example`

Reference template for all frontend environment variables. See the [Environment Variables](../environment.md) page for descriptions and required status of each variable.

!!! tip "Usage"

    ```bash
    cp frontend/.env.example frontend/.env
    ```
    **Never commit `.env` itself**, only `.env.example`.

!!! warning "Two rules that matter here"
    - Every variable must be prefixed with `VITE_` to be exposed to client-side code. Anything without the prefix is silently dropped at build time, not injected.
    - **Never put secrets in this file.** Frontend env vars are bundled into public JavaScript and are visible to anyone who opens dev tools.

!!! info "Derived from backend config"
    `VITE_API_URL` and `VITE_WEBSOCKET_URL` should match the Dashboard API's `APP_HOST`/`APP_PORT` — see [`backend/.env.example`](backend-env-example.md).

```env title="frontend/.env.example"
--8<-- "frontend/.env.example"
```
