## ADDED Requirements

### Requirement: Post-acquisition upload
GOFVML SHALL support uploading completed local artifacts after acquisition or as a standalone upload workflow.

#### Scenario: Upload existing artifact
- **WHEN** an operator uploads an existing image artifact to a supported destination
- **THEN** GOFVML MUST stream the local file to the destination and report transfer success or failure.

#### Scenario: Upload after acquisition
- **WHEN** an operator requests upload after successful acquisition
- **THEN** GOFVML MUST upload the completed local artifact only after acquisition has finished successfully.

### Requirement: HTTP PUT upload
GOFVML SHALL support HTTP PUT upload as the initial upload destination.

#### Scenario: Successful HTTP PUT
- **WHEN** the destination accepts the HTTP PUT request
- **THEN** GOFVML MUST report upload success with byte counts when available.

#### Scenario: Failed HTTP PUT
- **WHEN** the destination rejects or interrupts the HTTP PUT request
- **THEN** GOFVML MUST report the failure and preserve the local artifact.

### Requirement: Destination-neutral upload design
GOFVML SHALL keep upload workflow behavior separate from destination-specific transport details.

#### Scenario: Future upload destination
- **WHEN** a future destination such as an S3 bucket is added
- **THEN** GOFVML MUST reuse common artifact preservation, progress, result, and diagnostic behavior.

### Requirement: Evidence preservation on upload failure
GOFVML SHALL never delete a local artifact unless upload success is confirmed.

#### Scenario: Delete after successful upload
- **WHEN** an operator requests deletion after upload and upload succeeds
- **THEN** GOFVML MAY delete the local artifact and MUST report that deletion.

#### Scenario: Preserve after failed upload
- **WHEN** an operator requests deletion after upload and upload fails
- **THEN** GOFVML MUST preserve the local artifact.

### Requirement: Upload progress reporting
GOFVML SHALL expose upload progress through the same progress mechanism used by library callers and CLI output.

#### Scenario: Progress callback
- **WHEN** an upload transfers bytes
- **THEN** GOFVML MUST be able to emit progress events without coupling upload logic to terminal output.
