# Apple Container Live Tests

This folder contains optional live validation harnesses for running GOFVML
inside Apple's native `container` runtime on macOS.

These tests are not part of `go test ./...`. They require Apple `container`,
Linux image access, and process tracing capability inside the container.

## Process Acquisition

Run the process acquisition live test:

```bash
testing/apple-container/process-live.sh
```

What it verifies:

- builds `dist/gofvml-linux-arm64` and `dist/gofvml-linux-amd64`
- starts an Ubuntu container with `SYS_PTRACE`
- launches a deterministic target process with a marker in memory
- runs non-strict `gofvml process` and expects success with warning output
- runs strict `gofvml process` and expects a read error
- confirms the marker is recoverable from the generated process artifact

Useful overrides:

```bash
ARCH=amd64 testing/apple-container/process-live.sh
IMAGE=docker.io/library/ubuntu:24.04 testing/apple-container/process-live.sh
HOST_OUTDIR=/tmp/gofvml-live testing/apple-container/process-live.sh
```

Artifacts and captured stdout/stderr default to:

```text
validation-output/apple-container-process/
```

That directory is ignored by git.
