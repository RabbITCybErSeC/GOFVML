# Style, Interface, And Test Standards

## Naming Standard

GOFVML should optimize for forensic readability over terse idiom. This project will be read during incidents, reviews, and postmortems. Names should carry meaning.

Required:

- Use `context` for `context.Context` parameters, not `ctx`.
- Use `source` instead of `src`.
- Use `destination` or `output` instead of `dst`.
- Use `reader` and `writer` instead of `r` and `w` in non-trivial functions.
- Use `block` instead of `b`.
- Use `rangeValue`, `memoryRange`, or `mappingRange` instead of `r` when range semantics matter.
- Use `configuration` or `options` instead of `cfg`.
- Use `acquireError`, `readError`, or `operationError` in public examples instead of bare `err`.

Allowed:

- Very small loop indices like `index`.
- Bare `err` in tiny internal functions where the failure is immediately returned and no clarity is lost.
- Conventional receiver names only if still readable. Prefer `snapshotter` over `s`.

Examples:

```go
func Acquire(context context.Context, options Options) (Result, error)
```

Prefer:

```go
for _, memoryRange := range memoryRanges {
    block := Block{
        Offset: memoryRange.Start,
        Range:  memoryRange,
    }
    writeError := encoder.WriteBlock(context, block, source)
    if writeError != nil {
        return Result{}, writeError
    }
}
```

Avoid:

```go
func Acquire(ctx context.Context, cfg Options) (Result, error)
```

## Interface Naming

Interfaces should describe behavior:

- `MemorySource`
- `ImageEncoder`
- `ProcessReader`
- `MetadataWriter`
- `Uploader`
- `ProgressReporter`

Avoid vague interfaces:

- `Manager`
- `Handler`
- `Processor`
- `Helper`

Interfaces should be small. If an interface has more than three or four methods, split it or explain why it is cohesive.

## Package Boundary Tests

Each public package needs tests from the consumer's perspective:

- `pkg/gofvml/physical`: acquire from fake source.
- `pkg/gofvml/process`: dump from fake procfs or child process.
- `pkg/gofvml/image`: encode/decode known headers.
- `pkg/gofvml/convert`: convert generated images.
- `pkg/gofvml/upload`: upload to test server.

These are contract tests. They should fail only when a public promise changes.

## Internal Tests

Internal tests should cover edge cases:

- Range arithmetic.
- Overflow and underflow.
- Short reads.
- Permission errors.
- Missing files.
- Zero-length ranges.
- Empty images.
- Invalid headers.
- Malformed `/proc` files.
- Process exits during dump.

Use fixture files where format matters. Do not hide binary format behavior behind only high-level tests.

## Test Naming

Use behavior names:

```go
func TestAcquireUsesAutoSourceFallbackOrder(t *testing.T)
func TestLimeHeaderEncodesInclusiveEndAddress(t *testing.T)
func TestProcessDumpRecordsPartialReadWhenMappingVanishes(t *testing.T)
```

Avoid:

```go
func TestAcquire(t *testing.T)
func TestHeader(t *testing.T)
func TestError(t *testing.T)
```

## Test Doubles

Use explicit fake implementations:

```go
type FakeMemorySource struct {
    NameValue string
    Blocks    map[uint64][]byte
}
```

Do not overuse mocks. Most GOFVML behavior is deterministic and easier to test with fixtures or fake readers.

## Context Cancellation Tests

Every long operation must have a cancellation test:

- Physical acquisition stops between blocks.
- Process dump stops between mappings.
- Conversion stops between blocks.
- Upload stops during body read if possible.

The test should assert:

- The function returns a context cancellation error or wraps one.
- Partial result behavior is documented.
- Output file finalization behavior is deterministic.

## Volatility Compatibility Tests

Add a separate test group for external Volatility validation:

```text
internal/volatilitytest
```

These tests should be skipped unless Volatility is explicitly configured. They should never make normal `go test ./...` require Python or symbol files.

Environment variables:

- `VOL_PY`: path to `vol.py`.
- `VOL_SYMBOLS`: optional symbols path.
- `GOFVML_TEST_IMAGE`: optional existing image path.

## Documentation Tests

Public Go examples should compile. They must:

- Use descriptive variable names.
- Avoid `panic`.
- Avoid `os.Exit`.
- Show typed error inspection where useful.
- Show `context := context.Background()` instead of `ctx :=`.

## Linting Expectations

Recommended checks:

- `gofmt`
- `go vet`
- `go test ./...`
- `go test -race ./...` where feasible
- future custom lint or review checklist for forbidden short names in exported examples

Do not introduce a heavy lint stack before the package structure exists. Start with conventions, tests, and review checklists.

## Review Checklist

Before merging implementation:

- [ ] Public APIs use descriptive names.
- [ ] `context.Context` parameters are named `context`.
- [ ] CLI logic is thin and delegates to library APIs.
- [ ] Volatility-compatible LiME output has no GOFVML metadata embedded.
- [ ] Sidecar metadata is optional.
- [ ] Errors are typed enough for an IR framework to branch on.
- [ ] Cancellation behavior is tested.
- [ ] Public examples compile.
- [ ] Process dumps do not claim to be full Volatility images.

