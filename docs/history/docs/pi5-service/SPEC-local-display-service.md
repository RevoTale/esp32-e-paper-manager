# Spec: local-display-service

Status: implemented and software-tested; target acceptance remains separate.

## Objective and structure

An ordinary Go Debian process receives HTML or PNG, renders a complete frame
and owns serialized screen updates. Reuse `engine.RenderDetailed` and
`display.Frame`; do not fork the renderer or modify Pico provisioning.

Proposed packages in `experiments/03-remote-epaper/`:
`localdisplay/` for orchestration and HTTP tests, `cmd/epaper-local/` for Linux
composition. Go 1.26.8; portable dependencies retain their TinyGo checks.

## Proposed API

HTTP over a Unix-domain socket, default `/run/epaper-local/api.sock`, mode
0660. Bearer token supplied through a private service credential; never logged.
No TCP listener by default. Docker mounts the socket directory, not GPIO.

Transport choice accepted by the user on 2026-09-08 after discussing direct
`/dev` access: HTTP over a pathname Unix socket, 0660 with a dedicated group,
plus a bearer token. The host service owns hardware; the container receives
neither GPIO/SPI devices nor privileged mode. This is not a custom device node.
On Linux, connecting to this socket is governed by filesystem permissions:
https://man7.org/linux/man-pages/man7/unix.7.html.

Mount the dedicated socket directory, not the individual socket inode, so a
service restart can replace the socket without requiring a container restart.
Clients must not receive directory write permission. A read-only bind mount is
not API authorization: it can still permit socket connections and mutating
HTTP requests, which must pass the token and input checks.

- `GET /v1/status`: capabilities, phase, latest attempt/result and refresh floor.
- `POST /v1/frame`: `text/html; charset=utf-8` or `image/png` body. One complete
  update per request. HTML maximum 32768 bytes; PNG encoded maximum 1 MiB and
  decoded maximum 1 megapixel, checked before full decode. No network fetching.
- Success: HTTP 200 only after controller refresh and sleep complete, labelled
  `controller-complete`, never `visually-confirmed`. Include source-free
  rendering warnings, update ID and timing.
- Reject unauthorized requests with 401, oversized input with 413, unsupported
  MIME with 415, invalid content with 422, busy/cooldown with 429 and
  `Retry-After`, device faults with 503. Never return success for a failed upload.

One in-flight update, no unbounded queue or hidden resubmission. Authenticate
and enforce admission before rendering. Validate/decode completely before
hardware mutation. A disconnect before hardware work cancels the job; after
activation finish the bounded hardware lifecycle independently of the client.
Expose the outcome through status when the HTTP response cannot be delivered.

## Rendering and lifecycle

Default logical viewport 250 by 122 landscape, transformed once into native
122 by 250 panel coordinates. Add corner-labelled rotation golden tests.
PNG is composited on white and fitted preserving aspect ratio, then uses the
same deterministic monochrome conversion. HTML uses the current bounded
inline-style profile; report clipping instead of claiming browser compatibility.

Keep a small bottom-right last-full-refresh timestamp. Define it as the update
start timestamp printed by the last successfully completed full-refresh cycle;
it cannot show a future measured completion time in the same frame. Use host
time and configurable IANA zone, default Europe/Kiev. No periodic repaint on
startup or merely to update the clock.

Conservative proposed refresh floor: 180 seconds, including after restart;
document it as project policy, not a measured lifetime guarantee. Skip unchanged
content before changing its timestamp. Full refresh only initially. Failures
invalidate deduplication state. No retry loop. Sleep after successful update.

## Verification and operations

Planned commands from module directory after implementation:
`go test -race -cover ./localdisplay ./cmd/epaper-local` and
`CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/epaper-local ./cmd/epaper-local`.

Tests must cover real HTML/PNG conversion plus fake device traces, unauthorized
and malformed submissions, bounded decoding, token redaction, parallel clients,
unchanged suppression, restart floor, disconnects and stage failures. Add a
real Unix-socket HTTP test and signal/shutdown test. Keep all repository gates.

Always preserve old endpoints and firmware. Ask before exposing a TCP endpoint
or relaxing limits. Never pass user-supplied paths/commands to the OS or fetch
remote image URLs. Runtime acceptance must include actual Docker submission
and separately observed physical output.
