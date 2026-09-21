# Pi 5 / Pironman 5 / 2.13-inch service investigation

Status: hardware identified; implementation boundary proposed, not a wiring or deployment instruction.
Date: 2026-09-08.

## Requested result

- Preserve the existing Pico firmware and working Go rendering code.
- Add direct Raspberry Pi 5 display control as a Debian service written in Go,
  accepting HTML and image submissions from Docker clients. Updated user
  decision: ordinary Go compilation is allowed; keep shared modules usable
  with TinyGo. Linux-specific service code need not compile for Pico.
- Keep Pironman RGB, OLED and fans working; select a non-conflicting connection.
- Initial display description was Waveshare 2.13-inch e-Paper HAT "4V".
  Resolved by the user's second photograph: PCB reads `2.13inch e-Paper HAT`,
  `Rev2.1`, with a `V4` sticker. Eight-pin connector labels are VCC, GND,
  DIN, CLK, CS, DC, RST, BUSY. There is no separate PWR connector signal.
  This is not the earlier Pico-specific module or the separate Rev2.3 HAT.

## Verified software evidence

The existing Dev Container `35e5044beea1` is running. Its toolchain reports
TinyGo 0.42.0, LLVM 22.1.4, Go 1.26.8, Linux ARM64. The worktree was clean
before this note. No runtime, dependency, container or hardware changes made.

Inside `experiments/03-remote-epaper`, this diagnostic command failed:

```sh
GOOS=linux GOARCH=arm64 tinygo build \
  -o /tmp/epaper-manager-tinygo-probe ./cmd/epaper-manager
```

Errors: undefined `os.Root` and `os.OpenRoot` in
`hostprovision/{pending,read,snapshot}.go`. This establishes the first compile
barrier, not the complete compatibility list. Do not remove confined file
access to silence it. A separate service entry point may avoid Pico
provisioning dependencies; that approach remains unimplemented and unverified.
The later approval to use ordinary Go removes this blocker for the Debian
service. Preserve the existing confined-file APIs instead of weakening them
to accommodate TinyGo.

An independent renderer check succeeded with the same installed toolchain:

```sh
GOOS=linux GOARCH=arm64 tinygo build \
  -o /tmp/epaper-engine-tinygo-probe ./cmd/engine-preview
/tmp/epaper-engine-tinygo-probe -width 250 -height 122 \
  engine/testdata/dashboard.html /tmp/epaper-engine-tinygo-probe.png
```

Both commands exited 0. The renderer emitted `clipped` warnings for elements
10, 11, 13, 14 and 15. This proves this renderer fixture can execute under
TinyGo/Linux; it does not prove all rendering features, microcontroller
resource limits, visual quality, networking or panel output. The viewport
is a software probe, not confirmation of the user's exact display model.

The installed TinyGo source contains `src/net/netdev_native.go`, a native
Linux socket implementation. Source presence is not proof that HTTP serving,
deadlines, shutdown or concurrent rendering work. Verify those at runtime in
the selected build before making a service compatibility claim.

## Hardware boundary

`experiments/06-pico2w-epaper-2in13-v4/SPEC.md` explicitly describes
Pico-ePaper-2.13 V4 with Pico SPI1, not the Raspberry Pi 40-pin HAT.
Its pin assignments and electrical instructions must not be reused here.
The earlier 7.5-inch Driver HAT mapping is also not authority for the new HAT.

No new connection table is approved. GPIO idle state alone does not prove
that Pironman has no electrical connection to the pin. Resolve the exact
new HAT and Pironman pin ownership before giving physical instructions.

### Alternative SPI investigation

Target update, 2026-09-08: user reports `uname -r` as
`6.18.39+rpt-rpi-2712`. This identifies the installed kernel release, not its
complete source commit or active Device Tree/overlays.

Rechecked official `rpi-6.18.y` sources:

- [SPI5 overlay](https://github.com/raspberrypi/linux/blob/rpi-6.18.y/arch/arm/boot/dts/overlays/spi5-1cs-pi5-overlay.dts)
  still defaults CS to GPIO12 and exposes only `cs0_pin`/`cs0_spidev`.
- [RP1 pinctrl](https://github.com/raspberrypi/linux/blob/rpi-6.18.y/arch/arm64/boot/dts/broadcom/rp1.dtsi)
  still groups GPIO13/14/15 for SPI5. Moving CS alone does not remove GPIO13.
- Therefore the stock overlay remains unsuitable for the conservative
  Pironman-preserving plan. A TX-only pinctrl plus remapped CS remains a
  candidate, not a validated deployment or wiring instruction.

`git ls-remote` returned branch HEAD
`0ac97ba3443f519b61bbc96079736cd8b881ea22`; immutable raw-file requests missed
the web cache, so the inspected branch-page contents are not claimed as
verified at that exact SHA or identical to the installed kernel package.
Next read-only target check: `ls -l /dev/spidev* /dev/gpiochip*` for existing
nodes and access groups. Node existence alone does not prove line ownership.

Verified against Raspberry Pi kernel branch `rpi-6.12.y`, not the running
homelab kernel (historical investigation before the version was supplied):

- SPI3's RP1 pin group is GPIO5/6/7; GPIO6 overlaps Pironman fan control.
  Its stock overlay is not a conflict-free replacement.
- SPI5's RP1 pin group is GPIO13/14/15, with stock CE0 on GPIO12.
  GPIO13 overlaps Pironman IR. GPIO12 is also listed as an optional RGB
  connection in SunFounder's table. Do not enable the stock overlay as-is.
- The Pi 5 SPI5 overlay accepts `cs0_pin`, but offers no `no_miso` parameter.
  Merely moving chip select does not stop the default pinctrl group from
  claiming GPIO13.
- Candidate for further validation: a dedicated transmit-only SPI5 pinctrl
  group excluding MISO, with chip select outside Pironman's documented pins.
  This requires a custom overlay and validation against the actual Pi kernel;
  it is not a tested configuration or an approved wiring instruction.
- The SunFounder page still says RGB is connected to GPIO10 in its detailed
  description, but lists optional GPIO12/21 in its final table. Avoiding all
  listed pins is a conservative design direction, not a claim that all three
  are simultaneously wired to RGB on this unit.

Sources inspected:

- [RP1 pin groups](https://github.com/raspberrypi/linux/blob/rpi-6.12.y/arch/arm64/boot/dts/broadcom/rp1.dtsi),
  `rp1_spi3_gpio5` and `rp1_spi5_gpio13`.
- [Pi 5 SPI5 overlay](https://github.com/raspberrypi/linux/blob/rpi-6.12.y/arch/arm/boot/dts/overlays/spi5-1cs-pi5-overlay.dts).
- [Pi 5 SPI3 overlay](https://github.com/raspberrypi/linux/blob/rpi-6.12.y/arch/arm/boot/dts/overlays/spi3-1cs-pi5-overlay.dts).
- [Pironman IO Expander](https://docs.sunfounder.com/projects/pironman5/en/latest/pironman5/hardware/io_board.html),
  RGB Control Pin, Infrared Receiver, RGB Fan Pins and Pin Headers sections.

## Remaining work and acceptance

1. Model/revision confirmed from the photo; finish inspecting its schematic,
   V4 driver, BUSY polarity, reset timing, refresh and sleep requirements.
2. Resolve Pironman GPIO ownership against its schematic/source and the
   running Go service; select and document the Pi 5-specific SPI overlay.
3. Use ordinary Go for Linux service integration while retaining a TinyGo
   build gate for shared renderer/panel modules. Keep OS GPIO/SPI and service
   lifecycle in Linux-specific adapters; preserve manager security boundaries.
4. Implement a separate service with bounded authenticated input, serialized
   display updates, safe shutdown, diagnostic stages and panel recovery.
5. Add deterministic driver/transport tests, malformed input and concurrent
   submission tests, and Docker-to-service runtime coverage.
6. Supply Debian systemd and device-permission configuration, then verify on
   the actual Pi 5. Compilation and SPI success do not prove visible pixels.

## Primary references

- [TinyGo Linux executables](https://tinygo.org/docs/guides/linux/).
- [TinyGo standard-library support](https://tinygo.org/docs/reference/lang-support/stdlib/).
- [Waveshare 2.13-inch HAT manual](https://www.waveshare.com/wiki/2.13inch_e-Paper_HAT_Manual).
  Search indexing was accessible; full-page fetch returned HTTP 403. Do not
  represent that as a completed schematic review.
- [Waveshare monochrome V4 specification](https://files.waveshare.com/upload/4/4e/2.13inch_e-Paper_V4_Specification.pdf).
  Matched to the product marking; full electrical review remains pending.

## Proposed capability map

These are implementation boundaries, not implemented capabilities. The user
approved this map with "Так" on 2026-09-08. Module specifications below remain
proposed until reviewed. Build order follows the dependency direction below.

| Module | Responsibility | Depends on |
| --- | --- | --- |
| panel-v4 | Portable command lifecycle, geometry, BUSY deadlines and errors | Narrow injected I/O contract |
| linux-panel-io | Linux SPI and GPIO ownership; no direct register mmap | panel-v4 I/O contract |
| local-display-service | Bounded authenticated HTML/image ingress, rendering, serialized updates | Existing engine, panel-v4, linux-panel-io |
| debian-deployment | systemd, least-privilege device access, Docker client instructions | local-display-service |

Module specifications:
[panel-v4](pi5-service/SPEC-panel-v4.md),
[linux-panel-io](pi5-service/SPEC-linux-panel-io.md),
[local-display-service](pi5-service/SPEC-local-display-service.md),
[debian-deployment](pi5-service/SPEC-debian-deployment.md).

Preserve existing Pico entry points and V2 7.5-inch driver. Reuse rendering
and common error/transport practices only where compatible. A successful
HTTP request must distinguish admission, controller completion and visible
acceptance. Do not claim optical confirmation from a software response.

### Official V4 source evidence

Inspected the upstream monochrome driver (not executed):
https://github.com/waveshareteam/e-Paper/blob/a794fbc39656b0f93938d1ffb3fdc77eaed9e9fc/RaspberryPi_JetsonNano/python/lib/waveshare_epd/epd2in13_V4.py

Upstream HEAD `a794fbc39656b0f93938d1ffb3fdc77eaed9e9fc` verified with
`git ls-remote` on 2026-09-08. Use this immutable revision for the initial port.
The initial web inspection used `master` and the pinned raw URL missed cache.
Resolved on 2026-09-08: fetched this exact commit through Git in the existing
Dev Container and read both C and Python V4 implementations using `git show`.
The port uses C's ordinary full-refresh path, including 10 ms post-BUSY and
100 ms post-sleep delays. Python omits the first and uses 2000 ms plus module
cleanup for the latter. No fast/partial paths were ported. The upstream
permission notice is retained beside `panelv4` code.

- Native geometry: 122 by 250, row stride rounds up to 16 bytes (4000 bytes).
- BUSY is active HIGH, opposite to the preserved 7.5-inch driver's idle rule.
- Ordinary initialization and full refresh are distinct from fast/partial
  paths. Start with ordinary full refresh and deep sleep.
- Cross-check the datasheet before porting. The
  upstream unbounded BUSY loop must become a bounded, stage-labelled wait,
  not be copied as an infinite service hang.

Photo resolves identification only. Actual Pi kernel/overlays, physical
wiring, supply and first visible output still require validation.

### Datasheet cross-check

The official V4 specification, revision 4.0 dated 2023-03-17, confirms:

- Page 5: native 122(H) by 250(V), 1-bit black/white display.
- Page 8: BUSY HIGH prohibits interrupting the operation or sending commands;
  reset and chip select are active LOW. BS1 LOW selects 4-wire, 8-bit SPI.
- Page 10: SPI write data are sampled on rising edges, most-significant bit
  first; D/C must stay unchanged through each byte.
- Page 9: a full refresh is recommended after five consecutive fast/partial
  operations. Initial implementation remains ordinary full refresh only.
- Page 8 absolute maximum VCI is not a recommended supply voltage. These are
  bare-panel limits, not proof of the HAT connector's acceptable supply.

The parsed PDF omits some diagram/table contents. Electrical values and exact
HAT supply/orientation must be checked in the rendered tables and board
schematic before publishing a wiring instruction.

Board schematic located on the official file page:
https://www.waveshare.com/wiki/File:2.13inch_e-Paper_HAT_Schematic_diagram.pdf
with PDF at
https://files.waveshare.com/upload/4/44/2.13inch_e-Paper_HAT_Schematic_diagram.pdf.
Text extraction is available, but the screenshot tool did not expose a readable
diagram in this run. Net connectivity, revision match and supply recommendations
are therefore NOT marked verified from that extraction alone.

P4 library review completed: `github.com/warthog618/go-gpiocdev` v0.9.1,
MIT, commit `9d5d427a6821c365770b004f23d45df3dd12105b`, added to the Go module.
Uses GPIO character-device ABI v2, not sysfs GPIO or register mmap.
SPI uses a small standard spidev adapter: the inspected periph host v3.8.5
wrapper does not expose the positive ioctl transferred-byte count to callers;
our `os.File.Write` checks the actual count and stops on any short write.
No periph dependency was added. See `pi5-service/SPEC-linux-panel-io.md`.
