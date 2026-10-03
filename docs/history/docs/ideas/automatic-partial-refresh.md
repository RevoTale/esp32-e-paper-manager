# Automatic partial refresh for HTML updates

> Architecture update, 2026-09-02: ADR-008 supersedes the complete-HTML
> manager-to-Pico flow below with server-side HTML adaptation, layout, and
> resolved render-object diffs. The device-local refresh policy, framebuffer
> diff, dirty-region safety limits, and physical acceptance gates in this
> proposal still apply. See
> `../../experiments/03-remote-epaper/decisions/008-server-resolved-render-diff.md`.

## Problem Statement

How might we accept complete HTML updates at any server-selected cadence while
refreshing only changed pixels when safe, without allowing remote configuration
to damage the Waveshare 7.5-inch V2 panel?

## Earlier Recommended Direction

This section preserves the earlier full-HTML proposal for history. ADR-008 now
makes the Go manager responsible for HTML/CSS adaptation, layout, and
render-object diff. Pico rasterizes validated resolved operations into its
framebuffer, compares changed pixels, and lets a device-local policy select
`unchanged`, partial, fast full, or normal full refresh. The manager cannot
force an unsafe partial refresh.

Replace the single `MinimumRefreshInterval` with a typed, mode-aware policy.
The agreed initial profile is:

```yaml
mode: auto
partial_min_interval: 1s
full_min_interval: 30s
max_partials_per_full: 10
max_partial_area_percent: 35
max_partial_regions: 4
```

Both intervals are earliest permitted starts, not periodic timers. Unchanged
content performs no physical refresh. Requests received too early are
coalesced by the manager into the newest target display-list revision and
report why the update is queued and when it becomes eligible.

The firmware owns hard safety bounds. An authenticated manager may tune values
inside those bounds, but cannot lower the one-second partial floor, lower the
30-second full floor, disable full cleanup, or exceed the accepted partial
chain. USB provisioning and firmware recovery retain priority.

The first frame after boot, lost display history, driver error, timeout, or
policy change is always full. A large or highly fragmented bitmap diff is full.
Otherwise, changed tiles are clustered into at most four separate partial
rectangles. Horizontal bounds expand outward to whole 8-pixel groups, so old
and new glyph pixels are both refreshed without clipping.

The timestamp rectangle must remain independent from distant content changes.
Using one bounding rectangle for both could cover almost the entire panel and
incorrectly select full refresh. Partial eligibility therefore uses the sum of
the clustered rectangle areas, not the area of one screen-spanning union. The
driver should load all accepted windows and trigger one physical refresh; if
the exact controller cannot do that reliably, fall back to full rather than
perform several independent panel refresh cycles.

## Evidence and Constraints

- Waveshare currently specifies about 0.4 seconds for partial refresh, 1.5
  seconds for fast refresh, and 4 seconds for full refresh:
  https://www.waveshare.com/product/ai/displays/e-paper/7.5inch-e-paper-hat.htm
- The current official driver provides `EPD_7IN5_V2_Init_Part` and
  `EPD_7IN5_V2_Display_Part`:
  https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/c/lib/e-Paper/EPD_7in5_V2.c
- The official example performs ten partial updates with 500 ms delays before
  returning to full initialization and cleanup:
  https://github.com/waveshareteam/e-Paper/blob/master/RaspberryPi_JetsonNano/c/examples/EPD_7in5_V2_test.c
- Waveshare warns that repeated partial refresh requires periodic full refresh
  and that the panel must not remain powered indefinitely:
  https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT_Manual
- Partial-window garbage has been reported for some X positions. Partial mode
  remains experimental until the exact panel passes aligned-window tests:
  https://github.com/waveshareteam/e-Paper/issues/362

## Key Assumptions to Validate

- [ ] The exact purchased panel reliably refreshes aligned partial windows at
  several X positions and sizes, including multiple windows committed by one
  physical refresh.
- [ ] One-second spacing does not produce unacceptable ghosting during a
  bounded ten-update sequence followed by full cleanup.
- [ ] Retaining an exact previous 48,000-byte frame fits measured Pico 2 W RAM,
  stack, Wi-Fi, and render budgets; otherwise evaluate bounded tile digests.
- [ ] The controller can safely sleep or power-cycle between partial requests;
  otherwise use a strictly bounded partial session and measure its current.
- [ ] The initial 35% area threshold is faster and cleaner than full refresh on
  the real panel; tune it from measurements rather than assumption.

## MVP Scope

1. Add a transport-independent refresh policy and explicit queued diagnostics.
2. Add exact bitmap diff, dirty-tile clustering, and calculation of at most four
   aligned partial rectangles.
3. Port the official partial initialization/window sequence behind an
   experimental capability.
4. Physically test multiple X/Y positions, distant simultaneous regions,
   changing text widths, clearing old glyphs, ten one-second partials, full
   cleanup, reboot, and error recovery.
5. Measure RAM, SPI time, physical refresh time, current, and visible ghosting.
6. Enable `RefreshPartial` in production capabilities only after those checks
   pass; otherwise retain full refresh and evaluate fast full separately.

## Not Doing (and Why)

- **Incremental DOM or HTML fragments on Pico** -- ADR-008 uses manager-owned
  render-tree reconciliation instead. Pico receives bounded layout-resolved
  operations, not mutable HTML fragments or a browser DOM.
- **Client-forced partial refresh** -- the client cannot know panel history or
  safely override device policy.
- **Refreshing on unchanged content** -- wastes energy and panel cycles.
- **Unbounded rapid mode** -- a compromised or broken server must not disable
  local longevity controls.
- **Silent fallback after a driver error** -- the next update becomes full and
  reports the reason.

## Open Technical Questions

- Exact previous framebuffer versus tile digests after resource measurement.
- Sleep/power-cycle per partial versus a bounded powered partial session.
- Whether fast full is a useful fallback for large diffs on this exact panel.
- Final partial-area threshold and maximum region count after SPI and waveform
  timing measurements.
