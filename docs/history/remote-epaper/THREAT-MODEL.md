# Wi-Fi threat model

| Asset | Threat | Required control | Fail-closed test |
|---|---|---|---|
| Frame contents | LAN observation | AES-256-GCM after authentication | ciphertext does not contain plaintext |
| Display state | forged frame | mutual PSK proof and AEAD tag | bad proof/tag never reaches receiver |
| Display state | replay/reordering | durable boot epoch plus directional sequence | duplicate, gap, and old epoch rejected |
| Availability | stalled/memory-heavy peer | one connection, 3 s auth deadline, fixed bounds | timeout and oversize close session |
| USB control | Wi-Fi holds lease | USB preempts incomplete Wi-Fi | deterministic ownership matrix |
| Credentials | accidental repository/log leak | empty source vars, non-echoing build wrapper | secret scan and missing-config build |
| Nonce uniqueness | RP2 weak RNG or power cycle | no device RNG; durable epoch reservation | corrupt/exhausted store disables radio |

Residual risk: a physical attacker can extract the compiled PSK; key rotation
requires rebuilding and reflashing firmware. Wi-Fi and USB denial of service
cannot be eliminated, but bounds prevent it from corrupting an accepted frame.
