# Pi 5 / Pironman 5 overlay candidate

Status: compiled and tested against a minimal merge fixture. **Not installed
or approved for physical wiring.** The implemented Linux service and operator
instructions are in [SERVICE.md](SERVICE.md); this page covers its overlay gate.

## Scope

`epaper-spi5-pi5-overlay.dts` adds an SPI5 userspace device, replaces its entire
pinctrl list with GPIO14/15, and gives the kernel active-low CS on GPIO16.
It does not configure DC/RST/BUSY, power or the display lifecycle. No GPIO13
MISO connection is requested. The adapter must use write-only transactions.

The stock SPI5 overlay includes GPIO13 and defaults CS to GPIO12. Both are
excluded here because SunFounder documents IR on GPIO13 and optional RGB on
GPIO12. SPI0 and SPI10 remain untouched. Pi 5's base DTS identifies SPI10 as
the bootloader EEPROM bus; never probe it with display commands.

Sources reviewed from Raspberry Pi's `rpi-6.18.y` branch:

- [SPI5 stock overlay](https://github.com/raspberrypi/linux/blob/rpi-6.18.y/arch/arm/boot/dts/overlays/spi5-1cs-pi5-overlay.dts)
- [RP1 pin groups](https://github.com/raspberrypi/linux/blob/rpi-6.18.y/arch/arm64/boot/dts/broadcom/rp1.dtsi)
- [SPI aliases and default pinctrl](https://github.com/raspberrypi/linux/blob/rpi-6.18.y/arch/arm64/boot/dts/broadcom/bcm2712-rpi.dtsi)
- [Pi 5 base and SPI10 purpose](https://github.com/raspberrypi/linux/blob/rpi-6.18.y/arch/arm64/boot/dts/broadcom/bcm2712-rpi-5-b.dts)
- [Pironman wiring](https://docs.sunfounder.com/projects/pironman5/en/latest/pironman5/hardware/io_board.html)

These branch sources are not proof of the installed package's exact contents.
The user reports kernel `6.18.39+rpt-rpi-2712`. Installed base DTB and active
overlays still require review. UART0/other pin consumers on GPIO14/15/16 must
not be active. Do not disable another service to force a successful probe.

## Offline verification

Run inside the existing Dev Container, from the remote-epaper module:

```sh
go test -v ./deploy/pi5
```

`device-tree-compiler` is persisted in `.devcontainer/Dockerfile`. Installed
in the current container on 2026-09-08: Debian package 1.7.2-2+b1, DTC 1.7.2.
No container rebuild was performed or verified.

The test compiles DTS to DTBO, applies it with `fdtoverlay` to a deliberately
minimal fixture, then reads resolved properties with `fdtget`. It verifies
the old GPIO13 group is no longer referenced by SPI5, CS resolves to GPIO16,
SPI0 CS stays GPIO8/7 and SPI10 remains enabled. The fixture is not a Pi DTB
and must never be installed. Missing tools fail the test, not skip it.

The initial SPI frequency cap is 1 MHz, not a measured safe maximum. Mode 0,
MSB-first, speed and native SPI transfer ownership must be enforced by the
Linux adapter. GPIO16 is not requested through gpiochip.

## Before installation

1. Obtain the user's installed `bcm2712-rpi-5-b.dtb` and validate offline merge
   against it, preserving original bytes. Inspect the active overlay set too.
2. Validate HAT schematic/supply, signal mapping and current pin ownership.
3. Review the implemented Linux service, access policy and fault-test results.
4. Obtain explicit approval for boot configuration, reboot and wiring.
5. Verify kernel probe/device creation and intact Pironman features before any
   display command. Require optical acceptance separately.

Do not stack this HAT onto the entire Pironman header or enable stock SPI5 as
a shortcut. Physical installation remains a separate acceptance boundary.

### Deferred target-only preflight

The user explicitly chose to finish software first. Do not request a DTB copy
again just to continue implementation. When deploying, run read-only inspection
and an offline merge **on the Pi**, never overwrite its base DTB:

```sh
fdtget /boot/firmware/bcm2712-rpi-5-b.dtb /__symbols__ spi5
fdtget /boot/firmware/bcm2712-rpi-5-b.dtb /__symbols__ gpio
fdtoverlay -i /boot/firmware/bcm2712-rpi-5-b.dtb -o /tmp/epaper-pi5-merged.dtb build/epaper-spi5-pi5.dtbo
pinctrl get 14-16
pinctrl get 22-24
```

Review the merged SPI5 pinctrl references and current boot overlay configuration
before copying the custom DTBO to `/boot/firmware/overlays/` or adding
`dtoverlay=epaper-spi5-pi5`. Do not install the merged base DTB or the test fixture.
These commands do not establish supply voltage, connector orientation or
electrical compatibility. The final eight-wire table still needs that review.
