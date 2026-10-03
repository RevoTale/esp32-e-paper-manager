# Screen API

Use this API from a trusted dashboard producer to submit HTML or update existing
elements by ID. The Go manager renders the document; ESP32 receives prepared
pixels over EPS2/EPN2, not HTML. No Blitz worker is required.

Start with [container deployment](container-deployment.md). The current API is
HTTPS and loopback-only. A producer container must share the manager's network
namespace or use a separately reviewed proxy. A Docker service name alone cannot
reach a listener bound to another container's loopback interface.

## Submit a scene

1. Authenticate with `Authorization: Bearer <API token>` and verify the TLS
   certificate. The API token is separate from the ESP32 enrollment key.
2. `GET /v2/screen`; retain the exact `ETag` response header, including quotes.
   The initial body is empty but still has an ETag.
3. `PUT /v2/screen` with that ETag in `If-Match`, content type
   `text/html; charset=utf-8`, and the complete UTF-8 document as the body.
4. `202 {"revision":N}` means queued, not displayed. Check
   `GET /v2/screen/status` for subsequent delivery evidence.

Example document:

```html
<div style="padding:24px;background:white;color:black">
  <h1>Tasks</h1>
  <p id="status">Ready</p>
</div>
```

Keep tokens in private files or secret mounts; never bake them into images,
HTML, URLs or command arguments. With curl, supply the Authorization header
through standard input (`--header @-`), not a literal token argument.

## Routes

| Request | Result |
| --- | --- |
| `GET /v2/screen` | Current complete HTML and strong ETag |
| `PUT /v2/screen` | Replace the complete scene; exact current If-Match required |
| `PATCH /v2/screen` | Atomic by-ID edits; JSON body and exact current If-Match required |
| `GET /v2/screen/status` | Revision, delivery, refresh and diagnostic evidence |

All routes require the API token. PUT/PATCH bodies are bounded to 32 KiB.
Do not gzip requests. Mutations share a 30-attempts/minute process-local budget
and two body-validation slots. These admission limits are distinct from panel
refresh cadence.

## Update an element

After fetching the current ETag, send PATCH with `Content-Type: application/json`:

```json
{"edits":[{"id":"status","text":"Three tasks remaining"}]}
```

One batch is one scene revision; all edits succeed or none do. IDs must exist
and be unique. Do not target both an ancestor and its descendant in one batch.
Up to 64 edits are allowed. See the [authoring contract](engine-authoring-api.md)
for text/HTML/attribute/removal rules; its dated implementation notes are history.
The current panel profile performs full physical refreshes: a small JSON PATCH
does not promise partial physical refresh.

## Conflicts and restart

An ETag prevents one writer from overwriting another writer's newer document.
On `412`, fetch current HTML/ETag, decide how to reconcile, then resubmit. Do not
blindly retry with `If-Match: *`; weak tags, wildcard and tag lists are rejected.
Missing If-Match returns `428`. The tag changes across manager restarts, and the
in-memory document is not persisted. The producer must resubmit after restart.

Other common responses: `400` invalid input/options, `401` unauthorized, `413`
body too large, `415` unsupported type/encoding, `429` admission limit. A queued
PUT may later fail rendering; inspect status rather than treating 202 as success.

## Refresh and status

Optional mutation headers: `X-Update-Priority: normal|urgent` and
`X-Refresh-Mode: auto|partial|full`. Defaults are normal/auto. Duplicate or empty
option headers reject. Partial is a request, not a capability guarantee; the
accepted full-only firmware does not advertise partial refresh. The separate
candidate and `-experimental-partial` manager option enable negotiation, not
automatic physical qualification.

With negotiated `-refresh-policy`, normal/urgent defaults are 180s/30s. Urgent
cannot interrupt BUSY or bypass panel safety. Maintenance full refresh defaults
to 10 minutes. Optional debounce/max-wait batch scenes on the manager; neither
overrides device timing. See [refresh policy](refresh-priority.md).

- `current`: latest accepted scene revision.
- `in_flight`: revision currently being delivered, or zero.
- `delivered`: last revision with actual terminal delivery success.
- `confirmed`: newest revision whose pixels are known equivalent; identical
  pixels can advance it without transmitting another frame.
- `refresh_trusted`, `full_refresh`, `in_flight_cycle`: physical-refresh evidence
  and its validity; not proof of what a human sees.
- `failure` and `warnings`: bounded diagnostics when present.
- `refresh_failure`: optional `{revision, reason}` for a requested partial mode
  rejected before device I/O. Reasons are `full-refresh-required` (no trusted
  base or consecutive-partial limit) and `region-budget` (damage exceeds local
  region limits). This is not invalid HTML and does not stop the manager or
  silently send the rejected target as full. The next accepted submission clears
  it; maintenance may still refresh the older confirmed image. Without explicit
  partial configuration the endpoint returns501 before accepting the mutation.

Candidate-only options require `-screen -refresh-policy -experimental-partial`
and an800×480 viewport. `-partial-interval` and `-partial-urgent-interval` default
to1s, `-partial-max-updates` to5, and `-partial-max-bytes` to12000 per old/new
plane. These are configurable operator budgets, not manufacturer safety limits
or qualified performance. The byte limit bounds one final damage rectangle;
unrelated edits are composed before damage detection. Auto may choose full;
explicit partial rejects if its budget or confirmed baseline is unavailable.

Zero revisions with `refresh_trusted:false` are normal before a first delivery.
For unknown outcomes, follow [delivery recovery](screen-delivery-recovery.md),
not repeated blind sends. Physical acceptance still requires checking the panel.
