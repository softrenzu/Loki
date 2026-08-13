# RooomLog — Search-First Log Engine

Version: `0.3.0`

RooomLog is a source-available log engine designed for fast full-text and high-cardinality structured search with a small operational footprint. It provides native ingestion, search, live tail, anomaly discovery, and a built-in browser UI while retaining selected Loki-compatible migration endpoints.

## Differentiators

- Full-text token indexing of message content
- Direct search across high-cardinality structured fields and trace/request IDs
- Built-in browser UI
- Native OTLP/HTTP JSON ingestion
- Loki-compatible JSON push endpoint
- A deliberate LogQL migration subset
- SSE live tail
- Deterministic anomaly-pattern discovery
- Multi-tenancy with `X-Scope-OrgID`
- Prometheus-compatible metrics
- Single Go binary with no external database requirement for local deployment

Loki is a compatibility and comparison reference, not the product identity. RooomLog is an independent early implementation and does not claim distributed-scale parity with Loki's mature object-storage, compaction, scheduling, and sharding architecture.

## Quick start

```bash
go run ./cmd/rooomlog
```

Open `http://localhost:3100`, or run:

```bash
docker compose up --build
```

## Ingestion

Native JSON:

```text
POST /api/v1/ingest
```

OTLP/HTTP JSON:

```text
POST /v1/logs
```

Loki migration endpoint:

```text
POST /loki/api/v1/push
```

## Search

```bash
curl 'http://localhost:3100/api/v1/search?q=req-78321&label.service_name=payments&limit=100' \
  -H 'X-Scope-OrgID: demo'
```

The compatibility query endpoint supports a documented subset such as `{app="api"} |= "timeout"` and rejects unsupported constructs rather than silently changing semantics.

## Roadmap

- Compressed immutable segment engine
- S3-compatible object-storage tier
- Distributed ingest/query mode
- Broader Loki protocol and LogQL compatibility
- Phrase/regex indexing and cost-based query planning
- Change-point detection and trace/log correlation
- OIDC, tenant quotas, audit logging, and field-level redaction
- Reproducible RooomLog/Loki performance and TCO benchmarks

## Licensing and enterprise support

Starting with version `0.3.0`, ROOOMTECH-authored code is offered under either the PolyForm Noncommercial License 1.0.0 for uses permitted by that license, or a separate paid ROOOMTECH Commercial Software License for business/commercial-purpose uses and other uses outside the PolyForm permission.

Commercial license agreements, maintenance, technical support, implementation, integration, upgrades, security support, SLA options, private builds, and custom development are available.

Contact: `support@rooomtech.com`

PolyForm Noncommercial License 1.0.0: https://polyformproject.org/licenses/noncommercial/1.0.0

Earlier releases retain their published license terms. Third-party software retains its own licenses. See `LICENSE`.
