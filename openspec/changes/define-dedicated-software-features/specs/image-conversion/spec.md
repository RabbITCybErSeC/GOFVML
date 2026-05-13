## ADDED Requirements

### Requirement: Supported conversion formats
GOFVML SHALL convert between raw physical images, LiME images, and AVML-compressed images.

#### Scenario: Raw to encoded image
- **WHEN** a raw physical image is converted to LiME or AVML-compressed output
- **THEN** GOFVML MUST process the raw input in deterministic physical-offset chunks and skip all-zero chunks.

#### Scenario: Encoded image to raw
- **WHEN** a LiME or AVML-compressed image is converted to raw output
- **THEN** GOFVML MUST fill gaps before encoded blocks with zero bytes.

### Requirement: LiME format compatibility
GOFVML SHALL encode and decode LiME headers using AVML-compatible layout and validation.

#### Scenario: Write LiME header
- **WHEN** GOFVML writes a LiME block
- **THEN** the header MUST be 32 bytes, little-endian, magic `0x4c694d45`, version `1`, exclusive-end encoded as inclusive-end, and zero padding.

#### Scenario: Reject invalid LiME header
- **WHEN** a LiME header has unknown magic, unsupported version, non-zero padding, overflowed end, or end before start
- **THEN** GOFVML MUST reject the header with a structured conversion error.

### Requirement: AVML-compressed format compatibility
GOFVML SHALL encode and decode AVML-compressed blocks using AVML-compatible version 2 behavior.

#### Scenario: Write compressed block
- **WHEN** GOFVML writes an AVML-compressed block
- **THEN** it MUST write the AVML magic, version `2`, Snappy-framed payload, and an 8-byte little-endian compressed payload length trailer.

#### Scenario: Split large compressed range
- **WHEN** an AVML-compressed output range is larger than 16 MiB
- **THEN** GOFVML MUST split it into AVML-compatible block sizes.

### Requirement: Conversion no-op handling
GOFVML SHALL detect same-format conversion requests.

#### Scenario: Same source and target format
- **WHEN** the requested conversion source format and target format are the same
- **THEN** GOFVML MUST return a clear no-op status or error instead of rewriting the file as if conversion occurred.

### Requirement: Deterministic sparse behavior
GOFVML SHALL preserve AVML-compatible sparse image behavior during conversion.

#### Scenario: Trailing skipped zero block
- **WHEN** an encoded image omits a trailing all-zero block
- **THEN** raw conversion MUST end at the last encoded block rather than inventing an unknown trailing memory extent.
