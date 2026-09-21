# Engineering decisions

- User-approved exception, 2026-09-12: official ESP-IDF Python installation,
  configuration, build and verification tools may run inside the matching
  Dev Container for the ESP32 port. The user approved Python after the SDK
  dependency was explained. No custom Python scripts, host Python execution,
  Python server/device runtime or container rebuild is authorized by this
  exception. Preserve the existing TinyGo/Pico firmware and Go manager.

- Persist approved development/build dependencies in the Dev Container
  Dockerfile or configuration, with appropriate version pins and prerequisites.
  Installing only into a running container is temporary and is not completed
  environment setup. Check that recreation can restore the tools; distinguish
  configuration review from an actually verified rebuild. Do not rebuild or
  replace the container without the user's permission.

- User-approved exception, 2026-09-05: the upstream Stylo build-time Python
  property generator may run as part of building Blitz inside the project's
  Dev Container without repeated approval. This does not authorize other
  Python scripts, host Python execution, or Python in the renderer runtime or
  Pico firmware. All other no-Python and container-management rules remain.

- Historical status, 2026-09-07: the Blitz/Stylo renderer source and Rust-only
  Dev Container setup were removed after native Go preservation tests passed.
  The preceding approval documents checkpoint `37a1471`; it is not an active
  renderer build instruction or permission to execute other Python tooling.
  Keep Python/C/CMake/Pico SDK prerequisites for preserved physical experiments.
  Exact removal and test map: `experiments/03-remote-epaper/docs/blitz-removal-map.md`.

- Read `CONSTRAINTS.md` before writing code. Never weaken it, add an exception,
  suppress a checker, or reduce a test merely to make a change pass.
- Evaluate every material decision for correctness, energy efficiency, and performance, especially on embedded or battery-powered systems.
- Prefer the simplest maintainable option that reduces CPU, memory, I/O, network, wake time, and power use without compromising correctness.
- Verify consequential conclusions against current primary or official documentation when practical. Distinguish documented facts, measurements, assumptions, and uncertainty; do not claim improvements without evidence.
- For electronics, verify each pinout, voltage, polarity, timing, connector or FPC orientation, switch setting, power sequence, controller, and cross-brand compatibility before giving physical instructions or changing firmware.
- Prefer current manufacturer datasheets, schematics, reference designs, and official source code. When first-party documentation is incomplete, cross-check available reputable external resources, identify them as secondary evidence, and state what remains unverified.
- For this project's e-paper hardware, follow the source hierarchy and compatibility notes in `docs/epaper-hardware-sources.md` before changing wiring, power sequencing, SPI settings, or display initialization.
- Before implementing or debugging hardware support, search for existing source code that matches the exact controller, display revision, HAT revision, and MCU where possible. Review it instead of recreating the protocol from memory. Record stable links, the matched hardware/version, useful findings, and incompatibilities in the relevant project documentation so later work can reuse the evidence.
- Keep dashboard rendering paths explicit. Exact Glance compatibility remains a
  server-side 1-bit rendering concern. The proposed v2 firmware may implement
  only a versioned, bounded HTML/CSS subset after its capability map is approved
  and target resource probes pass; it must not grow into a browser engine or use
  Ebitengine.
- Use ordinary full refresh as the accepted baseline. Treat partial refresh as a separate hardware experiment for small changing regions; it is an energy/latency optimization, not a RAM-safety mechanism. Verify exact panel support and waveform behavior, bound consecutive partial updates, perform periodic full refreshes, and retain HAT power-off/sleep after updates.

# Decision memory and regression prevention

- Before increasing bitmap receive storage or implementing the new USB screen
  path, read `experiments/03-remote-epaper/docs/controller-ram-streaming.md`.
  The user selected streaming into controller RAM for investigation. Do not
  present RAM/latency/energy savings as measured or replace the working buffered
  path before interruption, integrity, plane, power and resource gates pass.

- Before changing render batching, z-order, transparency, damage, or screen
  protocol modules, read
  `experiments/03-remote-epaper/docs/render-conflicts-and-module-boundaries.md`.
  Group scene edits, not patches from independent snapshots. Resolve stacking
  contexts and alpha on the manager before monochrome conversion. Prove optimized
  replay equals the final target; planner tests alone do not prove Blitz CSS.

- The preceding Blitz wording is historical after ADR-013/E1. Current manager
  pixel proof uses the native Go engine and the complete transport replay;
  planner-only tests still cannot establish rendering or physical acceptance.

- Before changing the remote-display architecture, read
  `experiments/03-remote-epaper/REQUIREMENTS-v2.md` and
  `experiments/03-remote-epaper/CAPABILITY-MAP-v2.md`. For manager rendering,
  object diff, display-list transport, or Pico rasterization, also read
  `experiments/03-remote-epaper/decisions/008-server-resolved-render-diff.md`.
  For manager HTML/CSS, images, assets, or custom raster fallback, also read
  `experiments/03-remote-epaper/decisions/009-server-image-and-raster-fallback.md`
  and `experiments/03-remote-epaper/docs/manager-html-css-profile-v2.md`.
  Do not implement an open protocol question or silently restore the
  superseded complete-HTML manager-to-Pico flow.
- Before changing e-paper wiring, firmware, timing, or tooling, read `docs/epaper-debugging-history.md`, `docs/epaper-hardware-sources.md`, and `docs/epaper-code-reuse-research.md`. Start from the latest verified checkpoint, not from chat memory.
- After every material experiment or newly resolved contradiction, update the debugging history before starting the next experiment. Record: date, exact hardware and firmware, symptom, facts and measurements, hypotheses, decision or root cause, verification evidence, remaining uncertainty, and the regression guard.
- Label statements as `Fact`, `Measurement`, `Inference`, or `Unknown`. A successful build, SPI write, BUSY transition, USB log, and visible panel output are different evidence boundaries; never substitute one for another.
- Keep corrections append-only in meaning: preserve the earlier failure and mark it superseded or resolved instead of rewriting it as if it never happened.
- Maintain one explicit known-good checkpoint containing the exact wiring, switch positions, firmware artifact or experiment, observed USB result, and visible result. Any change from it must name the single variable being changed.
- For hardware debugging, change one variable at a time, power down before rewiring, and restore the known-good checkpoint after a failed experiment. Do not infer a wiring fault from a voltage measured after firmware intentionally disabled `PWR`.
- For a suspected jumper fault, measure the driven signal at both endpoints during a known HIGH/LOW probe. Continuity alone is insufficient because probe pressure can temporarily restore a loose female Dupont contact. Treat `accepted` and `refreshed` as protocol evidence, never as visible panel acceptance.
- If documentation, code, measurements, or photos disagree, stop. Write down the contradiction and resolve it from the exact hardware documentation or a targeted measurement before giving a physical instruction.
- Every fixed failure needs a durable guard: a test, runtime diagnostic, explicit precondition, wiring table, or documented acceptance check. Do not restore fixed delays where a documented hardware state such as `BUSY` is available; state-based waits must still have a bounded safety timeout and report the failing stage.
- Before declaring e-paper work complete, re-run the smallest relevant software check and a physical acceptance test. Record the actual output; do not claim that compilation or logs prove that pixels changed.

# E-paper learning materials

- Sources in this section are learning materials for understanding possible designs, established practices, failure modes, and case-specific solutions. They are not the project's implementation base, dependency, or material to copy verbatim. Independently verify every adopted idea against the exact panel, controller, HAT, Pico 2 W, TinyGo version, and current primary documentation.
- Treat the exact panel/controller datasheet and Waveshare hardware documentation as authoritative for registers, commands, BUSY polarity, timing, power sequencing, and electrical behavior. Use source-code examples to understand how those requirements are applied, not as a substitute for the specifications.
- Study the pinned official Waveshare 7.5-inch V2 example when investigating initialization, refresh, clear, sleep, image conversion, or physical acceptance checks: https://github.com/waveshareteam/e-Paper/blob/86aa9932f471a50157cf02fafadc2c1b4a965449/RaspberryPi_JetsonNano/python/examples/epd_7in5_V2_test.py#L97. Inspect the matching driver module at the same commit for context. Do not copy it mechanically or enable 4-gray mode unless the exact panel revision is documented as compatible.
- Study InkyPi for higher-level e-paper application practices such as image preparation, display abstraction, scheduling, update lifecycle, recovery, and power-aware operation: https://github.com/fatihak/InkyPi. Treat it as secondary evidence: it targets Linux Raspberry Pi systems, not TinyGo on Pico 2 W, so do not copy its GPIO, electrical, controller, dependency, or operating-system assumptions into the firmware.
- For every adopted practice, document the source commit and file, the exact behavior reused, why it applies to our hardware, and which assumptions are incompatible. Prefer small targeted ports over copying whole drivers or application layers.
