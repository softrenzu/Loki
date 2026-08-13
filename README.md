# RooomLog

RooomLog is an Apache-2.0 log engine designed as a simpler, search-first alternative to Grafana Loki for teams that need fast full-text and high-cardinality structured search without operating Grafana plus a multi-component log stack.

> Repository name remains `Loki` for compatibility with this project location; the software itself is called **RooomLog**.

## Why RooomLog

Loki is excellent at low-cost object-storage-centric logging, but its core model indexes labels rather than the full log body, its Bloom query acceleration is still experimental in current Loki 3.7 documentation, and Loki itself has no UI. RooomLog deliberately takes a different trade-off: it maintains an adaptive in-process inverted index across message text, labels, structured fields, trace IDs and span IDs, while persisting an append-only JSONL WAL.

### Implemented differentiators

| Capability | RooomLog | Loki-oriented workflow |
|---|---|---|
| Full-text token index | Yes, built in | Primarily label/metadata index + chunk scan |
| High-cardinality field search | Direct search across fields/trace/request IDs | Usually structured metadata / careful label design |
| Native browser UI | Built in at `/` | Typically Grafana required |
| Native OTLP/HTTP JSON ingest | Yes, `/v1/logs` | Yes |
| Loki push compatibility | Yes, `/loki/api/v1/push` JSON | Native |
| LogQL migration path | `query_range` subset (`{labels} |= "text"`) | Full LogQL |
| Live tail | SSE `/api/v1/tail` | Loki tail APIs/tools |
| Anomaly discovery | Built-in deterministic pattern rarity endpoint | Usually external tooling/rules |
| Multi-tenancy | `X-Scope-OrgID` | `X-Scope-OrgID` |
| Prometheus metrics | `/metrics` | Yes |
| Single binary | Yes | Yes in single-binary mode |
| External database required | No | No for local/single-store patterns |
| License | Apache-2.0 | AGPL-3.0-only with stated exceptions |

RooomLog is an early working implementation, not a claim that it already beats Loki at every production scale. Loki has years of distributed-systems engineering, object-store durability, compaction, query scheduling, sharding, and ecosystem maturity. The goal is to outperform it on **simplicity, direct search ergonomics, high-cardinality search, and built-in analysis**, then scale the storage engine iteratively.

## Quick start

```bash
go run ./cmd/rooomlog
```

Open `http://localhost:3100`.

Or:

```bash
docker compose up --build
```

Default data directory is `./data`. With Docker it is `/data`.

## Ingest logs

### RooomLog JSON API

```bash
curl -X POST http://localhost:3100/api/v1/ingest \
  -H 'Content-Type: application/json' \
  -H 'X-Scope-OrgID: demo' \
  -d '{
    "message":"database timeout for request 78321",
    "labels":{"service_name":"payments","level":"error"},
    "fields":{"request_id":"req-78321","customer_id":"cust-99401"},
    "trace_id":"9f7c2e11b88d4f16"
  }'
```

A JSON array is accepted for batch ingest.

### Loki-compatible JSON push

```bash
curl -X POST http://localhost:3100/loki/api/v1/push \
  -H 'Content-Type: application/json' \
  -d '{"streams":[{"stream":{"app":"api","level":"error"},"values":[["1710000000000000000","timeout talking to database",{"request_id":"r-1"}]]}]}'
```

This MVP accepts Loki's JSON push representation. Snappy-compressed protobuf ingest is on the roadmap.

### OTLP/HTTP JSON

Point an OpenTelemetry Collector HTTP exporter at:

```text
http://rooomlog:3100/v1/logs
```

RooomLog maps `service.name` to `service_name`, severity to `level`, keeps other resource/log attributes as structured fields, and preserves trace/span IDs.

## Search

```bash
curl 'http://localhost:3100/api/v1/search?q=req-78321&label.service_name=payments&limit=100' \
  -H 'X-Scope-OrgID: demo'
```

Parameters:

- `q`: full-text / structured-field substring query
- `label.<name>`: exact indexed label filter
- `from`, `to`: Unix seconds/ms/ns or RFC3339 timestamps
- `limit`: max 10,000, default 200

Search candidates are selected using the smallest available posting list from exact labels or query tokens, then verified against the complete log record. This avoids scanning every record for common targeted searches while preserving simple semantics.

## Loki query_range compatibility

Supported migration subset:

```text
{app="api",level="error"} |= "timeout"
```

Example:

```bash
curl -G http://localhost:3100/loki/api/v1/query_range \
  --data-urlencode 'query={app="api"} |= "timeout"'
```

The compatibility endpoint intentionally rejects unsupported LogQL constructs instead of silently returning incorrect results.

## Live tail

```bash
curl -N http://localhost:3100/api/v1/tail \
  -H 'X-Scope-OrgID: demo'
```

Records are emitted as Server-Sent Events.

## Anomaly patterns

```bash
curl 'http://localhost:3100/api/v1/anomalies?hours=6' \
  -H 'X-Scope-OrgID: demo'
```

The current engine normalizes highly dynamic tokens such as long numeric IDs and hex-like IDs, groups messages into patterns, and ranks rare patterns. It is deliberately deterministic and runs without an external LLM. A future optional LLM layer can explain clusters without making ingestion dependent on an AI service.

## Configuration

| Environment variable | Default | Meaning |
|---|---:|---|
| `ROOOMLOG_ADDR` | `:3100` | HTTP listen address |
| `ROOOMLOG_DATA` | `./data` | WAL/data directory |
| `ROOOMLOG_RETENTION_HOURS` | `168` | Local retention; `0` disables |

## Architecture

```text
Loki clients -----------\
OTel Collector ----------> HTTP ingest ----> canonical LogEntry
Native JSON ------------/                       |
                                                v
                               +----------------+----------------+
                               | append-only durable JSONL WAL   |
                               +----------------+----------------+
                                                |
                      +-------------------------+-------------------------+
                      |                         |                         |
                token postings            label postings             live tail
          msg + fields + trace IDs       exact label pairs              SSE
                      |                         |
                      +------------+------------+
                                   v
                              query verifier
                                   |
                    +--------------+---------------+
                    |                              |
              native search API             Loki query_range subset
                    |
          built-in UI / anomaly patterns
```

## Durability and operational behavior

Each accepted record is appended to the WAL and `fsync`ed before the request is acknowledged. On startup the indexes are rebuilt from the WAL. Retention rewrites the WAL atomically using a temporary file and rename. This favors correctness and simple recovery over peak ingest throughput in the first release.

For high ingest volume, the next storage generation should introduce batched WAL fsync, immutable compressed segments, memory-mapped posting lists, background segment merge, and optional S3-compatible cold tiering.

## Roadmap to production-scale Loki competition

1. **Segment engine**: compressed immutable blocks, per-segment FST/posting indexes, background compaction.
2. **Object storage tier**: S3-compatible hot/warm/cold lifecycle with local cache.
3. **Distributed mode**: stateless ingest/query nodes, consistent hashing, replicated WAL, query fan-out.
4. **Compatibility**: Loki Snappy protobuf push, broader LogQL, Grafana datasource conformance tests.
5. **Search**: phrase/regex indexes, field statistics, query planner, approximate cardinality.
6. **Observability intelligence**: online change-point detection, pattern baselines, trace-log correlation and optional local/remote LLM explanations.
7. **Security**: API keys/OIDC, tenant quotas, audit log and field-level redaction policies.
8. **Benchmarks**: reproducible ingest/search/TCO suite against Loki 3.7.x on identical hardware and datasets.

## Development

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/rooomlog
```

The core currently uses only the Go standard library, keeping the binary and dependency surface small.

## API status

This is `v0.1.0`-quality software. Native APIs may evolve before `v1.0`. Loki-compatible endpoints will be kept backward compatible wherever practical.

## License

Apache License 2.0. See `LICENSE`.
