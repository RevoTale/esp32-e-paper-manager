# Working rules

- Read `CONSTRAINTS.md` and the relevant current guide before editing. Historical
  decisions are in `docs/history`; archived commands and hardware mappings are
  not automatically applicable to this checkout or board.
- Preserve the original Pico repository. This receiver is ESP32 C/ESP-IDF;
  rendering, manager and portable protocol packages are Go.
- Run project commands in the already-running matching Dev Container. Never
  create/rebuild/start a container or run project builds on the host without
  explicit permission. Do not flash, restart services, send hardware frames,
  rotate keys, commit or push unless explicitly requested.
- Do not invoke Python except the previously approved official ESP-IDF tooling.
  Ask before introducing other Python use.
- Verify hardware claims against exact board/panel documentation and official
  source code. Never infer GPIO/FPC orientation or refresh compatibility from a
  different model. Explain uncertainty before asking the user to touch hardware.
- Prefer tested modules and primary-source practices over rewrites. Evaluate
  memory, bandwidth, panel lifetime and energy cost for each design.
- Preserve golden vectors and failure cases. Run `make quality`, reporting each
  unrun or failing boundary separately from hardware acceptance.
- Record debugging as symptom, measured evidence, cause versus hypothesis, fix,
  verification, regression protection and source links. Keep current instructions
  separate from history; never turn an assumption into a documented fact.
- Keep feedback concise. Ask when missing authority, hardware action or a
  material product choice blocks safe progress. Never log secrets.
