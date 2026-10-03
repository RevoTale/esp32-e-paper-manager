# Capability Map: Remote HTML e-paper v2

Current map, 2026-09-06: use the component table in
`decisions/013-pure-go-render-engine.md` and the contract `SPEC-engine.md`.
The map below is preserved historical context; it does not restore Blitz,
stylesheets or a Pico framebuffer into the accepted streaming architecture.

Latest clarification (2026-09-05): native Pico IPv6 is deferred and not a
first-release gate. Server grouping settings are optional. Follow
`docs/render-design-review-2026-09-05.md` for current batching, timestamp and
network scope; older dual-stack acceptance wording below is historical.

Current checkpoint (2026-09-06): `tasks/plan.md` governs the implemented manager
and EPS1 streaming increments. The isolated `refreshstamp` component and real
Blitz PNG preview now pass; live timestamp metadata/layout integration remains
open. See `docs/refresh-timestamp.md`. The implementation status below is
historical, not a claim that those newer increments are absent.

Historical baseline: the original on-device HTML path was implemented in software on
2026-09-02. ADR-008's manager-side render diff and Pico display-list path are
accepted but not implemented. Physical Wi-Fi, energy, partial-panel, and
target-IPv6 acceptance remain explicit gates.

2026-09-05 increment: `renderdiff` implements the isolated bounded final-frame
damage reference with reconstruction/fuzz tests. The new renderer, screen
codec and runtime integration are still pending. Follow
`docs/render-conflicts-and-module-boundaries.md` for conflict rules and module
ownership; do not interpret planner coverage as CSS or hardware acceptance.

The approved v1 map in `CAPABILITY-MAP.md` remains historical and valid for the
existing frame-transfer implementation. This map records the implemented v2
modules and the accepted ADR-008 direction without claiming that the new path
already exists.

Architecture amendment, 2026-09-03: ADR-011 restores normal HTML and a tested
CSS subset as the general authoring input. ADR-012 selects pinned Blitz
as the isolated manager candidate, not yet a production-accepted dependency.
The approved 2026-09-06 beta.2 evaluation still has two image-layout bugs.
The user chose to enable bounded PNG handoff and skip only their reproductions;
real Go → BZR2 → Blitz pixels are verified locally, not on the panel. See
`docs/manager-image-assets.md` and the exact root test exception. Pico receives bounded layout-resolved
display-list updates and retains
rasterization, framebuffer diff, and refresh safety. Existing on-device HTML
modules remain a direct-USB compatibility/recovery path until that contract is
finalized.

| Module id | Responsibility | Depends on |
|---|---|---|
| `display-surface` | Device-independent monochrome dimensions, pixel writes, refresh capabilities, and lifecycle contract | — |
| `panel-waveshare-7in5` | Verified Waveshare/HAT power, SPI, BUSY, full refresh, sleep, and optional separately accepted partial refresh | `display-surface` |
| `document-model` | Bounded, typed document representation independent of transport and panel | — |
| `html-css-profile` | Versioned supported HTML elements, CSS properties/selectors, asset rules, viewport behavior, and typed diagnostics | `document-model` |
| `layout-engine` | Display-size-aware box layout, wrapping, content priority, and typed overflow outcomes | `document-model`, `display-surface` |
| `mono-rasterizer` | Existing direct-USB text/primitives/assets and reserved timestamp overlay into one 1-bit framebuffer | `layout-engine`, `display-surface` |
| `render-pipeline` | Existing direct-USB parse-to-frame recovery path, resource budgets, content hashing, and diagnostics | `html-css-profile`, `layout-engine`, `mono-rasterizer` |
| `manager-renderer` | Provider-owned HTML/CSS parsing, viewport layout, and deterministic output; pinned Blitz is the spike candidate | `html-css-profile`, `display-list` |
| `render-diff` | Manager-owned stable-ID reconciliation, old/new invalidation, and bounded layout-resolved display-list patches | `manager-renderer`, `update-contract` |
| `manager-assets` | Bounded image acquisition, validation, decode, sizing, resampling, alpha composition, monochrome conversion, content identity, and SSRF-safe optional fetching | `manager-renderer` |
| `manager-raster-fallback` | Render supported static HTML/CSS into deterministic clipped one-bit bitmap objects when object operations are not viable | `manager-renderer`, `manager-assets` |
| `refreshstamp` | Isolated confirmed full-cycle label state and protected-corner composition; no clock source, scheduler or live transport integration | `display-surface`, shared bitmap font |
| `display-list` | Transport-independent complete/patch drawing operations, geometry, clipping, assets, revision negotiation, acknowledgements, and resynchronization | `display-surface` |
| `device-rasterizer` | Pico-side validation and rasterization of resolved display-list operations into the monochrome framebuffer | `display-list`, `display-surface` |
| `credential-store` | USB-only provisioning, integrity-checked persistent secrets, rotation, recovery, and factory reset | — |
| `update-contract` | Versioned update request/result/error/idempotency contract shared by USB, manager, and device link | `render-pipeline` |
| `usb-update` | Preferred USB transport plus physical provisioning/control channel | `credential-store`, `update-contract` |
| `manager-api` | Home-server HTTPS/VPN API, user authentication, bounded pending HTML update, manager rendering/diff, device status, and retention policy | `render-diff`, `update-contract` |
| `device-link` | Pico-initiated mutually authenticated and encrypted IPv4/IPv6 session carrying the resolved display-list protocol | `credential-store`, `manager-api`, `display-list` |
| `device-runtime` | USB priority, device-link arbitration, queue/coalescing, clock policy, refresh scheduling, energy policy, and diagnostics | `panel-waveshare-7in5`, `usb-update`, `device-link`, `device-rasterizer`, `render-pipeline` |
| `host-tools` | USB provisioning, HTTPS sending, compatibility checks, golden fixtures, and operator diagnostics | `usb-update`, `manager-api`, `update-contract` |
| `testkit` | Reusable fake display, clock, storage, streams, network adapters, and physical acceptance harness | all provider contracts |

Historical implemented build order:

1. Approve security topology, provisioning semantics, HTML profile input, time
   source, resource budgets, and quality constraints.
2. Freeze contracts for `display-surface`, `document-model`, `update-contract`,
   `credential-store`, and `testkit`.
3. Build `html-profile` and `css-profile` with bounded parser tests and measured
   target resource probes.
4. Build `layout-engine`, `mono-rasterizer`, and `render-pipeline`; accept golden
   output and overflow behavior on multiple display sizes.
5. Adapt the known-good `panel-waveshare-7in5` path without changing its
   physical sequence; prove the first USB HTML-to-panel slice.
6. Add USB provisioning and recovery, then the authenticated encrypted IPv4
   `device-link` and home `manager-api`.
7. Add WPA3 policy and HTTPS/VPN gateway integration; physically accept Wi-Fi
   and energy behavior independently. Target IPv6 remains blocked by the
   embedded stack's incomplete SLAAC/router lifecycle.
8. Run complete review/fix/simplify and physical-acceptance loops without
   weakening the approved constraints.

ADR-008 next work:

Research basis: `docs/resolved-render-protocol-research.md`.

1. Freeze `display-list` capabilities, complete/patch operations, bounds,
   revision/acknowledgement, resynchronization, and error contracts.
2. Implement and golden-test `manager-renderer` plus `render-diff` without
   changing the accepted direct-USB path.
3. Implement and resource-test the bounded `device-rasterizer`.
4. Carry the same display-list protocol over USB and the encrypted device link,
   then run reconnect, full-resync, panel-safety, and physical acceptance.

Dependency rule: rendering depends on `display-surface`, never on a concrete
panel. Transports depend on `update-contract`, never on renderer internals. The
panel driver never parses HTML, credentials, HTTP, or USB. The public API never
exposes or receives the Pico device key.
