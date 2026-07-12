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

Brief explanation of what this project is about

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

This is a short tutorial on connecting a service to logarithm itself, and sending some sample logs.

**Prerequisites**: You must have logarithm running locally on your machine via docker. See [installation-guide](#installation-guide)

* dockerised http server maybe using python fastapi
* setup otlp into loggers and point directly to logarithm or to an otel collector
* send curl requests to force log events and view dashboard
