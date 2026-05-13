# Volatility 3 Compatibility Plan

## Compatibility Goal

GOFVML must produce memory images that work naturally with Volatility 3 Linux analysis workflows. This means the primary physical-memory output should be a LiME-compatible image, with AVML-compressed output supported as a secondary convenience format.

Volatility 3 compatibility is not just "write bytes to disk." It requires:

- A physical memory image format Volatility can stack.
- Accurate Linux kernel banner bytes in the captured memory.
- A corresponding Linux symbol table, usually an ISF JSON or compressed JSON file.
- Stable segment ordering and header semantics.
- Validation with real Volatility plugins, not only GOFVML's own decoder.

Sources consulted:

- [Volatility 3 Linux Tutorial](https://volatility3.readthedocs.io/en/latest/getting-started-linux-tutorial.html)
- [Volatility 3 Creating New Symbol Tables](https://volatility3.readthedocs.io/en/latest/symbol-tables.html)
- [Volatility 3 LiME layer source docs](https://volatility3.readthedocs.io/en/v2.27.0/_modules/volatility3/framework/layers/lime.html)
- [Volatility 3 AVML layer docs](https://volatility3.readthedocs.io/en/stable/volatility3.framework.layers.avml.html)
- [Volatility 3 Basics: memory layers](https://volatility3.readthedocs.io/en/v2.4.0/basics.html)

## What Volatility 3 Expects

The Volatility 3 Linux tutorial says Volatility does not acquire memory and points to AVML as an example acquisition tool. It then uses the CLI shape:

```text
python3 vol.py -f <path to memory image> <plugin_name> <plugin_option>
```

GOFVML's physical output should therefore be directly usable as:

```text
python3 vol.py -f host.lime banners
python3 vol.py -f host.lime linux.pslist
python3 vol.py -f host.lime linux.pstree
python3 vol.py -f host.lime linux.bash
python3 vol.py -f host.lime linux.malfind
```

The first validation plugin should be `banners`, because Volatility's Linux symbol selection depends on exact kernel banner matching.

## LiME Layer Requirements

Volatility's LiME layer treats LiME as a segmented physical memory layer. Its documented source uses:

- Magic: `0x4C694D45`
- Version: `1`
- Header struct: `<IIQQQ`
- Segment length: `end - start + 1`
- Segment file offset: `header_offset + header_size`

GOFVML must preserve this exactly for `--format lime`.

Important implications:

- The `End` field on disk is inclusive.
- Segments must appear in non-decreasing physical order.
- A segment with `start < previous_max` or `end < start` can fail layer construction.
- At least one segment must be present.
- Missing physical ranges are acceptable because LiME is a segmented layer.

Compatibility contract:

```text
[LiME header][raw bytes][LiME header][raw bytes]...
```

No GOFVML-specific metadata should be inserted into the LiME stream. If we need metadata, write it to a sidecar file.

## AVML-Compressed Layer Requirements

Volatility 3 has an AVML layer for compressed AVML files. Its docs say it reads AVML files and hides compression from the user, but random access is not allowed.

GOFVML should keep AVML-compressed output compatible, but LiME should remain the recommended default for broad Volatility workflows.

Reasons:

- LiME is simpler and random-access friendly as a segmented physical layer.
- AVML compressed requires Snappy support in the Volatility environment.
- The Volatility AVML layer explicitly warns that random access is not available.
- Some workflows and tools outside Volatility may understand LiME but not AVML compression.

Recommended CLI default:

```text
gofvml physical --format lime host.lime
```

Convenience:

```text
gofvml physical --compress host.avml
gofvml physical --format avml-compressed host.avml
```

## Symbol Table Workflow

Volatility 3 Linux analysis requires matching symbols. Its symbol table docs explain:

- Linux/macOS symbol files contain an identifying operating system banner.
- Volatility caches banner-to-symbol mappings.
- For Linux, banner strings must match exactly, not just by version number.
- `dwarf2json linux --elf <debug-kernel>` generates an ISF JSON file.
- Symbol files are placed under `volatility3/symbols`, commonly under `volatility3/symbols/linux`.

GOFVML should help the operator bridge acquisition and symbol selection by optionally producing a sidecar metadata file.

Recommended sidecar:

```text
host.lime
host.lime.gofvml.json
```

Sidecar fields:

```json
{
  "format": "lime",
  "tool": "gofvml",
  "toolVersion": "0.1.0",
  "kernelRelease": "6.8.0-...",
  "kernelVersion": "Linux version ...",
  "hostname": "target-host",
  "architecture": "x86_64",
  "acquisitionSource": "/proc/kcore",
  "physicalRanges": [
    {"start": "0x1000", "endInclusive": "0x9ffff"}
  ],
  "volatility": {
    "recommendedFirstCommand": "python3 vol.py -f host.lime banners",
    "symbolsDirectoryHint": "volatility3/symbols/linux"
  }
}
```

This sidecar must not be required for Volatility. It is an operator aid only.

## Volatility-Oriented Validation Commands

Minimum validation on a captured physical image:

```text
python3 vol.py -f host.lime banners
python3 vol.py -f host.lime linux.boottime
python3 vol.py -f host.lime linux.pslist
python3 vol.py -f host.lime linux.pstree
```

Expanded incident-response validation:

```text
python3 vol.py -f host.lime linux.bash
python3 vol.py -f host.lime linux.lsmod
python3 vol.py -f host.lime linux.kmsg
python3 vol.py -f host.lime linux.elfs
python3 vol.py -f host.lime linux.check_creds
python3 vol.py -f host.lime linux.vmayarascan
python3 vol.py -f host.lime linux.malfind
```

The docs should make clear that many Linux plugins will not work until the correct symbol table is installed.

## Compatibility Test Plan

### Header-Level Tests

GOFVML tests must assert:

- First 4 bytes are LiME magic for default Volatility output.
- Version is `1`.
- Header is exactly 32 bytes.
- Header byte order is little endian.
- On-disk end address is inclusive.
- Segment payload length equals `end - start + 1`.
- Segment file offsets line up exactly with Volatility's parser expectations.

### Volatility Smoke Tests

Add optional integration tests that run only when `VOLATILITY3_PATH` or `VOL_PY` is set:

```text
VOL_PY=/path/to/vol.py go test ./internal/volatilitytest -run TestVolatilitySmoke
```

The test should:

1. Acquire or use a known fixture image.
2. Run `vol.py -f <image> banners`.
3. Assert command exits successfully.
4. Capture banner output into test artifacts.

For full plugin tests, require a symbols directory:

```text
VOL_PY=/path/to/vol.py VOL_SYMBOLS=/path/to/volatility3/symbols go test ./internal/volatilitytest -run TestVolatilityLinuxPlugins
```

### AVML-Compressed Tests

If testing compressed output with Volatility:

- Ensure Python Snappy dependency is installed in the Volatility environment.
- Run `banners` on the compressed image.
- Expect slower or more constrained access than LiME.

## Output Format Policy

GOFVML should classify output formats by consumer:

| Format | Primary consumer | Volatility support | Recommended use |
| --- | --- | --- | --- |
| `lime` | Volatility and general forensic tools | Strong | Default physical acquisition format. |
| `avml-compressed` | Volatility AVML layer and storage-constrained workflows | Supported but less random-access friendly | Use when disk/network size matters. |
| `raw` | Low-level tools and conversion | Tool-dependent | Conversion/testing, not default. |
| `gofvml-process-v1` | GOFVML library and IR framework workflows | Not native Volatility input | PID-scoped process dumps and richer metadata. |

## Process Dump Compatibility With Volatility

PID-scoped dumps should not be marketed as full Volatility Linux images. They are process virtual memory captures, while Volatility's Linux plugins generally expect a physical memory layer plus symbols so it can reconstruct kernel and process structures.

GOFVML can still make process dumps useful alongside Volatility:

- Include PID, process name, maps, permissions, and virtual address ranges.
- Offer `raw-dir` export with one file per mapping and a manifest.
- Include enough metadata to correlate a PID dump with a full Volatility image.
- Later, consider a Volatility plugin or layer for `gofvml-process-v1`, but that is a separate project.

Do not imply that `python3 vol.py -f process.gopmem linux.pslist` will work. It should not be expected to.

## Volatility Compatibility Definition Of Done

GOFVML physical acquisition reaches Volatility compatibility when:

- `gofvml physical --format lime host.lime` emits a valid LiME segmented image.
- Volatility's `banners` plugin detects Linux kernel banners from the image.
- With matching symbols installed, `linux.pslist`, `linux.pstree`, and `linux.boottime` run successfully on at least one real Linux VM image.
- The same image can be converted by GOFVML without breaking Volatility ingestion.
- The sidecar metadata helps locate the kernel release/banner but is not required for Volatility.

