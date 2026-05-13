# Image Formats And Conversion

## Format Families

AVML works with three conceptual formats:

- Raw physical image: byte offset equals physical address, with zero-filled gaps.
- LiME image: sequence of headers plus raw payload blocks.
- AVML-compressed image: LiME-like block sequence with AVML magic, Snappy-framed payloads, and compressed-length trailers.

GOFVML needs exact compatibility for LiME and AVML-compressed output.

## LiME Header

Header length is 32 bytes.

Little-endian layout:

| Offset | Size | Field |
| --- | ---: | --- |
| 0 | 4 | magic |
| 4 | 4 | version |
| 8 | 8 | range start |
| 16 | 8 | range end inclusive |
| 24 | 8 | padding, must be zero |

For LiME:

- Magic numeric value: `0x4c69_4d45`.
- Bytes on disk: `45 4d 69 4c`.
- Version: `1`.
- Payload: uncompressed bytes for the represented physical range.

AVML's in-memory range uses an exclusive end, but encodes `range.end - 1` into the header. When reading, it adds one to the encoded end.

GOFVML header type:

```go
type Header struct {
    Magic   uint32
    Version uint32
    Start   uint64
    End     uint64 // exclusive in memory
}
```

Encoding rule:

```text
encoded_end = End - 1
```

Decoding rule:

```text
End = encoded_end + 1
```

Reject:

- Unknown magic/version pair.
- Non-zero padding.
- End overflow when adding one.
- End before start.

## AVML-Compressed Header

The compressed format uses the same 32-byte header shape.

For AVML compressed:

- Magic numeric value: `0x4c4d_5641`.
- Bytes on disk: `41 56 4d 4c`.
- Version: `2`.
- Payload: Snappy frame stream.
- Trailer: 8-byte little-endian compressed payload length.

Block layout:

```text
[32-byte AVML header][snappy frame bytes][8-byte compressed length]
```

The trailer length counts only Snappy frame bytes, not the header and not the trailer.

Why the trailer exists:

- Snappy frame streams do not expose their compressed length from the uncompressed range size alone.
- Conversion needs to skip exactly past the compressed block before reading the next header.

## Block Size Rules

AVML constant:

```text
MAX_BLOCK_SIZE = 0x1000 * 0x1000 = 16 MiB
```

Rules:

- Version 2 copy splits ranges larger than 16 MiB into multiple blocks.
- Version 1 large blocks can be streamed as one block.
- Blocks up to 16 MiB are read fully into memory to detect all-zero blocks.
- All-zero blocks are omitted from both LiME and AVML-compressed output.
- Large version 1 blocks are not all-zero checked because they use the large streaming path.

GOFVML recommendation:

- Preserve this behavior for compatibility.
- Consider a later optimization that zero-checks large blocks in streaming windows, but treat that as a format-affecting change because output block presence may differ.

## Copy Semantics

AVML copy behavior:

- If the source requires alignment, read in 4 KiB pages using `read_exact`.
- If not, copy exactly `size` bytes.
- Any read or write failure aborts the snapshot.

GOFVML copy behavior should:

- Use `io.ReaderAt` or `ReadAt` loops for deterministic offset reads.
- For device-like sources, use `Seek` plus `ReadFull` if `ReadAt` is unsupported or unreliable.
- Return short reads with source, offset, and range context.

## Raw Conversion

### Encoded Image To Raw

AVML raw output expands gaps:

1. Read next LiME/AVML header.
2. Compare destination position with header start.
3. Write zero padding for unmapped gap.
4. Copy/decompress block payload to destination.
5. Repeat until source position reaches source file length.

This creates a raw address-space image from sparse blocks.

Important behavior:

- If trailing physical memory is absent because it was all zeros and skipped, raw output ends before the skipped tail.
- Tests explicitly verify trailing zero block is dropped from raw round trip.

### Raw To Encoded Image

AVML raw-to-image:

1. Get raw source length.
2. Loop from offset 0 to length in `MAX_BLOCK_SIZE` chunks.
3. `copy_block(start..end)` for each chunk.
4. `copy_block` skips zero chunks.

GOFVML should preserve this because it gives deterministic conversion and avoids emitting empty regions.

## LiME To AVML-Compressed

AVML conversion:

1. Read source header.
2. Set output version to 2.
3. Copy the block represented by the header through `copy_block`.
4. If source was LiME, payload is read raw.
5. If output is version 2, payload is compressed and length trailer is appended.

## AVML-Compressed To LiME

AVML conversion:

1. Read AVML header.
2. Write LiME header for the same range.
3. Create Snappy frame decoder over source.
4. Decode up to uncompressed range size into destination.
5. Seek forward 8 bytes to skip compressed length trailer.

Potential issue:

- AVML does not appear to verify the trailer value against actual consumed compressed length during decode. GOFVML should decide whether strict mode verifies it. For compatibility, default mode can match AVML; strict mode can validate.

## Header Test Vectors

LiME version 1 example from AVML tests:

```text
45 4d 69 4c 01 00 00 00
00 10 00 00 00 00 00 00
00 00 02 00 00 00 00 00
00 00 00 00 00 00 00 00
```

This represents:

- Magic: LiME.
- Version: 1.
- Start: `0x1000`.
- End exclusive: `0x20001`.
- End encoded: `0x20000`.

AVML compressed version 2 example:

```text
41 56 4d 4c 02 00 00 00
00 10 00 00 00 00 00 00
00 00 02 00 00 00 00 00
00 00 00 00 00 00 00 00
```

GOFVML should include these as direct unit tests.

## Proposed Go Interfaces

```go
type Encoder interface {
    WriteBlock(context context.Context, block Block, source Source) error
    Close() error
}

type Decoder interface {
    Next() (Header, io.Reader, error)
}

type Format int

const (
    FormatRaw Format = iota
    FormatLiME
    FormatAVMLCompressed
)
```

Avoid letting the acquisition orchestrator know Snappy or header details. It should hand physical blocks to an encoder.

## Compatibility Checklist

- [ ] Header byte order matches AVML.
- [ ] Magic/version validation matches AVML.
- [ ] Padding must be zero.
- [ ] Version 2 writes compressed length trailer after each Snappy frame.
- [ ] Version 2 splits ranges larger than 16 MiB.
- [ ] Zero blocks are skipped.
- [ ] Raw conversion fills gaps before blocks.
- [ ] Same-format conversion returns a no-op error/status.
- [ ] Conversion tests round-trip sparse raw data.
- [ ] Recompressing a decompressed AVML image produces byte-identical output for AVML fixtures where compression library behavior is stable.
- [ ] Default LiME output can be opened by Volatility 3's LiME layer.
- [ ] No GOFVML metadata is embedded in LiME output; metadata goes in an optional sidecar.
