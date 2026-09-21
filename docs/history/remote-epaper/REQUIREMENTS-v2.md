# Remote HTML e-paper firmware requirements (v2)

Current authority, 2026-09-06: ADR-013 and `SPEC-engine.md` supersede the Blitz,
stylesheet and Pico-rendering assumptions below. HTML + inline style is rendered
by the Go manager; Pico receives bounded resolved pixels. Execute the active
`tasks/pure-go-engine.md` list. Security and quality constraints remain in force.

Status: accepted for incremental implementation and verification.
Recorded: 2026-09-02.
Target: Raspberry Pi Pico 2 W, TinyGo, and the verified Waveshare 7.5-inch
800x480 black/white setup through e-Paper Driver HAT Rev2.3.

This document records the requested product direction without erasing the
approved v1 frame-transfer design. A requirement marked **Research gate** is
not an implemented or hardware-verified capability.

Architecture amendment, 2026-09-03: ADR-008 and ADR-011 supersede the
manager-to-Pico parts that require HTML/CSS resolution or layout on Pico. The
Go manager owns bounded HTML/CSS validation, layout, render-tree state, and
object diff; Pico receives bounded layout-resolved render operations and keeps
the final framebuffer diff and panel-refresh decision. Direct HTML-over-USB is
retained as a compatibility/recovery capability pending its final contract.
All security, USB priority, display lifecycle, energy, diagnostics, resource,
quality, and physical-acceptance requirements remain in force. See
`decisions/008-server-resolved-render-diff.md`.

## Objective

Latest clarification (2026-09-05):
`docs/render-design-review-2026-09-05.md` takes precedence for server-owned
batching with optional `debounce_interval` and `max_wait`, the configurable
600-second full-refresh timestamp policy, and deferral of native Pico IPv6.
IPv4 is sufficient for the first firmware release through the central manager.

Build a modular Go manager and TinyGo firmware. The manager accepts bounded
HTML/CSS, resolves it into a versioned display list or bitmap patch, and sends complete or
incremental render operations over the authenticated device link. Pico
validates and rasterizes those operations into a monochrome display surface and
updates the attached e-paper safely. Direct USB remains the preferred control,
provisioning, recovery, and compatibility path. The display-specific driver
must be replaceable without rewriting manager rendering, authentication,
transport, or display-list code.

Success means a user can provision the device through physical USB, submit a
supported document to the authenticated manager API or USB compatibility tool,
receive a precise result or error, and see identical resolved output plus a
small last-successful-update timestamp. Success also requires measured memory,
flash, latency, radio, and refresh-energy evidence on the physical Pico 2 W.

## Product requirements

### Inputs and priority

- Accept resolved display-list updates through USB and Wi-Fi. The manager API
  accepts bounded HTML/CSS; direct HTML over USB remains a compatibility path.
  USB has priority over Wi-Fi.
- A USB update may cancel an incomplete Wi-Fi upload, but no transport may
  interrupt a physical panel refresh already in progress.
- Both transports feed the same bounded display-list validation, rasterization,
  and refresh pipeline. When USB accepts HTML, the host compatibility tool must
  use the same manager renderer contract and produce the same display list.
- Reject input before display mutation when authorization, framing, length,
  encoding, parsing, layout, or resource validation fails.
- Duplicate content must not trigger a physical refresh unless the reserved
  timestamp overlay must change and policy explicitly permits that refresh.

### On-device renderer

- The manager-facing path uses a versioned resolved-render protocol. Pico does
  not parse manager-supplied HTML, resolve CSS, wrap text, or calculate layout.
- The manager sends a complete resolved display list initially and later sends
  only changed/removed objects plus all dirty regions affected by their old and
  new bounds.
- Pico validates and rasterizes bounded drawing operations, calculates the
  resulting framebuffer difference, and retains final authority over physical
  refresh mode and timing.
- Stable IDs optimize manager reconciliation; they are not assumed to bound
  visual damage. Layout changes must include all affected objects and regions.
- The manager profile must support the HTML/CSS and image contract in
  `docs/manager-html-css-profile-v2.md`. Images are decoded, sized, clipped,
  composited, and converted to one-bit pixels on the manager; Pico receives a
  bounded `BLIT_1BPP_RAW` operation, never PNG, JPEG, SVG, or browser input.
- Authors may explicitly mark a bounded custom static raster subtree.
  Unsupported HTML/CSS outside the profile is a typed error, never
  a silent approximation. JavaScript and active content remain forbidden.
- The direct HTML-over-USB renderer below remains a compatibility/recovery path
  until its final role is approved. It must not silently produce different
  visible semantics from manager-generated render operations.

- The manager and direct USB compatibility tool expose a documented HTML/CSS
  profile. They are not browsers and do not promise general web compatibility.
- JavaScript, external stylesheets, external fonts, remote images, navigation,
  forms, animation, video, audio, SVG scripting, and dynamic DOM mutation are
  out of scope.
- Supported elements, CSS properties, units, inheritance, defaults, limits,
  and fallback behavior must be documented as a public contract and covered by
  golden tests.
- The initial profile should prefer static dashboard primitives: text, headings,
  paragraphs, simple containers, rows/columns, borders, spacing, alignment,
  monochrome colors, and explicitly bounded embedded bitmap assets.
- **Research gate:** choose the smallest subset that can represent the intended
  Glance-like dashboard natively while server raster fallback covers explicit
  custom content. Manager CPU/RAM and wire-byte limits apply to the expanded
  profile; Pico budgets apply only to the bounded display-list decoder,
  rasterizer, framebuffer, and incoming bitmap region.
- Unsupported syntax must have deterministic behavior: either a stable fallback
  or a typed rejection. It must never be silently interpreted as a different
  security-sensitive feature.

### Representative Glance input

- The user-supplied Glance capture is research input only. Its raw 373,166-byte
  form is not stored because it contains private incident and task names,
  scripts, dynamic attributes, and external references.
- The repository stores only `testdata/glance-dashboard-sanitized.html`, a
  manually reduced fixture with invented service and task names. It must not
  contain copied URLs, identifiers, tokens, personal text, or executable code.
- The first profile must express the sample's useful static semantics: page
  heading, two-column card layout, card heading and count, status rows, grouped
  task lists, labels, relative-time text, borders, spacing, and text hierarchy.
- Navigation, buttons, inputs, links, popovers, scripts, dynamic loading state,
  event behavior, and external stylesheets are not rendered. Remote resources
  are not fetched implicitly; `<img>` follows the bounded manager asset policy
  and remains off for arbitrary URLs by default. The adapter removes other
  unsupported behavior or converts its visible state into supported static
  output before anything reaches Pico.
- SVG icons are not accepted from input. The adapter may omit them or map an
  allowlisted semantic icon name to a firmware-owned monochrome glyph. A later
  accepted server image codec may rasterize static SVG as an image without
  exposing SVG parsing or scripting to Pico.
- Linked Glance CSS is not part of the capture, so it is not evidence for exact
  layout values. Supported CSS and limits remain a versioned renderer contract,
  validated against the sanitized semantic fixture and physical 800x480 output.

### Layout and overflow

- Layout targets the logical dimensions and color model reported by the display
  abstraction, not hard-coded 800x480 constants outside the Waveshare adapter.
- Before implementation, research e-paper/dashboard conventions for content
  priority, truncation, wrapping, clipping, pagination, scaling, and overflow
  indicators.
- Every supported container has an explicit overflow rule. No content may write
  outside the framebuffer.
- When content cannot fit, the renderer must return structured diagnostics that
  identify the affected node and chosen outcome: wrapped, clipped, elided,
  scaled, paginated, or rejected.
- The default must preserve legibility over completeness; shrinking text below
  the documented minimum size is forbidden.

### Initial document and render budgets

- Accept at most 32 KiB of encoded HTML per update. Reject a larger declared or
  streamed body before parsing or framebuffer mutation.
- Target at most 5 seconds from complete input to a validated framebuffer on
  Pico 2 W. This is a measured performance target, not a correctness timeout.
- Use a 15-second watchdog only to recover from stalled or defective execution.
  A valid document is bounded primarily by bytes, nodes, depth, text, assets,
  and parser/layout work; elapsed time must not silently change rendering rules.
- Panel power-up, SPI transfer, physical refresh, and BUSY waiting are measured
  and reported separately from the 5-second software-rendering target.
- These are initial acceptance limits. Tightening or increasing them requires
  repeatable Pico RAM, flash, stack, latency, and energy measurements plus an
  explicit requirements update.

### Update timestamp

- Reserve a compact rectangle in the bottom-right corner for the date and time
  of the last successfully accepted update. It is neither a full-width strip
  nor an overlay; submitted HTML cannot occupy or paint over this rectangle.
- Compute the rectangle from the selected timestamp font metrics, padding, and
  format. Do not guess a fixed pixel size before framebuffer and physical-panel
  legibility tests. Layout and overflow diagnostics must use the remaining
  non-rectangular document region.
- Failed, rejected, or interrupted uploads do not advance the timestamp.
- The trusted sender supplies the update time as UTC inside the authenticated
  USB or manager-to-Pico record. Pico does not run an independent NTP client.
- The IANA timezone is a non-secret USB provisioning parameter applied by the
  same host workflow that flashes the generic UF2. `EPAPER_TIMEZONE` defaults to
  `Europe/Kyiv` in the local provisioning environment and remains changeable
  without rebuilding firmware.
- The host or manager applies timezone and daylight-saving rules. This avoids a
  timezone database in firmware; Pico receives bounded display-time metadata
  authenticated together with the UTC update time and document.
- Timestamp format, behavior when trusted time is unavailable, and whether the
  overlay may cover submitted content remain explicit decisions.

## Connectivity and security

### Wi-Fi link

- Bluetooth is excluded and must not be initialized or linked unless a measured
  dependency makes that unavoidable.
- Require WPA3-SAE. WPA2, WPA1, TKIP, open Wi-Fi, transition mode, and automatic
  security downgrade are forbidden. An incompatible access point is a visible
  connection error, not a reason to weaken the configured policy.
- Hardware capability does not prove TinyGo-driver interoperability. WPA3 must
  pass a physical join/reconnect test against the target access point.
- Radio power management, reconnect backoff, idle policy, and wake behavior must
  be measured rather than inferred.

### Home manager and public HTTP endpoint

- The accepted control topology is `Internet -> HTTPS/VPN -> home manager ->
  encrypted Wi-Fi device link -> Pico`. Pico is not a public Internet endpoint.
- The home manager exposes one small, versioned HTTPS update API with bounded
  headers and body, fixed timeouts, consistent typed errors, authentication,
  rate limiting, and no debug or secret material in responses.
- The manager retains only the bounded current/pending update and minimum
  operational status required to deliver it; retention and deletion are
  explicit policy.
- Support IPv4 and IPv6 only after each path passes host integration and
  physical-device tests. Parser support alone is not acceptance.
- A bearer token over plaintext HTTP is not sufficient for hostile or public
  networks because it exposes the token and document to interception.
- The device must be securely manageable by an authorized user connecting from
  any Internet source address; access must not depend on the client being in the
  home LAN or on an IP allowlist.
- Public access terminates as HTTPS or VPN at the trusted home gateway/manager.
  Direct public port forwarding to Pico is forbidden.
- Pico initiates or polls the manager over the home Wi-Fi. The manager queues a
  bounded update; Pico does not need a public address or inbound NAT rule.
- The provisioned manager endpoint may resolve to a LAN address, a public IPv4
  address, or a public IPv6 address. Firmware must not require RFC1918/private
  addressing, same-subnet routing, broadcast discovery, or an open inbound
  firewall port.
- The default deployment keeps the manager at home and reaches it through an
  outbound tunnel. The same `device-link` must also support moving the manager
  to a public server without changing renderer, panel, USB, or authentication
  semantics.
- Authentication trusts the provisioned device identity and cryptographic
  proof, never the peer's source IP or whether it appears to be local.
- WPA3 protects the Wi-Fi radio link but is not the only security boundary.
  The manager-to-Pico link additionally uses mutual challenge-response and
  authenticated encryption derived from a USB-provisioned 256-bit device key.
- The long-term device key is never transmitted. HMAC-SHA-256 proves possession
  in both directions; domain-separated session keys and AES-256-GCM protect
  every record. Strict sequence numbers reject replay, omission, reordering,
  corruption, and forgery before HTML reaches the renderer.
- A passive or active LAN observer must learn neither the device key nor HTML
  content and must be unable to authorize an update. This does not protect a
  compromised manager, compromised firmware, physical debug/flash extraction,
  or a trusted host that already possesses the key.
- Public user credentials and the Pico device key are separate secrets. A
  stolen user session must not reveal the device key.
- Bound upload size, nesting depth, node count, attribute/style count, text
  length, decoded asset bytes, parsing work, render work, and failed-auth rate
  to resist memory exhaustion and denial of service.
- A complete request is validated and rendered before it becomes eligible for
  display. Retries need an explicit idempotency/content-hash contract.

### USB-only provisioning

- Build one reproducible generic UF2 containing no SSID, Wi-Fi passphrase,
  public-user token, device key, or manager-specific secret.
- A blank or invalidly provisioned device keeps Wi-Fi and the network update
  path disabled while retaining the USB provisioning and diagnostic channel.
- Authorization material is created or installed only through a physical USB
  provisioning flow, never through Wi-Fi or an HTTP endpoint.
- Wi-Fi SSID, passphrase, and authentication mode are also installed and
  rotated exclusively through USB. Changing them must not require rebuilding
  or reflashing the generic UF2.
- A trusted host tool generates the 256-bit device key with an operating-system
  cryptographic RNG, writes it to Pico over USB, and installs the corresponding
  manager record through a local privileged path. The key is never printed or
  passed through the public manager API.
- Secrets are never committed, printed, logged, returned by status APIs, or
  accepted as command-line arguments that appear in process listings.
- Provisioning must distinguish blank, provisioned, corrupted, rotated, and
  factory-reset states and make interrupted flash writes recoverable.
- Provisioning storage needs versioning, integrity checks, atomic replacement,
  rollback-safe recovery, explicit rotation, and factory reset. Storage cannot
  promise confidentiality against physical flash extraction without a separate
  hardware root of trust; this risk is handled by physical security and key
  rotation rather than cosmetic encryption with a key stored beside the data.
- Network startup requires one complete, integrity-valid record containing the
  device identity and key, manager endpoint, Wi-Fi authentication mode, SSID,
  passphrase, and IANA timezone. Partial or interrupted replacement keeps the
  previous valid record or returns to the clearly unprovisioned USB-only state.

## Display lifecycle, longevity, and energy

- Preserve the physically verified panel wiring, switch settings, controller
  sequence, BUSY semantics, power sequencing, and known-good checkpoint in the
  existing hardware documentation.
- Full refresh is the stable baseline. Partial refresh is a separate,
  hardware-verified optimization with a bounded run and periodic full refresh.
- Never replace BUSY state checks with an unbounded wait or an assumed fixed
  readiness delay. Safety timeouts report the exact stage and observed state.
- Coalesce superseded updates, avoid refreshing unchanged pixels/content, enter
  panel sleep after updates, and deassert HAT `PWR` when the verified lifecycle
  permits it.
- E-paper retention must be used: keeping Pico, radio, or panel awake solely to
  preserve the image is forbidden.
- Refresh interval, partial-refresh policy, reconnect policy, and timestamp-only
  refreshes require measured energy and ghosting evidence.

## Modularity and Go design

- Use narrow Go interfaces at hardware and transport boundaries; internal
  packages use concrete types where interfaces add no testability or replacement
  value.
- Separate document parsing, style resolution, layout, rasterization, display
  surface, panel lifecycle, runtime arbitration, authentication/provisioning,
  USB transport, HTTP transport, and clock/time policy.
- Display capabilities include logical size, supported color model, refresh
  modes, alignment constraints, and lifecycle operations. The renderer must not
  import the Waveshare driver.
- External input becomes typed, validated data at the boundary. Internal code
  must not repeatedly re-parse raw HTML, headers, or credentials.
- Prefer fixed-capacity storage, streaming, explicit ownership, and one 1-bit
  framebuffer. Any additional full-frame or full-document copy needs a measured
  and documented justification.
- Follow normal Go naming, error wrapping, package, interface, and zero-value
  idioms supported by the pinned Go/TinyGo versions.

## Quality contract

- Replace deprecated `golint` with a pinned `golangci-lint` configuration or
  another maintained Go-native toolchain. Exact enabled linters and thresholds
  must be approved and reproducible in the Dev Container.
- Enforce numerical limits for function length, file length, cyclomatic or
  cognitive complexity, and maintainability. Generated/vendor/reference code is
  excluded only by an explicit path rule.
- Limit every non-generated Go file, including tests, to 300 physical lines,
  every function to 60 lines, and cyclomatic complexity to 10.
- Require at least 90% coverage of changed executable lines and prevent overall
  project coverage from decreasing. TinyGo-only code that host Go cannot
  instrument still requires extracted testable logic, target builds, reusable
  fakes, and physical acceptance; it is not silently excluded.
- No new lint suppressions, skipped/deleted tests, weakened assertions,
  unimplemented stubs, secrets, or lowered budgets to make a change pass.
  Explicit 2026-09-06 exception: the user keeps image support enabled while
  skipping only the two BLITZ-IMG-001/002 layout reproductions. Scope, owner and
  expiry are in root `CONSTRAINTS.md`; no other requirement or test is waived.
- Store the final measurable bar in repository-root `CONSTRAINTS.md`; every
  number names its checker, rationale, and run stage.
- For each implementation slice: failing test first where practical, implement,
  run focused checks, run the broader regression suite, perform full code
  review, fix findings, simplify without changing behavior, and repeat until
  there are no actionable findings and all gates pass.
- Keep the local task-end gate within 90 seconds. Move slower exhaustive checks
  to full CI or an explicit physical-acceptance stage; never skip them or abort
  a valid in-progress check merely to satisfy the local duration budget.
- Never treat a successful host test or firmware build as physical acceptance.

## Test strategy

- Unit tests: markup tokenizer/parser, attribute validation, viewport mapping,
  layout, overflow,
  image validation/resampling/monochrome conversion, raster fallback,
  rasterization, timestamp overlay, auth, provisioning state, protocol codecs,
  arbitration, refresh policy, and error mapping.
- Property/fuzz tests: malformed markup/images, decompression bombs, arbitrary
  transport fragmentation, extreme nesting/counts/dimensions, Unicode, integer
  boundaries, replay/corruption, and parser recovery. Fuzz cases must remain
  bounded for manager and embedded semantics.
- Golden tests: supported HTML/CSS examples produce exact monochrome frames and
  exact overflow/error diagnostics at several display sizes.
- Contract tests: reusable fake display, clock, credential store, USB stream,
  and IPv4/IPv6 network adapter exercise modules without Pico hardware.
- Integration tests: identical document behavior over USB and HTTP, USB
  preemption, interrupted provisioning, duplicate/retry handling, resource
  exhaustion, reconnect, and failed authorization.
- Target gates: TinyGo build, flash/RAM/stack report, no unexpected heap growth,
  maximum-document measurements, and Bluetooth absence.
- Physical acceptance: verified panel output, timestamp, repeated USB and Wi-Fi
  transfers, IPv4 and IPv6, power loss/recovery, WPA policy, malformed-input
  recovery, refresh lifecycle, current draw, refresh time, ghosting policy, and
  no regression from the known-good image path.
- Test helpers and interfaces must be reusable by future panel drivers; tests
  must not encode Waveshare constants outside the Waveshare adapter suite.

## Boundaries

Always:

- Use TinyGo, the Dev Container, current official documentation, pinned source
  links, bounded input, USB priority, structured diagnostics, and physical
  acceptance for hardware claims.
- Record facts, measurements, inferences, unknowns, and resolved failures in the
  project documentation.

Ask first:

- Add or upgrade dependencies, weaken a resource/security/quality limit, change
  credential storage, weaken WPA3-only policy, expose Pico outside the trusted
  LAN, add an HTML/CSS feature, enable partial refresh, or add another
  full-frame copy.

Never:

- Add JavaScript execution, Bluetooth, open/WPA1/TKIP Wi-Fi, plaintext public
  bearer-token exposure, secrets in Git/logs, unbounded parsing/waits/retries,
  hot FPC changes, or undocumented panel sequences.

## Evidence recorded now

- Raspberry Pi documents Pico 2 W as 520 kB SRAM, 4 MB flash, 2.4 GHz
  802.11n, WPA3-capable, and Bluetooth 5.2-capable hardware:
  https://www.raspberrypi.com/documentation/microcontrollers/pico-series.html
- TinyGo documents `pico2-w` Wi-Fi through `soypat/cyw43439`:
  https://tinygo.org/docs/reference/microcontrollers/boards/pico2-w/
- The pinned `cyw43439` v0.1.1 source exposes WPA3 and WPA2/WPA3 join modes;
  target hardware interoperability is still a physical research gate:
  https://github.com/soypat/cyw43439/blob/v0.1.1/wifi.go
- Current `lneto` documents small HTTP/1.1 primitives and core IPv6 support,
  but no TLS 1.3 implementation. The project-pinned `lneto` v0.1.0 must be
  checked separately before claiming IPv6 server support:
  https://github.com/soypat/lneto
- Exact panel/HAT sources and reusable secondary material remain indexed in
  `../../docs/epaper-hardware-sources.md` and
  `../../docs/epaper-code-reuse-research.md`.
