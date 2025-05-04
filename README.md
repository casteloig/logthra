# LogTool

![Go Version](https://img.shields.io/badge/go-1.18+-blue)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Logthra** is a log ingestion prototype written in Go. Inspired by [Grafana Loki](https://grafana.com/oss/loki/), it exposes a `/api/push` endpoint and stores structured logs into a ClickHouse database using a gRPC-based ingester.

<div align="center">
  <img src="assets/images/logthra.png" width="260" alt="Logthra Logo" />
</div>


---

> ⚠️ This project is in early development.

---

## ✨ Features

- Log ingestion via HTTP `POST /api/push`
- Structured log format (streams, values)
- Multi-tenant support via `tenant-id` HTTP header
- gRPC communication between API and ingester
- ClickHouse integration with automatic DB/table creation
- WAL recovery logic (partially implemented)

---

## Example payload

```json
{
  "streams": [
    {
      "stream": {
        "level": "info",
        "app": "test-app"
      },
      "values": [
        ["2024-01-01", "starting up"],
        ["2024-01-01", "ready"]
      ]
    }
  ]
}
```

## How it works

`logthra` is designed as a lightweight log ingestion pipeline.

It consists of two main components:

- **API Layer (`api.go`)**: exposes an HTTP endpoint `/api/push` to receive logs. Logs are expected in a structured format similar to Grafana Loki's push API. The tenant is identified via the `tenant-id` HTTP header. Once received, the data is forwarded via gRPC to the ingester.

- **Ingester Layer (`ingester.go`)**: receives incoming log data through gRPC, processes and buffers it. Every 30 seconds, logs are inserted into a ClickHouse database. A basic WAL mechanism is in place to allow recovery of unflushed data on restart (though WAL writing is not yet complete).

Data is stored in a ClickHouse table with the following schema:

```sql
CREATE TABLE IF NOT EXISTS logs.log(
    tenant_id FixedString(10),
    label Array(String),
    timestamp Date,
    msg String
)
ENGINE = MergeTree()
ORDER BY timestamp
```

Tenants are isolated via their tenant_id, and each log entry includes its label set and timestamp.

The ingester creates the database and table automatically on startup, assuming ClickHouse is reachable at the provided address.

Currently, there is no authentication, query API, or advanced indexing.