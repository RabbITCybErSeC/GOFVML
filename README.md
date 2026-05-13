# GOFVML

Go for Volatile Memory Linux (GOFVML) is a Go-based Linux physical memory acquisition tool.

## Privilege Warning

Physical memory acquisition requires root privileges. Running without sufficient privileges will fail with clear diagnostics.

## Overview

GOFVML is a Go rewrite of [AVML](https://github.com/microsoft/avml) that is being built to preserve AVML's core operational strengths while adding:

- **PID-scoped process memory dumping** - capture virtual memory ranges for specific processes
- **Importable Go library** - use GOFVML as a library in incident-response frameworks
- **First-class Volatility 3 compatibility** - LiME output works directly with Volatility 3 Linux plugins

## Installation

```bash
go install github.com/RabbITCybErSeC/gofvml/cmd/gofvml@latest
```

Compatibility helper binaries remain available:

```bash
go install github.com/RabbITCybErSeC/gofvml/cmd/gofvml-convert@latest
go install github.com/RabbITCybErSeC/gofvml/cmd/gofvml-upload@latest
```

## Build From Source

```bash
make build              # bin/gofvml
make build-tools        # bin/gofvml plus compatibility helper binaries
make build-linux-amd64  # dist/gofvml-linux-amd64
make build-linux-arm64  # dist/gofvml-linux-arm64
make build-linux-armv7  # dist/gofvml-linux-arm-v7
make build-linux-386    # dist/gofvml-linux-386
make build-all          # all configured Linux targets for gofvml
```

## Commands

- `gofvml physical` - acquire physical memory to a LiME or AVML-compressed image
- `gofvml process --pid <pid>` - dump process virtual memory ranges
- `gofvml convert` - convert between raw, LiME, and AVML-compressed formats
- `gofvml compress` - compress a memory image to AVML-compatible format
- `gofvml upload` - upload memory images via HTTP PUT

The `gofvml-convert` and `gofvml-upload` compatibility binaries remain available for existing automation.

## Module Path

```
github.com/RabbITCybErSeC/gofvml
```

## Library Usage

```go
import "github.com/RabbITCybErSeC/gofvml/pkg/gofvml"
```

## Development Status

This project is under active development. See `openspec/changes/define-dedicated-software-features/` for current implementation tasks and `docs/avml-rewrite-plan/` for architecture background.

## License

MIT. See [LICENSE](LICENSE).
