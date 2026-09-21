# Unified native firmware: resource and release evidence

Measured 2026-09-07 for `cmd/screen-device`, Pico 2 W, in the existing Dev Container. This records the unflashed candidate, not target acceptance. The historical [resource budget](resource-budget.md) describes the buffered/HTML firmware and is not this runtime's RAM budget.

## Reproduce and identify

Run only inside the project's already-running Dev Container, from this experiment:

```sh
tinygo build -target=pico2-w -scheduler=tasks -size=full -print-stacks -print-allocs=. -o build/screen-device-review.uf2 ./cmd/screen-device > build/screen-device-review-resources.txt 2>&1
tinygo build -target=pico2-w -scheduler=tasks -o build/screen-device-review.elf ./cmd/screen-device
tinygo list -deps -json -target=pico2-w ./cmd/screen-device > build/screen-device-review-deps.json
arm-none-eabi-readelf -SW build/screen-device-review.elf
arm-none-eabi-readelf -sW build/screen-device-review.elf
```

Actual compiler: TinyGo **0.41.1**, LLVM **20.1.1**, using Go **1.26.8** at `/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.8.linux-arm64`. `go env GOTOOLCHAIN` is `auto`; both `go version` and `tinygo version/env` resolve this patched GOROOT. The build succeeded without disabling compatibility/bounds checks or changing the default conservative GC, `z` optimization or panic strategy. Pinned TinyGo's loader includes Go 1.26 standard-library overrides. This proves compilation compatibility, not execution of Go 1.26 crypto/runtime code on the MCU.

Candidate `build/screen-device-review.uf2`: **1,393,152 bytes**, SHA-256 `d6c5b0726bc08c4c9ed1bff4594c1685ae24779151ccd45a3851e7a28c01c5ef`. UF2 container size is not flash consumption. This hash identifies this review build, not a later rebuilt/repacked release file.

## Linked memory, not just the size headline

| Measurement | Bytes | Interpretation |
| --- | ---: | --- |
| Compiler flash total | 696,380 | Code + constants + initialized data |
| Compiler `ram` headline | 8,512 | Incomplete static-reservation accounting; do not use as total RAM |
| Linked `.data` | 26,204 | Initialized tables/data and RAM-resident functions |
| Linked `.bss` | 4,400 | Zero-initialized storage |
| Linked `.stack` + `.stack1` | 4,096 | Two 2,048-byte system-stack reservations |
| Inter-section padding | 4 | Before `.bss` |
| **Actual linked pre-heap reservation** | **34,704** | `_heap_start - 0x20000000` |
| Linker heap region | 489,584 | `0x20008790..0x20080000`, before GC metadata/objects |
| Conservative-GC state metadata | 7,533 | Computed from this target's allocator formula |
| Allocatable heap blocks | 482,048 | 30,128 blocks × 16 bytes; **not free memory** |

The target linker uses the first **512 KiB** of RP2350 SRAM; the additional two 4 KiB banks are not added to this heap. `arm.ld` puts `.ramfuncs*` into RAM-backed `.data`, and the size summary also classifies some initialized tables as read-only data. ELF section addresses, not package categories, determine occupied SRAM.

The linked image ends at `__flash_data_start=0x100aa03c`, well below the physical flash tail. `__flash_data_end=0x10400000`; `screenboot` reserves the last four erase blocks, with credentials in blocks -4/-3 and durable epochs in -2/-1. This candidate build supplies no SSID, passphrase, device key or enrollment record. It does not initialize or erase those persisted records merely by being built.

## Tasks and dynamic memory

Normal firmware has four Go tasks: owner/main, one USB writer, one network worker and one packet pump. Network construction failure leaves only owner/main and USB writer. There is no goroutine per frame, reconnect or timeout; a stalled hardware worker is not replaced.

The ELF's four `internal/task.stackSizes` entries are all **8,192 bytes**, the Pico 2 target fallback. Task stacks therefore reserve **32,768 heap payload bytes**. Including four 32-byte task objects and this allocator's 8-byte aligned object header/16-byte block rounding gives **33,024 heap bytes** for those stacks and task objects, before their closures or other objects. These bytes are additional to the 34,704 linked reservation, not included in TinyGo's static RAM headline.

`-print-stacks` reports all five roots (reset handler and the four tasks) as `recursive, runtime.nilPanic may call itself`. Thus **no finite maximum call-stack usage is proven**. The 8 KiB allocation is a configured limit, not a measured high-water mark; stack canaries do not establish that it is sufficient for every path.

Selected ARM object sizes below come from this ELF's DWARF, not native `unsafe.Sizeof`. Inline arrays are already included; do not count them twice. These are allocation payloads, excluding GC headers/rounding and referenced objects.

| Owner/object | Payload bytes | Bound / separately allocated storage |
| --- | ---: | --- |
| `screenlink.Device` | 1,104 | One; includes shared 1,024-byte PackBits output scratch |
| `streamsession.Session` | 256 | One; one high-water mark/terminal record, no pixel history |
| `streamrx.Receiver` + SHA-256 digest | 184 + 120 | One current receiver; replaced receivers await GC |
| Panel `Driver` / `Stream` / `ChunkedSink` | 176 / 12 / 12 | Driver includes 100-byte inversion row; chunk adapter only borrows slices |
| `screenlink.Connection` | 2,352 | Includes two 1,056-byte arrays + 88-byte reply body; active transport ownership is serialized |
| `screenapp.App` / `screenusb.Session` | 304 / 616 | Includes 256-byte input, 32-byte prefix and 512-byte provision record |
| `usbwrite.Writer` | 48 | Plus one 512-byte buffer and three bounded channels; created once per boot |
| `screenowner.Owner` / provision `Store` | 216 / 536 | Store includes its 512-byte page; configuration strings are separately bounded by the codec |
| `screenbridge.Bridge` | 24 | One-slot request/response queues; element sizes 48/256, plus channel runtime storage |
| `screenwifi.Worker` / packet pump | 80 / 1,624 | Pump includes 1,536-byte packet; two one-slot control channels |
| `stacknet.Client` | 3,448 | Includes TCP connection and two 1,536-byte buffers; TX packet queue has three slots |
| lneto `StackAsync` | 1,560 | Plus bounded port/ARP/DHCP/DNS allocations; replaced on network reset |
| CYW `Device` | 6,624 | Includes three 2,048-byte bus/IOCTL/RX buffers; one instance |
| AEAD `RecordStream` / `Session` | 2,164 / 40 | Stream includes 1,088-byte envelope + 1,056-byte plaintext arrays |
| AES-GCM objects | 2 × 492 | Directional sessions; handshake has additional temporary AES/HMAC objects |

Network `Bridge.serveRecords` additionally uses a 1,056-byte record buffer. `securetransport.Session.Open` uses another 1,056-byte authentication scratch; the allocation report marks it escaping. The report also shows per-exchange error/closure/timer allocations and handshake temporaries. It is an **allocation-site report, not runtime allocation counts**: its zero counters are not evidence of allocation-free I/O. PackBits decoding and the 100-byte panel chunk adapter separately have zero-steady-allocation regression coverage.

The logical data path has bounded record/chunk memory; neither 48,000-byte plane nor HTML/renderer/frame history is retained on the Pico. Both full planes are streamed into controller RAM, authenticated/ordered/hashed completely before the refresh command. Old connection, receiver, stack and credential objects become garbage; conservative retention, fragmentation, GC pauses, handshake peaks, repeated provisioning/reconnect churn and stack high-water remain **unmeasured on hardware**. Do not sum this partial object inventory into a claimed worst-case heap bound or subtract it from SRAM to advertise free headroom. A whole-runtime peak bound has not been established.

## Firmware closure and security scope

The selected dependency closure excludes `manager`, `engine`, `blitzworker`, `htmlruntime`, `render`, `devicelink`, Canvas/parse and `golang.org/x/net/html`. It includes only board/provisioning/session/transport/panel functionality and its networking/crypto dependencies. Legacy buffered `cmd/device` is not the unified entry point. The firmware still includes TinyGo/C runtime helpers and a vendor radio blob; “pure Go” describes the application/server implementation, not a claim that every linked byte is Go source.

`cywradio.Join` accepts only `AuthWPA3SAE` and passes `JoinAuthWPA3`, never mixed mode or default authentication. Pinned CYW v0.1.1 selects SAE, AES and **required** management-frame protection for that value. Unsupported provisioning auth is rejected; no WPA1/WPA2 fallback is requested. Association/rejection on an actual AP remains untested for this candidate.

Bluetooth is **disabled, not entirely absent from the binary**. `DefaultWifiConfig` sets Wi-Fi-only mode, and no HCI service is opened. The linker nevertheless retains `ReadHCI`/`WriteHCI` helpers and `btfw.bin` (6,164 bytes). Exact embedded data symbols identify Wi-Fi firmware (230,321), CLM (4,752) and that dormant BT patch (6,164): total 241,237 flash bytes. Do not claim complete Bluetooth code/blob exclusion.

Startup does not refresh a boot image: the panel enable is low before serving USB. A missing/corrupt durable epoch yields physical control-only recovery; no fake boot identity is used. Normal blank credentials keep network admission disabled. Optional radio construction failure retains normal USB pixel/provision service and numeric setup-failure status. USB DTR priority revokes queued network access and aborts incomplete staging; an already executing synchronous physical refresh finishes first. Provision/rotate/erase fence ownership before storage changes and always reload authoritative storage afterward, including uncertain writes.

Network peers authenticate before owner admission, using EPN2 mutual HMAC and directional AES-GCM. Each authenticated envelope is exactly one EPS2 request; partial/bundled/reply records are rejected before owner dispatch. Frame IDs, digests, nonwrapping leases and terminal proof support Query reconciliation after a lost ACK; no blind display retry is inferred from reconnect. Read-only status contains bounded numeric failures, not source/credentials. See [runtime constraints](wifi-runtime-constraints.md) and [ADR-014](../decisions/014-durable-device-sessions.md).

## Verified without hardware; remaining acceptance

Focused `go test -race` passed for `screenboot`, `epochstore`, `provision`, `screenapp`, `screenowner`, `screenusb`, `usbwrite`, `screenwifi`, `cywradio`, `stacknet`, `screenpeer`, `securetransport`, `screenbridge`, `screenlink`, `streamsession`, `streamrx`, `packbits`, `waveshare75` and `integration`. The captured output is `build/screen-device-review-tests.txt`.

The production-path integration tests cover real HTTPS → native engine → EPN2/AEAD → bridge → owner → streaming recording panel. Complete SPI/GPIO/delay traces match the buffered driver for both planes; lost Commit ACK reconciles without additional physical I/O; USB preemption aborts staging without refresh. Separate wrong-key/tamper fixtures prove no panel admission, and packed/raw uploads produce identical traces. These use simulated panel/network/storage boundaries, not an actual CYW radio, USB CDC controller or display.

Manual candidate acceptance is deferred until the final release candidate is ready. It must still establish visible output and plane polarity, BUSY timing and power-off, real WPA3-only association/no downgrade, USB DTR/reconnect behavior, interruption/reboot recovery, long-lived networking and runtime memory/stack stability. The pinned radio's Init/Join can remain inside unbounded upstream loops: fixed worker isolation and between-call cancellation preserve the design boundary but do not prove hard deadlines for those calls. No power/current/energy estimate or measurement is claimed.

## Release notices and evidence

`build/native-firmware-notices-input.txt` lists actual selected module versions/directories, Go/TinyGo licenses, compiler-rt and pinned picolibc/newlib notices, plus all three embedded radio blob hashes. The installed TinyGo image omits its top-level LICENSE; the exact v0.41.1 source module provides it. Picolibc notices were retrieved from the exact submodule revision pinned by TinyGo v0.41.1. CYW binary assets use the separate **Permissive Binary License 1.0**, not the driver's MIT license; keep that text with the candidate. This inventory is not a legal-conformance claim.

Build evidence is intentionally under ignored `build/`: UF2/ELF, full resources/stacks/allocation-site report, dependency JSON/text, symbol/DWARF reports, focused test output and notice inputs. No flash, hardware run or commit was performed for this review.
