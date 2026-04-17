# core-shared

Shared Go packages used across ECQ Core services (core-binary-analysis, core-orchestration, core-binary-remediation, core-binary-assessment, core-vulnerability, core-binary-scope, core-agent).

Module path: `github.com/khaicode-xyz/core-shared`

## Packages

| Package | Purpose |
|---|---|
| `apperror` | Typed errors with HTTP status + code |
| `logger` | slog JSON logger + context helpers |
| `response` | Unified JSON response writer (Success/Data/Error) |
| `middleware` | Chi middlewares — requestid, logging, recovery, timeout, cors |
| `validator` | go-playground/validator wrapper returning AppError |
| `camunda` | Zeebe client — NewClient + PublishMessage (+ Zeebe() for extensions) |
| `redis` | go-redis wrapper — Get/Set/Del/Ping/Close |
| `client` | FileClient — file-service presign (otelhttp instrumented) |
| `telemetry` | OpenTelemetry init for SignOz (OTLP/gRPC traces + metrics) |

## Usage

```go
import (
    "github.com/khaicode-xyz/core-shared/logger"
    "github.com/khaicode-xyz/core-shared/response"
    "github.com/khaicode-xyz/core-shared/telemetry"
)
```

In the service's `go.mod`:

```
require github.com/khaicode-xyz/core-shared v0.1.0
```

## Release process

Go modules are versioned by git tags. To cut a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Then bump `go.mod` in each service and run `go mod tidy`.

- **Breaking changes** → major bump (v1 → v2); also requires renaming the module path to `/v2` per Go's semantic import versioning.
- **New packages / additive changes** → minor bump.
- **Bug fixes only** → patch bump.

## Local development (before first publish)

While `core-shared` is not yet pushed, services use a `replace` directive pointing to the local path:

```
// in core-binary-analysis/go.mod
replace github.com/khaicode-xyz/core-shared => ../core-shared
```

After the first tag is pushed, remove the `replace` directive and run `go mod tidy` in each service.

## Private repo note

If the GitHub repo is private, consumers need:

```bash
export GOPRIVATE=github.com/khaicode/*
```

…and a GitHub token configured for `go get` (via `~/.netrc` or `git config`).
