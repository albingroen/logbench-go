# logbench-go

Go SDK for [Logbench](https://logbench.com) — structured logging with a visual UI.

## Installation

```bash
go get github.com/albingroen/logbench-go
```

## Usage

```go
package main

import (
    "errors"
    "github.com/albingroen/logbench-go"
)

func main() {
    logger := logbench.New("your-project-id")
    defer logger.Close()

    logger.Info("Server started", map[string]any{"port": 8080})
    logger.Warn("Slow query detected", 1200, "ms")

    err := errors.New("connection refused")
    logger.Err("Connection failed", err)
}
```

### Configuration

```go
logger := logbench.New("your-project-id", logbench.Options{
    URL:                  "http://localhost:1447", // default
    DisableCaptureSource: false,                   // default: captures file/line
    Cwd:                  "/path/to/project",      // for relative source paths
})
```

### Log with metadata

```go
logger.InfoWith(logbench.LogOptions{
    Bookmark:   true,
    Annotation: "deployment checkpoint",
}, "Deployment complete", version)
```

### Methods

| Method | Level |
|--------|-------|
| `Info(content ...any)` | INFO |
| `Warn(content ...any)` | WARNING |
| `Err(content ...any)` | ERROR |
| `InfoWith(opts, content ...any)` | INFO |
| `WarnWith(opts, content ...any)` | WARNING |
| `ErrWith(opts, content ...any)` | ERROR |

### Special type handling

Go-specific types are encoded with `@go/` prefixed type envelopes:

- `complex64/128` — `@go/complex64`, `@go/complex128`
- `error` — `@go/error` (with message and type)
- `func` — `@go/Function`
- `*big.Int` / `*big.Float` — `@go/big.Int`, `@go/big.Float`
- `chan` — `@go/chan`
- `NaN` / `±Inf` — `@go/NaN`, `@go/Infinity`
- `[]byte` — `@go/bytes` (base64-encoded)
- structs — `@go/Struct` (exported fields only)

Types implementing `json.Marshaler` (e.g. `time.Time`) use their standard encoding.

## Non-blocking

All HTTP requests are sent in background goroutines. Call `Close()` (or `defer logger.Close()`) to wait for in-flight logs before exiting.
