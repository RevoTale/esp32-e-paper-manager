# Implementation plan: Pi 5 local e-paper service

Status: software implementation and checks completed on 2026-09-08;
P1/P4 electrical/real-DTB gates and P8 target acceptance remain pending.
Architecture, Unix-socket transport and implementation order accepted.
Task list: [todo.md](todo.md). Module specs are in the parent directory.

## Scope

Extend the existing Go renderer with an isolated 2.13-inch V4 panel driver,
Linux I/O adapter and Debian service. Docker submits HTML/PNG over an
authenticated Unix socket. Preserve all Pico entry points and Pironman
RGB/OLED/fan functionality. No deployment or physical acceptance is implied
by a successful build.

## Order

1. P1: pin and verify the exact HAT and upstream sources.
2. P2: portable full-refresh path with a deterministic bus transcript.
3. P3: portable fault handling and TinyGo build proof.
4. P4: Linux I/O ownership and a non-conflicting overlay.
5. P5: real HTML/PNG to native frame through a fake display.
6. P6: authenticated Unix-socket HTTP admission and status.
7. P7: ordinary Go executable, systemd lifecycle and Docker configuration.
8. P8: regression gates and actual Pi/Docker/display acceptance.

P4 and P5 both consume P2/P3 contracts; complete them sequentially unless
parallel work is explicitly authorized. P6 consumes P5; P7 combines P4/P6.

## Decisions and risks

- Keep native 122x250 packing distinct from logical 250x122 layout. Test
  rotation and odd-width padding, not merely total byte count.
- No custom kernel display driver. Use Linux SPI/GPIO character-device APIs.
- Proposed SPI5 TX-only overlay must not claim IR GPIO13 or default CS GPIO12.
  Kernel pinctrl and board schematic are authoritative; idle pin state alone
  does not establish physical freedom. Final wiring follows verified overlay.
- Do not silently turn a valid BUSY timeout into success or blind retries.
- Reject input before I/O, own one complete update, sleep when safe, retain
  stage errors. A failed HTTP response after activation is an ambiguous client
  result; provide status rather than automatically displaying again.
- Keep source-free warnings and no secret/request-body logging.
- Ordinary Go for Linux; TinyGo compile probes for portable controller and
  existing renderer. No security downgrade to avoid `os.Root` incompatibility.

## Quality and completion

Use existing `CONSTRAINTS.md`: at most 300 lines/file, 60 lines/function,
complexity 10, changed coverage at least 90%, no baseline coverage decrease.
Run focused tests per task, then existing quality gates without suppressions.
Hardware acceptance must include HTML and PNG submitted from Docker, visibly
different results, a sleep/wake cycle, restart safety and intact Pironman.

Human boundaries: authorize actual target configuration,
service install/reboot and hardware checks when artifacts are ready. Never
create/rebuild containers, commit, push or mutate homelab as a side effect.
