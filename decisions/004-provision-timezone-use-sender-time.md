# ADR-004: Provision timezone and use authenticated sender time

## Status

Accepted.

## Date

2026-09-02.

## Context

The display must show when its last successful update was received. Pico 2 W
has no battery-backed real-time clock, and embedding a complete timezone
database would consume flash and require independent rule updates. The timezone
must remain a per-device setting chosen during flashing.

## Decision

The trusted USB host or home manager supplies UTC update time inside the same
authenticated record as the document. Failed authentication, validation,
rendering, or panel refresh does not advance the displayed timestamp. Pico does
not contact NTP independently.

The flash workflow is `flash generic UF2 -> provision over USB`. During that
workflow, the host reads the IANA timezone from `EPAPER_TIMEZONE`, defaulting to
`Europe/Kyiv`, and stores it in the versioned device configuration. The setting
is non-secret and can later be changed through USB without rebuilding firmware.

Timezone and daylight-saving conversion run on the trusted host or manager.
Pico receives bounded display-time metadata authenticated together with the UTC
time and document. The timezone identifier must match the provisioned setting.

Render the timestamp in a reserved bottom-right rectangle. It must not span the
full display width and must not overlay submitted HTML. Determine its exact
dimensions from timestamp font metrics, padding, format, framebuffer tests, and
physical legibility rather than an assumed pixel constant. Document layout must
treat the remaining non-rectangular area as its hard paint boundary.

## Alternatives considered

### Compile timezone into each UF2

Rejected. It would make a non-secret device setting require a firmware rebuild
and would break the generic reproducible UF2 decision.

### Run NTP and timezone conversion on Pico

Rejected for the initial version. It adds another network protocol, clock trust
path, timezone data, flash use, and update responsibility without improving the
authenticated update timestamp.

### Use a fixed UTC display

Rejected. The requested default is local `Europe/Kyiv` time and the timezone
must be configurable per device.

## Consequences

- Timestamp correctness depends on the authenticated host or manager clock.
- Firmware avoids NTP traffic and a timezone database, reducing code size,
  radio use, and maintenance.
- Host, manager, USB, and Wi-Fi paths must produce identical timestamp metadata.
- The renderer and overflow diagnostics must account for the bottom-right
  cutout.
- Tests must cover daylight-saving boundaries, invalid timezone identifiers,
  stale or malformed timestamps, and failure paths that must not advance time.
