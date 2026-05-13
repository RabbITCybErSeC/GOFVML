# PID-Scoped Memory Dumping

## Why PID Dumping Is A Separate Mode

AVML is a physical memory acquisition tool. A PID dump is process virtual memory acquisition. These are related forensic workflows, but the data model is different:

- Physical dump addresses are machine physical addresses.
- PID dump addresses are process virtual addresses.
- Physical ranges come from `/proc/iomem`.
- PID ranges come from `/proc/<pid>/maps`.
- Physical reads use `/dev/crash`, `/proc/kcore`, or `/dev/mem`.
- PID reads use `/proc/<pid>/mem`, `process_vm_readv`, ptrace-assisted reads, or core dumps.

GOFVML should expose PID dumping as a sibling mode, not a flag that mutates the physical snapshot pipeline beyond recognition.

Recommended CLI:

```text
gofvml physical [options] <output>
gofvml process --pid <pid> [options] <output>
gofvml process --pid <pid> --pid <pid> [options] <output>
```

Possible shorthand:

```text
gofvml --pid 1234 output.gopmem
```

But internally it should route to `process` mode.

## Linux Process Memory Sources

### `/proc/<pid>/maps`

Defines virtual memory mappings.

Example fields:

```text
address           perms offset  dev   inode pathname
00400000-00452000 r-xp 00000000 08:02 12345 /usr/bin/example
```

Fields to retain:

- Start virtual address.
- End virtual address.
- Permissions (`rwxsp`).
- File offset.
- Device.
- Inode.
- Pathname, including special names like `[heap]`, `[stack]`, `[vdso]`.

Default dump filter:

- Include readable mappings: permissions beginning with `r`.
- Exclude non-readable mappings by default.
- Allow `--include-non-readable` to record metadata and attempted read errors, but do not expect payload.

### `/proc/<pid>/mem`

Byte-addressable view of process virtual memory.

Access constraints:

- Usually requires same UID and ptrace permission, or elevated privileges.
- Yama `ptrace_scope` can block access.
- Process dumpability matters.
- Namespaces and LSMs can affect access.
- Mappings can vanish while reading.

Recommended behavior:

- Open `/proc/<pid>/mem` once after reading maps.
- For each readable mapping, seek to virtual start and read in chunks.
- Record partial reads and errno per range.
- Continue by default when a mapping fails, unless `--strict` is set.

### `process_vm_readv`

Alternative direct syscall for reading another process.

Pros:

- Avoids seeking in `/proc/<pid>/mem`.
- Can read ranges into buffers efficiently.

Cons:

- Still subject to ptrace permission checks.
- More syscall-specific complexity.
- Chunking and partial reads require careful handling.

Recommendation:

- MVP uses `/proc/<pid>/mem`.
- Add `--method process-vm-readv` later if performance or reliability demands it.

### Ptrace Stop/Resume

Consistent dumps are hard if the process keeps mutating mappings. Options:

- No stop: least intrusive, most likely inconsistent.
- `ptrace(PTRACE_ATTACH)` or `PTRACE_SEIZE` plus interrupt: more consistent, intrusive, permission-sensitive.
- Freeze cgroup: useful for controlled environments, broader side effects.

MVP recommendation:

- Default: no stop, with metadata warning that dump is best-effort.
- Optional: `--suspend` to stop target during dump, implemented only after careful testing.

## GOFVML Process Dump Container

Raw process memory is not enough because virtual mappings are sparse and meaningful. GOFVML should define a native process container.

Proposed extension: `gofvml-process-v1`.

High-level layout:

```text
[file header]
[metadata JSON length][metadata JSON]
[range block header][payload]
[range block header][payload]
...
```

Metadata should include:

- Tool name and version.
- Dump format version.
- Hostname.
- Kernel release.
- Timestamp UTC.
- PID.
- Process start time if available.
- Command line.
- Executable path.
- UID/GID if available.
- Maps snapshot.
- Acquisition options.
- Errors and skipped ranges.

Range block header should include:

- Virtual start.
- Virtual end.
- Original mapping index.
- Payload offset/length.
- Compression type.
- Error status if payload missing or partial.

Compression:

- Use no compression first.
- Add Snappy block compression after the container is stable.

Why not LiME for PID mode:

- LiME headers imply physical memory ranges.
- Permissions, pathnames, file offsets, and virtual address semantics would be lost.
- Analyst tooling would need out-of-band metadata.

Interoperability option:

- `--format raw-dir` emits one file per mapping plus `manifest.json`.
- Useful for quick inspection and testing.

## Mapping Filters

Required filters:

- `--readable-only` default true.
- `--include-perm r-xp` or `--perm r` style filter.
- `--range START-END` for explicit virtual address ranges.
- `--name REGEX` for pathname filtering.
- `--max-bytes` safety cap.

Recommended default exclusions:

- Do not exclude `[vdso]`, `[vvar]`, or file-backed mappings by default; they can be forensically useful.
- Exclude non-readable mappings unless user opts in.

## Process Lifetime Hazards

The process may:

- Exit before maps are read.
- Exit after maps are read but before mem is opened.
- Unmap a region while it is being read.
- Change permissions during acquisition.
- Race with copy-on-write changes.

GOFVML should record these as first-class acquisition events, not hide them.

Manifest event example:

```json
{
  "range": "7f0000000000-7f0000010000",
  "event": "short_read",
  "errno": "EIO",
  "bytes_read": 8192
}
```

## PID Dump Algorithm

MVP algorithm:

1. Validate PID exists.
2. Read `/proc/<pid>/status`, `/proc/<pid>/cmdline`, `/proc/<pid>/exe`, and `/proc/<pid>/maps`.
3. Parse mappings and apply filters.
4. Open output with private no-follow behavior.
5. Write container header and metadata placeholder or buffered metadata.
6. Open `/proc/<pid>/mem`.
7. For each selected mapping:
   - Seek to mapping start.
   - Read in bounded chunks.
   - Write block payload.
   - Record success, short read, or error.
8. Finalize metadata/index.

Simpler MVP container option:

- Buffer block metadata in memory.
- Write payloads to a temporary file.
- At finalize, write metadata plus payloads to final destination.

Better streaming container option:

- Write fixed file header.
- Stream blocks.
- Append final index at EOF.
- Header points to index offset.

Recommendation:

- Use EOF index design for MVP because it avoids needing to seek back into output on stdout-like targets.

## Access Diagnostics

PID dumping failures are often policy failures. GOFVML should print actionable diagnostics:

- Current UID/EUID.
- Target UID from `/proc/<pid>/status`.
- Whether `/proc/sys/kernel/yama/ptrace_scope` exists and its value.
- Whether `/proc/<pid>/mem` open failed.
- Whether maps were readable.
- Suggested `sudo` only as an operational hint, not a magic fix.

## Security And Safety

Rules:

- Never write to `/proc/<pid>/mem`.
- Open process memory read-only.
- Avoid ptrace attach in MVP.
- Output files are `0600`.
- Avoid symlink-following for output.
- Record command line carefully because it may contain secrets. Since memory dumps are already sensitive, metadata sensitivity is acceptable but should be documented.

## Tests For PID Mode

Unit tests:

- Parse representative `/proc/<pid>/maps` lines.
- Filter by permissions, path, and address range.
- Serialize/deserialize process dump manifest.
- Handle short reads and partial blocks.

Integration tests:

- Spawn a child process with known memory content.
- Dump child by PID.
- Verify known marker is present.
- Test unreadable mapping behavior if possible.
- Test process exits mid-dump with a controlled short-lived child.

Privileged tests:

- Same UID process dump.
- Root dump of another UID process.
- Yama restricted environment behavior.

