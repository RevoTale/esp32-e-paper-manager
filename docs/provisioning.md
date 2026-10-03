# USB-only provisioning

Migration note (2026-09-07): current source tools use EPCQ/EPCR **version 2**.

Unified boot recovery additionally reports result **5 / Recovery** when no safe
boot lifetime exists. This is control-only: no EPS2 identity, panel update or
networking. Inspect/diagnose still reports actual credential state; unreadable
storage takes precedence as result 3 / Storage with state 3 / Unknown. Provision
and rotate are disabled until recovery completes and the board reboots.

In this mode only, explicit `erase -confirm-erase` first erases and verifies all
bytes of both credential blocks, then erases/verifies both epoch blocks. Success
means erased, **not** a live runtime: reboot, remove retained manager/USB claims
and pixel baselines, then provision with a freshly generated key. Never restore
an old enrollment key after resetting epochs. Normal runtime credential erase
does not touch epochs. A missing acknowledgement is ambiguous; no automatic
mutation retry. See [ADR-014](../decisions/014-durable-device-sessions.md).
Do not use them with the immutable older firmware checkpoint. Unified EPS2
runtime packaging is still in progress; no new physical acceptance is implied.

The generic UF2 contains no SSID, passphrase, API token, device ID, or device
key. Provision only over physical USB CDC:

```sh
go build -trimpath -o epaperprovision ./cmd/epaperprovision
printf '%s' '{"ssid":"WPA3-NETWORK","passphrase":"SECRET","manager":"tcp://manager.example:9757","timezone":"Europe/Kyiv"}' |
  ./epaperprovision -port auto -registry ./device-enrollment.json provision
```

The JSON secret comes from stdin, not command arguments. `timezone` is optional
and defaults to `Europe/Kyiv`. `device-enrollment.json` is written atomically
with mode `0600`; it contains the manager copy of the generated device identity
and 256-bit key. Never commit or share it.

Port selection (2026-09-07 pure-Go host tools): the examples' default `auto`
uses USB VID/PID `2E8A:000A` on Linux. Matching ignores hexadecimal letter case;
zero matches fail and multiple matches require an explicit `-port`. Never
select the first matching Pico when more than one is attached.

On macOS, pass `-port` with the exact `/dev/cu.usbmodem...` path on every
operation. Use `epaperctl -list` to list port names, then identify the intended
physical device. The pure-Go listing cannot establish USB VID/PID identity;
neither a pathname pattern nor a single returned name grants that identity.
Automatic selection rejects before opening any device. This follows the
existing epaperctl policy and removes the Cgo-only detailed macOS enumerator
from the provisioning binary. Explicit-port protocol and transactional
enrollment behavior are unchanged. No OS subprocess or Cgo fallback is used.

Commands:

```sh
./epaperprovision inspect
./epaperprovision diagnose
./epaperprovision -registry ./device-enrollment.json rotate < config.json
./epaperprovision -confirm-erase erase
```

Only WPA3-SAE is accepted. Responses redact the Wi-Fi passphrase and device
key. Two flash erase blocks form an integrity-checked generation journal;
interrupted writes retain the previous valid generation. The final two flash
blocks remain reserved for the older boot-epoch store. Rotation preserves the
device ID but changes the key. The legacy runtime requires a reboot after
provisioning/rotation; the unified runtime will revoke connections and reload
the authoritative journal without a separate USB firmware.

If storage is blank, corrupt, or invalid, Wi-Fi stays disabled and USB remains
available. Erase is destructive and requires the explicit confirmation flag.

## Version 2 control contract

Both request and response remain 512 bytes with CRC32 at bytes 508–511
(big-endian). Byte 4 is 2. Unknown versions, fields, nonzero reserved/padding
bytes, inconsistent public metadata and impossible success states reject.
There is no v1 fallback: rejecting the request version must happen **before**
an old runtime could mutate credentials. The persisted `EPC2` flash journal
remains version 1; no destructive storage migration is needed.

Response byte 12 is the result: 0 success, 1 configuration, 2 state transition,
3 storage, 4 owner busy. State byte 6: 0 blank, 1 provisioned, 2 corrupt,
3 unknown because the journal could not be read. Unknown always reports storage
failure and empty identity/configuration metadata. SSID and public endpoint may
appear in inspect output; passwords and keys never do.

An acknowledged write error does not prove nothing changed. `Service.Report`
always reloads: a valid slot may have been committed before a verification error,
and two-block erase can fail after erasing one block. Its response and runtime
configuration come from that reload, not the request. Even an apparent successful
operation must satisfy its postcondition: erase reloads blank; provision/rotate
reload the exact requested config and acknowledged storage generation. Otherwise
return storage failure with the actual state. Never report success optimistically.

The unified owner must revoke **all** old authentication contexts/queued work
before mutation, fence screen leases without resetting refresh cadence, and
adopt only the loaded config afterward. Blank/corrupt/unknown disables networking.
An old authenticated-but-unbound connection cannot obtain a new post-rotation
lease. Physical refresh finishes safely before the serialized owner can mutate.

The native client checks the echoed operation, numerical result and canonical
response. Ambiguous I/O/framing poisons its transport: close/reopen and inspect;
never automatically replay a mutation. Retain the private enrollment candidate
for recovery. Acknowledged numeric failures still return the authoritative state.

USB multiplexing selects `EPS2` or `EPCQ` only at complete record boundaries.
Pixel payloads containing those bytes are ordinary pixels. Partial byte records
have bounded idle/total deadlines; malformed input requires DTR disconnect and
buffer drain, not scanning arbitrary bytes for a new magic string.

## Host enrollment transaction (2026-09-07)

Fact: before sending provision/rotate, the host exclusively creates
`device-enrollment.json.pending` with mode `0600`, writes the generated identity,
key and timezone, and synchronizes both the file and its directory. The active
enrollment remains unchanged. The pending file contains no Wi-Fi passphrase.
An existing pending file, including a symlink or directory, blocks another
attempt. Staging failure does not send the credential mutation; any partial
candidate remains as evidence and also blocks retries.

Fact: only the original successful response for the requested operation can
promote the candidate. Its state must be provisioned, its generation nonzero,
and its device ID, authentication mode, SSID, manager endpoint and timezone must
exactly match the request. The native v2 service separately verifies the whole
persisted configuration, including its secret fields, before acknowledging.
The host rechecks both file snapshots, atomically renames pending to active in
the same directory, then synchronizes that directory. No automatic replay,
discard, rollback or recovery promotion occurs.

Fact: existing active and pending snapshots must be regular mode-`0600` files,
contain valid enrollment JSON and be at most 1024 bytes. Symlinks, replacement
inodes, altered bytes and unexpected active-file creation/removal reject.
The directory is held open throughout the transaction using Go `os.Root`.
Keep it in operator-controlled local storage and serialize external edits;
these checks do not provide a compare-and-swap against a compromised host or
arbitrary concurrent manual file replacement. The filesystem must support
atomic same-directory rename and directory synchronization.

Recovery codes are host diagnostics, separate from the device's `code=`:

| Host result | Action |
| --- | --- |
| `recovery=1` | An unresolved pending exists. Retain both files and inspect over a reopened USB connection. |
| `recovery=2` | Staging failed before the credential mutation. Active is preserved; inspect storage and retain any partial pending. |
| `recovery=3` | USB failed, the device rejected the operation, or the ACK did not match. Retain active and pending; reopen USB and inspect. |
| `recovery=4` | ACK matched but file validation/rename failed. Retain active and pending, repair storage and inspect. |
| `recovery=5` | Rename succeeded but directory synchronization failed. The candidate is at the active path; retain it and verify storage durability before any new mutation. |

Never select a key merely because later inspect metadata matches: key rotation
can leave all public fields unchanged. Recovery requires proving which retained
key authenticates with the device through an existing controlled authenticated
workflow, or explicitly erasing/rebooting and provisioning a fresh key into a
new enrollment path after revoking previous manager claims. Keep unresolved
files until that decision is complete; the CLI does not delete them for you.

Unknown: sudden host power-loss durability and the complete USB rotation flow
on the physical Pico remain target acceptance tests. Host fault tests cover
lost/rejected/mismatched ACK, duplicate candidates, file changes, write/sync and
rename failures; they do not prove physical flash, radio or panel behavior.

Sources: [Go file synchronization](https://pkg.go.dev/os#File.Sync),
[directory-relative operations](https://pkg.go.dev/os#Root),
and [rename guarantees](https://pkg.go.dev/os#Rename).
