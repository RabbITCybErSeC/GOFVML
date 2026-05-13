## ADDED Requirements

### Requirement: Physical memory acquisition mode
GOFVML SHALL provide a physical acquisition capability for whole-machine Linux volatile memory collection using supported physical memory sources.

#### Scenario: Acquire physical memory from selected source
- **WHEN** an operator starts physical acquisition with an available supported source
- **THEN** GOFVML MUST read physical memory ranges from that source and write an encoded physical memory artifact.

#### Scenario: Unsupported host source
- **WHEN** no supported physical memory source is available or accessible
- **THEN** GOFVML MUST fail without creating a misleading successful artifact and MUST report source availability diagnostics.

### Requirement: System RAM range discovery
GOFVML SHALL discover physical memory ranges from `/proc/iomem` and include top-level `System RAM` ranges for acquisition.

#### Scenario: Merge adjacent and overlapping ranges
- **WHEN** parsed `System RAM` ranges overlap or touch
- **THEN** GOFVML MUST merge them before building acquisition blocks.

#### Scenario: Permission-denied iomem view
- **WHEN** `/proc/iomem` exposes unusable zeroed ranges because host policy hides addresses
- **THEN** GOFVML MUST report a clear range discovery failure instead of silently acquiring an empty image.

### Requirement: Physical source selection
GOFVML SHALL support `/dev/crash`, `/proc/kcore`, and `/dev/mem` as physical acquisition sources on Linux.

#### Scenario: Auto source order for regular files
- **WHEN** the operator selects automatic source discovery for a regular output file
- **THEN** GOFVML MUST try `/dev/crash`, then `/proc/kcore`, then `/dev/mem` in that order.

#### Scenario: Explicit source failure
- **WHEN** the operator requests a specific source that cannot be opened or mapped
- **THEN** GOFVML MUST fail with diagnostics for that source and MUST NOT fall back to a different source.

### Requirement: Physical output formats
GOFVML SHALL write physical acquisition output as LiME or AVML-compressed physical memory artifacts.

#### Scenario: Default Volatility-compatible output
- **WHEN** the operator does not request compression
- **THEN** GOFVML MUST write LiME-compatible physical memory output without embedding GOFVML-specific metadata in the image stream.

#### Scenario: Compressed physical output
- **WHEN** the operator requests AVML-compatible compression
- **THEN** GOFVML MUST write AVML-compressed blocks with the version 2 header, Snappy-framed payload, and compressed-length trailer.

### Requirement: Safe artifact creation
GOFVML SHALL create acquisition output files with private permissions and symlink-safe behavior on Unix-like systems.

#### Scenario: Create local output file
- **WHEN** GOFVML creates a local physical acquisition artifact
- **THEN** the file MUST be created with owner-only permissions and MUST NOT follow an existing symlink.

### Requirement: Zero block handling
GOFVML SHALL omit all-zero physical memory blocks according to AVML-compatible behavior.

#### Scenario: Omit zero block
- **WHEN** a physical acquisition block contains only zero bytes and is eligible for zero checking
- **THEN** GOFVML MUST omit that block from LiME and AVML-compressed output.
