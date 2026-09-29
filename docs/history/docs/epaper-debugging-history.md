# E-paper debugging history

This is the durable record for the Raspberry Pi Pico 2 W, Waveshare e-Paper
Driver HAT Rev2.3, and 7.5-inch 800 x 480 black-and-white panel. Read it before
changing wiring, display firmware, timing, or the flashing workflow.

Last verified physical checkpoint: 2026-09-07, native Go HTML dashboard via USB.
Unified TinyGo firmware `8b844bb0…2b69b4a` is installed. Bounded native sender
`c8072286…981fa74` returned `screen=confirmed`, exit0, by11:46:22 UTC; user
separately confirmed «Так, новий dashboard видно». A later repeat update returned
`screen=confirmed` on the same firmware, but the user reports the old dashboard
remains. Repeat-update visible acceptance FAILED; see the correction below.
The USB RX-size fix passes software reproduction and this physical scenario;
no hardware trace measured dropped bytes in the earlier failed attempt.
UID and epoch-admission failures were resolved separately before this test.
Wi-Fi is not enrolled/accepted on this candidate. Historical HTTPS-manager
acceptance remains S5; EPS1/S6 artifacts remain unchanged recovery controls.

Evidence labels:

- **Fact**: exact documentation, source code, marking, or configuration.
- **Measurement**: observed with a meter, USB log, or visible display output.
- **Inference**: explanation supported by facts and measurements but not
  uniquely proven.
- **Unknown**: unresolved or not measured.

## Current known-good checkpoint

### Hardware identity

- **Fact:** Raspberry Pi Pico 2 W; Pico SDK builds use
  `-DPICO_BOARD=pico2_w`.
- **Fact:** Waveshare e-Paper Driver HAT Rev2.3.
- **Fact:** purchased Waveshare 7.5-inch 800 x 480 black-and-white kit.
- **Fact:** panel FPC is marked `GDEY075T7`; the controller documentation used
  by the successful control experiment is UC8179.
- **Fact:** HAT switches are Display Config `B` (`0.47R`) and Interface Config
  `0` (4-line SPI).

### Wiring

| HAT signal | Wire | Pico GPIO / rail | Physical pin |
| --- | --- | --- | ---: |
| `PWR` | brown | `GP15` | 20 |
| `CS` | orange | `GP17` | 22 |
| `CLK` | yellow | `GP18` | 24 |
| `DIN` | blue | `GP19` | 25 |
| `DC` | green | `GP20` | 26 |
| `RST` | white | `GP21` | 27 |
| `BUSY` | purple | `GP22` | 29 |
| `GND` | black | `GND` | 38 |
| `VCC` | red | `3V3 OUT` | 36 |

`GND` is the common zero-volt return; it does not supply 3.3 V. `3V3 OUT`
supplies the HAT logic rail. `VBUS` is USB 5 V and is not the HAT `VCC`
connection. HAT `PWR` is a logic-controlled power-enable signal driven by
`GP15`, not the HAT logic supply.

### Connector state

- **Measurement:** in the working assembly, the exposed contacts of the panel
  FPC are visible facing upward at the small adapter board.
- **Guard:** never generalize FPC orientation as "toward the board" or "away
  from the board". Inspect the connector's internal contact side and the exact
  vendor assembly. Unlock, insert square and fully, then lock; do this without
  power.

### Current native Go USB checkpoint — 2026-09-07

- **Hardware:** same identity, wiring, switches and connector state above.
- **Firmware:** unified TinyGo0.42.0, Go1.26.8,
  `experiments/03-remote-epaper/build/screen-device-tinygo-0.42.0-recovery.uf2`,
  SHA-256 `8b844bb0f369989e6f6639dc04f8320cdf1b2853437e2520d6485c9622b69b4a`.
  Filename is historical: this is the unified firmware, not recovery-only.
- **Sender:** native macOS/arm64 pure Go,
  `experiments/03-remote-epaper/build/epaperscreen-usb-rx-480-darwin-arm64`,
  SHA-256 `c8072286c5d6ceb2a0e5477e5fb1fc14499468d7dc79547454f0e3c80981fa74`.
  EPS2 decoded USB chunks480, header32, total<=512; stop-and-wait ACKs.
- **Input:** `experiments/03-remote-epaper/build/native-candidate.gEnGoa/dashboard.html`,
  SHA-256 `5728807c724d3ecc74e15b0a06b2b68829f6d3d3c6bb3f15cec10b30bde3392a`.
  Sender rendered HTML in the native Go engine, timezone Europe/Kiev; timestamp
  sampled after readiness, not the retained synthetic preview. The exact
  timestamped transmitted raster was not separately saved in this attempt.
- **Invocation:** bounded sender with `-timezone Europe/Kiev`, that HTML,
  `/dev/cu.usbmodem1101`. Started11:42:36 UTC; completion observed by11:46:22 UTC,
  exit0 `screen=confirmed (verify visible pixels separately)`. Includes cold
  conservative180s floor; these timestamps are not a transfer-only benchmark.
- **Visible acceptance:** user answered «Так, новий dashboard видно» to the
  explicit check for heading «Мій e-paper dashboard», two blocks and time at
  bottom right. This establishes the native HTML→Go renderer→USB→TinyGo→panel
  path on this assembly, not Wi-Fi, IPv6, partial refresh, peak RAM or energy.
- **Recovery:** retain byte-identical EPS1 `a17ae2a2…a92185f` and captured S6
  controls below. Do not substitute the old frozen package sender or UF2 for
  this accepted pairing. No automatic follow-up update after confirmation.

### Retained physical control — captured S6, 2026-09-06

- **Measurement, visible:** user confirmed «є зображенння» after the inspected
  S6 bitmap was replayed directly. This was the visible checkpoint before the
  native dashboard acceptance above.
- **Fact:** existing manager Screen -> checked BZR4 -> Blitz produced the
  immutable `experiments/03-remote-epaper/build/stream/s6/replay.bzm`, SHA-256
  `85255ce0e5e2849b029b30885aeadea36eb6de49cd46445c857a9db193468011`.
  Its decoded preview matches the earlier S6 preview. The file, not freshly
  rendered HTML, was then sent through the same raw client as the successful
  half-black control. Firmware, wiring, switches and refresh guard unchanged.
- **Measurement, protocol:** one direct send, exit 0 by 17:05:31 UTC,
  `stream=committed`, followed by the separate user observation. Exact client
  invocation/hash and immutable artifact are recorded in the replay entry below.
- **Inference:** current renderer can produce a displayable S6 frame, and the
  raw USB/firmware/panel path can show it. No cause of earlier failed sends has
  been proven; transient contact and live-process/session differences remain
  unexcluded. This is not acceptance of the live HTTPS manager S6 delivery.
- **Guard:** retain the exact successful bitmap and PNG for byte-identical
  replay; do not replace them with a fresh render under the same filenames.
  No further send after confirmation. Compare live delivery against this
  boundary before altering hardware or claiming a software fix.

### Previous physical control — bitmap-only, 2026-09-06

- **Measurement, visible:** user confirmed «є» after the direct raw upload:
  black left half and white right half appeared. This replaces the formerly
  visible S5 image as the latest physical control.
- **Fact:** same firmware, pins and switches; no agent-triggered reset or
  reflash. Artifact `experiments/03-remote-epaper/build/stream/raw-control.bzm`,
  SHA-256 `da78c255621494fd363998ddea4f6b5343c8c795ce325d8e0ad5750e20b33676`.
  Existing S6 client raw mode; no HTML, Blitz or manager. Exact invocation and
  client hash are recorded in the bitmap-only experiment below.
- **Inference:** direct USB, firmware and the physical display path worked in
  this attempt. This does not identify the S4/S6 failure or rule out intermittent
  electrical contact. A permanently broken rendering-to-panel path is not
  supported by this successful control.
- **Next isolation:** capture and inspect the exact rendered S6 BZM1 bytes,
  then replay that immutable file through this same raw path after authorization.
  Do not change wiring or firmware, and do not infer a Blitz defect yet.

### Previously accepted manager result — S5, 2026-09-06

- **Fact:** experiment `experiments/03-remote-epaper`, isolated streaming
  `build/stream/stream-device.uf2`, SHA-256
  `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
  Hardware, wiring and switches above were unchanged; no reflash for S5.
- **Measurement, protocol:** one HTML scene passed through the macOS loopback
  HTTPS manager, actual container Blitz renderer and supervised direct USB
  client. Status reached `confirmed=1,current=1,in_flight=0` after the existing
  180s startup guard, by 00:43:25 Europe/Kyiv.
- **Measurement, visible:** the user then replied «так» to seeing `MANAGER S5`
  and lower-right `ТЕСТ S5 · 06.09.2026`. This separately confirms the new
  scene appeared, not merely that the protocol completed.
- **Fact:** manager stopped cleanly; no background listener, extra update or
  agent-triggered Pico reset. The corner is a static test label, not the
  last-full-refresh timestamp feature.
- **Unknown:** new Wi-Fi path, partial refresh, remaining failure scenarios
  and measured energy/resource acceptance are still open. The buffered path
  remains available for recovery; this checkpoint does not replace it.
- Exact host artifact hashes, invocation and TLS compatibility evidence:
  [S5 manager test](../../screen-manager-usb.md).

### Historical first visible control — 2026-09-02

The following records the earlier smiley experiment, not the current firmware.

- Experiment: `experiments/11-gdey075t7-smiley`.
- **Measurement, USB:** reset released with `BUSY=1`; controller power-on
  returned idle after 200 ms; 48,000 bytes were sent to each of planes `0x10`
  and `0x13`; refresh returned idle after 2,500 ms; power-off returned idle
  after 100 ms; deep sleep `0x07/0xA5` completed; HAT `PWR=0`; final result was
  `RESULT=DONE`.
- **Measurement, frame:** checksum `0x2a1032a4`, 24,222 requested black pixels.
- **Measurement, visible:** the user confirmed that a smiley appeared. Its
  background was black.
- **Conclusion:** the Pico, HAT, adapter, FPC, panel, power path, SPI/control
  path, refresh, and sleep path can work together. Frame polarity is not yet
  accepted because the intended background was white.

The e-paper image persists after power-off. Seeing an old image proves neither
that the latest firmware ran nor that a refresh succeeded.

## Resolved problems and regression guards

### 1. Exact hardware was conflated with similar Waveshare products

- **Symptom:** driver, pin, and connector advice alternated between an
  integrated `Pico-ePaper-7.5` board, a separate Driver HAT, and the panel
  manufacturer marking.
- **Cause:** selection by screen size or brand instead of the complete hardware
  identity.
- **Resolution:** identify all four boundaries separately: Pico 2 W, Driver HAT
  Rev2.3, purchased Waveshare product, and panel/controller marking. Waveshare
  is the product baseline; the exact Good Display panel sample is an explicit
  control experiment, not an automatic replacement for all Waveshare code.
- **Guard:** follow `epaper-hardware-sources.md`; never select a driver from
  `7.5-inch` or `800 x 480` alone.

### 2. Incorrect FPC orientation instruction

- **Symptom:** the panel FPC was instructed to face the wrong way.
- **Cause:** assuming all ZIF connectors contact the same side.
- **Resolution:** the user corrected the orientation using the exact assembly;
  the current contact-up state later produced a visible image.
- **Impact:** no damage was demonstrated. The working visible test is evidence
  that the panel and connector path survived.
- **Guard:** use the connector-contact inspection rule in the known-good
  checkpoint; do not describe orientation solely as up/down without a photo or
  exact connector documentation.

### 3. GPIO and power names were inconsistent

- **Symptom:** `PWR` was at different times described as `GP16` and `GP15`, and
  the distinction among `GND`, `3V3 OUT`, and `VBUS` became unclear.
- **Resolution:** the accepted mapping is the wiring table above: `PWR=GP15`,
  HAT `VCC=3V3 OUT`, and HAT `GND=GND`.
- **Guard:** always state signal name, GPIO or rail name, and physical pin
  number together. Never say only `3V3` or only `pin 36`.

### 4. Four signal wires were shifted by one pin

- **Symptom:** no usable refresh; `BUSY` was connected to `GND`, and four wires
  were offset.
- **Measurement:** after the mapping was corrected, the checkerboard control
  experiment produced visible output.
- **Cause:** physical wiring mismatch, confirmed by inspection.
- **Guard:** compare every wire against the full table before applying power;
  verify signal-wire continuity with power disconnected. A photo or a partial
  row count is not a pinout verification.

### 5. Solder and jumper continuity was unreliable during bring-up

- **Symptom:** continuity existed at a header contact but not reliably at the
  corresponding solder pad or wire endpoint.
- **Measurement:** after reworking the joints, tested paths were approximately
  0.5-0.8 ohm end to end.
- **Resolution:** reflow/rework and meter verification restored continuity. A
  second Pico was later used, but no evidence proves that the first Pico was
  electrically damaged.
- **Guard:** with all power removed, test each required signal end to end and
  test adjacent pins for unintended bridges. Visual appearance alone is not a
  continuity test.

### 6. `PWR=0 V` was initially interpreted as a broken power path

- **Symptom:** HAT `PWR` sometimes measured 0 V after a test.
- **Measurements:** the brown `PWR` path measured about 0.8 ohm; while firmware
  actively drove it, HAT `PWR` measured about 3.2 V; after failure or shutdown,
  firmware intentionally drove it low.
- **Cause:** the voltage was measured outside the active stage.
- **Guard:** log the firmware stage and hold or pulse the signal during a
  targeted measurement. Never diagnose a broken wire from a post-shutdown
  voltage.

### 7. BUSY state was ambiguous and once physically miswired

- **Fact:** for the UC8179 path used by the successful control experiment,
  `BUSY=0` means busy and `BUSY=1` means idle.
- **Symptom:** failed runs showed BUSY fall from about 3.2 V to 0 V and remain
  low. A separate wiring mistake had also placed BUSY on GND.
- **Resolution:** correct `BUSY=GP22`, then report BUSY at every protocol
  boundary. A BUSY level alone does not prove that SPI commands were received.
- **Guard:** diagnostics must distinguish reset release, controller power-on,
  refresh, and power-off waits and report the level, elapsed time, and stage.

### 8. Firmware sent commands before reset release was confirmed

- **Symptom:** an earlier run waited a fixed 200 ms after raising `RST`, sent
  controller power-on command `0x04`, then remained at BUSY low until
  `E_BUSY_TIMEOUT`.
- **Root cause:** the demonstrated protocol error was sending SPI commands
  without first confirming that BUSY had returned high after hardware reset.
  The underlying reason reset release sometimes exceeded or did not match the
  fixed delay remains **Unknown**.
- **Resolution:** `experiments/10-gdey075t7-init-diagnostic` and
  `experiments/11-gdey075t7-smiley` require BUSY high after reset before the
  first SPI command. Initialization and visible refresh then completed.
- **Guard:** wait for documented state transitions, not elapsed time alone.
  Keep a bounded 15-second safety deadline so a disconnected or failed panel
  cannot hang forever, and report the exact failed stage.

### 9. Timeouts and retries hid rather than localized failures

- **Symptom:** fixed waits gave no proof of readiness; vague LED patterns and
  automatic retry proposals could not identify the failing boundary.
- **Resolution:** use state-based waits with a bounded deadline and stable USB
  records such as `RESET_RELEASE_WAIT`, `POWER_ON_WAIT`,
  `DISPLAY_REFRESH_WAIT`, and `POWER_OFF_WAIT`.
- **Guard:** do not retry an initialization automatically until the first
  failure has been reported and the hardware can be returned to a documented
  safe state. A timeout is a safety limit, not the readiness condition.

### 10. USB and OrbStack workflow lost diagnostic output

- **Symptom:** repeated attach/detach operations and starting the serial reader
  after reboot caused the earliest, most useful log lines to be missed.
- **Resolution:** use direct macOS UF2 copying through `/Volumes/RP2350` and
  start the CDC reader before triggering the reboot/copy. Do not switch back to
  OrbStack dedicated USB attachment for this workflow unless the direct method
  stops working and the reason is documented.
- **Guard:** each diagnostic must print a version and stage markers; capture
  output from boot through final result.

### 11. Software evidence was confused with physical acceptance

- **Symptom:** successful compilation, SPI writes, BUSY changes, or
  `RESULT=DONE` were at times treated as proof that the screen changed.
- **Resolution:** require separate boundaries: build, flash, firmware start,
  pin states, command completion, frame checksum/length, and visible output.
- **Guard:** a display experiment is physically accepted only after the user
  confirms the intended unique image. Use a versioned image rather than an
  all-white clear that is indistinguishable from no refresh.

### 12. E-paper persistence made failed firmware look unchanged

- **Symptom:** the old checkerboard remained after later firmware attempts.
- **Cause:** e-paper retains its last image without power; a failed refresh does
  not clear it.
- **Resolution:** use a distinct acceptance image and correlate it with the USB
  version, checksum, and `RESULT`.
- **Guard:** never infer that new firmware ran from the presence of any retained
  image.

### 13. Pico 2 W boot behavior was confused with a reset button

- **Symptom:** instructions referred to a reset action that this assembly did
  not expose as a reset button.
- **Resolution:** use BOOTSEL for mass-storage flashing or the already-proven
  1200-baud USB reboot flow when CDC is available. Build explicitly for
  `pico2_w`.
- **Guard:** name the exact physical control and expected USB device or volume;
  do not use generic "press reset" instructions.

### 14. Remote/Wi-Fi work started before display acceptance

- **Symptom:** substantial remote and Wi-Fi work accumulated while the basic
  display path still had unresolved identity, wiring, connector, and BUSY
  failures.
- **Cause:** no enforced stage gate between physical display acceptance and
  remote features, plus insufficient early research of reusable code.
- **Resolution:** retain reusable TinyGo transport/security work, but do not
  expand it until the exact panel lifecycle passes in TinyGo. Research existing
  exact-hardware and TinyGo sources before writing another driver or network
  layer; record the reuse decision in `epaper-code-reuse-research.md`.
- **Guard:** work in this order: known-good C control -> TinyGo display-only
  acceptance -> frame polarity -> authenticated USB protocol -> Wi-Fi -> power
  and battery measurements. Each stage needs its own evidence and must not be
  inferred from a later or earlier stage.

### 15. `refreshed` повертався, але екран не змінювався

- **Симптом:** USB-клієнт отримував `status=refreshed`, але на екрані лишався
  старий кадр `Firmware installed`.
- **Факт:** e-paper не має зворотного читання пікселів. `refreshed` означає
  лише те, що програмна послідовність завершилась без відомої помилки.
- **Вимірювання:** окрема checker-прошивка передала обидві площини й завершила
  refresh, але видимого нового кадру не було. `BUSY` жодного разу не був
  помічений у стані active-low: `BUSY reads: 3`, `active-LOW reads: 0`.
- **Висновок:** це був хибний idle, а не доказ успішної роботи контролера.
- **Guard:** `accepted` і `refreshed` — статуси протоколу, не фізичне
  підтвердження. Для acceptance потрібен унікальний видимий кадр. Якщо кадр не
  змінився, перевіряємо `BUSY`, живлення й сигнали, а не оголошуємо успіх.

### 16. Нестабільний female Dupont розривав `PWR`

- **Симптом:** логіка HAT мала `VCC=3.2 V`, але дисплей не оновлювався.
  На Pico та збоку female-наконечника `PWR` було близько `3.2 V`, а на вході
  `PWR` HAT — близько `0.01 V`.
- **Змінена змінна:** безпечна probe-прошивка тримала `RST=0` і почергово
  подавала на `GP15/PWR` HIGH та LOW по п'ять секунд.
- **Вимірювання:** HIGH стабільно з'являвся на стороні Pico, але не доходив до
  HAT. Натискання щупом або рух конектора могло тимчасово відновити контакт і
  створювало суперечливі покази прозвонки.
- **Root cause, підтверджено:** слабкий або нестабільний контакт female Dupont
  у ланцюгу `Pico GP15 -> HAT PWR`. Після додавання ще одного перехідника
  female-male з'єднання запрацювало.
- **Не причина:** HTML, USB framing, TinyGo-драйвер, SPI-послідовність і сам
  e-paper не пояснювали різницю напруг між двома кінцями одного `PWR` ланцюга.
- **Guard:** перевіряти сигнал на обох кінцях з'єднання під час відомого HIGH,
  а не лише прозвонкою без навантаження. Якщо з боку Pico HIGH є, а на HAT
  немає — спочатку замінити або обійти jumper. Перед будь-яким переставлянням
  проводів повністю від'єднати USB і батареї.
- **Acceptance:** після виправлення `PWR` користувач підтвердив: «все працює».
  Поточний додатковий jumper вважаємо тимчасовим робочим обходом; для постійної
  збірки краще замінити несправний Dupont або використати надійний конектор.

### 17. Перша повна HTML-передача через USB нарешті працює

- **Дата:** 2026-09-02.
- **Точна збірка:** Pico 2 W, Waveshare e-Paper Driver HAT Rev2.3,
  Waveshare 7.5-inch V2 800 x 480, TinyGo USB firmware.
- **Firmware:** `dist/remote-epaper-usb.uf2`, SHA-256
  `8d9430b8feb76a3811e6308e9b14c1ca1cfa325dbe906348318cd64c1fa676aa`.
- **Факт:** Pico програмно переведено в bootloader через TinyGo CDC 1200 baud;
  UF2 скопійовано напряму через macOS `/Volumes/RP2350`, без фізичного
  `BOOTSEL` і без OrbStack USB passthrough.
- **USB evidence:** `usb=ready`, `usb=transmitted`, `status=accepted`, а потім
  terminal `status=refreshed` для одного й того самого content ID і hash.
- **Visible evidence:** користувач підтвердив появу нового унікального кадру
  `USB UPDATE`: «так, все супер».
- **Acceptance:** повний шлях `UF2 -> boot -> CDC -> HTML transfer -> local
  parse/layout/raster -> panel refresh -> visible image` фізично підтверджено.
- **Важливе обмеження:** це одна підтверджена end-to-end передача. Серія з 20
  оновлень, WPA3, reconnect, живлення від батарей і заміри споживання лишаються
  окремими acceptance-кроками.

Це далось дуже тяжко. Тут не було однієї красивої програмної помилки. Одночасно
нашарувались неправильні припущення про FPC, зміщені піни, пайка, нестабільні
Dupont-контакти, старий кадр e-paper, який виглядав як результат нової прошивки,
занадто слабкі USB-статуси й незручний flash/debug flow. Користувачу доводилось
довго тримати щупи, переставляти дроти, перепаювати Pico і фізично перевіряти
кожен контакт; руки втомились настільки, що збірка розсипалась. Частину цього
мав локалізувати софт раніше.

Саме тому цей результат — не просто «картинка з'явилась». Ми отримали робочий
еталон, відділили програмні докази від фізичних, навчили USB-клієнт чекати
terminal result, довели драйвер на точному залізі й знайшли реальну несправність
`PWR` вимірюванням на двох кінцях. Наступна проблема має починатись із цього
checkpoint, а не з повторення всього шляху.

### 18. Тимчасовий 10-секундний full-refresh demo

- **Дата:** 2026-09-02.
- **Мета:** на прохання користувача показати три різні HTML-статуси з коротким
  інтервалом: `STARTING -> PROCESSING -> COMPLETE`.
- **Факт:** production firmware правильно тримала другий кадр у черзі, бо
  `MinimumRefreshInterval` дорівнює 180 секундам. Перший експеримент тому дав
  `refreshed` лише для `STARTING`; `PROCESSING` лишився `accepted`.
- **Рішення:** додано opt-in build tag `rapid_demo`, який змінює лише мінімальний
  інтервал на 10 секунд. Default build лишається на 180 секундах. Окремий тест
  перевіряє 10-секундне значення, default test suite — production значення.
- **Physical evidence:** усі три окремі передачі повернули `usb=ready`,
  `usb=transmitted`, `status=accepted` і terminal `status=refreshed`.
- **Завершення:** після третього кадру Pico повторно прошито перевіреним
  production artifact SHA-256
  `8d9430b8feb76a3811e6308e9b14c1ca1cfa325dbe906348318cd64c1fa676aa`;
  USB CDC повернувся. Кадр `COMPLETE` лишився на e-paper.
- **Guard:** `rapid_demo` не можна використовувати постійно, додавати до
  release build або Wi-Fi deployment. Це лише явно ввімкнений короткий
  фізичний тест; після нього завжди відновлюється production firmware.

### 19. `accepted` іноді повертає пошкоджений кінець content hash

- **Симптом:** у кількох реальних USB-передачах ID збігався, але останні байти
  hash у проміжному `status=accepted` відрізнялись від правильного terminal
  `status=refreshed`.
- **Measurement:** проблема повторилась на dashboard, великому HTML demo та
  кожному rapid-demo кадрі. Видимий refresh при цьому завершувався.
- **Inference:** це окрема помилка lifetime/encoding проміжного result або
  спільного USB buffer; це не помилка HTML renderer і не доказ зміни контенту.
- **Unknown:** точне місце мутації ще не локалізоване.
- **Guard:** клієнт не повинен довіряти `accepted` hash як terminal evidence.
  Виправлення потребує окремого regression test із реальним порядком
  `accepted -> refreshed`; до цього authoritative є hash із request і
  terminal result.

## Open problems

### Frame polarity

- **Measurement:** the bordered smiley appeared with a black background even
  though the test framebuffer was initialized as white and cleared bits for
  black pixels.
- **Unknown:** which of the two transmitted planes must be inverted or changed
  for the exact panel mode. Do not guess from comments in a different driver.
- **Next proof:** compare the exact successful controller sequence with the
  panel datasheet and vendor sample, then run one versioned black/white polarity
  pattern. Accept only the intended visible result.

### TinyGo integration

- **Fact:** the successful visible acceptance firmware is a Pico SDK C++
  control experiment, not the final TinyGo remote firmware.
- **Next work:** port the reset-release BUSY precondition and verified
  init/refresh/sleep lifecycle into the TinyGo panel driver, preserve detailed
  USB diagnostics, run host tests, then repeat physical acceptance before
  enabling Wi-Fi.

### Remote Wi-Fi security and battery operation

- **Status:** not physically accepted by the successful display control test.
  They remain separate stages. Display acceptance must not be used as evidence
  for Wi-Fi authentication, transport security, sleep current, or battery life.

## Required update template

### 2026-09-05: isolated streaming receiver, not display acceptance

- **Scope:** candidate for Pico 2 W / 7.5-inch V2 / Driver HAT Rev2.3;
  `experiments/03-remote-epaper/streamrx` and `cmd/stream-contract-check` only.
- **Changed variable:** added a typed two-pass transaction state machine and
  checking sink. No working firmware, buffer size, GPIO or panel lifecycle change.
- **Fact:** pinned official Pico driver writes complete planes sequentially;
  no local frame implies investigating two host passes, 96,000 payload bytes.
- **Measurement:** host/race tests pass; receiver coverage 96.3%; zero host
  allocations per successful transaction after construction. TinyGo probe
  cross-build: flash 69,972 bytes, static RAM 7,976 bytes. Not full firmware or
  peak RAM. Quality task PASS, 16 seconds. No flashing/physical experiment.
- **Decision/guard:** abort incomplete or inconsistent passes without commit;
  reject wrong order/identity/padding; bound waiting; replay latest completed
  status without another refresh. Failed intent requires a new ID/full upload.
- **Unknown:** actual streaming SPI/USB behavior, powered-wait cost, recovery,
  peak RAM, visible pixels and energy. Next: wire/adapter integration and exact
  controller acceptance; preserve known-good firmware meanwhile.
- **Details/source:** `experiments/03-remote-epaper/docs/controller-ram-streaming.md`.

## 2026-09-05 — isolated EPS1 USB/SPI streaming integration

- **Scope:** Pico 2 W, 7.5-inch V2, HAT Rev2.3; new `cmd/stream-device` only.
  No physical experiment, flashing or wiring change. Buffered driver lifecycle
  extracted without wire-sequence changes, verified by event-for-event oracle.
- **Fact:** semantic invalid records previously could leave staging usable;
  review added explicit invalidation. Queued USB RX now drains on disconnect.
  Installed TinyGo CDC Write can wait indefinitely for TX space; its loop yields.
- **Decision/guard:** bounded single USB writer worker; stalled response returns
  to owner and aborts panel. Writer remains poisoned until reboot. Tests include
  active Begin plus blocked response, malformed replies, every SPI write fault,
  expired operations and plane equivalence. Never reuse a blocked write buffer.
- **Measurement:** focused race tests pass; quality task PASS in 22 seconds,
  changed coverage 91.0%, total 89.3%. Target build flash 96,152, static RAM 7,964
  bytes. Not peak RAM. Independent review found no remaining required defects.
- **Unknown:** real USB/SPI/visible output, stopped-reader target behavior,
  peak RAM, power/energy savings. No authenticated transport, partial or timestamp
  integration in this experimental firmware. Existing recovery path retained.
- **Details:** `experiments/03-remote-epaper/docs/eps1-usb-streaming.md` contains
  framing, byte costs, artifact paths, source references and remaining gates.

## 2026-09-05 — first EPS1 transfer on the connected Pico

- **Changed variable:** user-authorized replacement of buffered USB firmware
  with isolated streaming UF2; no wiring or HAT switch changes.
- **Facts:** verified known-good recovery UF2 hash; 1200-baud CDC reset exposed
  RP2350 bootloader. Direct UF2 copy succeeded and CDC reappeared. macOS client
  rendered through the existing container Blitz worker and sent one S1 image.
- **Measurement:** client exit 0, `stream=committed`. Firmware hash and exact
  artifacts are recorded in `experiments/03-remote-epaper/docs/eps1-usb-streaming.md`.
- **Visible acceptance:** initially awaited observation; the user then confirmed
  «так» to seeing `STREAM USB` and `ТЕСТ S1 · 05.09.2026`. The first full
  HTML → Blitz → USB → streamed SPI → visible image is physically confirmed on
  Pico 2 W / 7.5-inch V2 / HAT Rev2.3. This is a new streaming checkpoint;
  the earlier buffered recovery artifact remains preserved.
- **Unknown:** repeated-update reliability, fault recovery, energy and peak RAM.
- **Guard:** no additional refresh was sent. Retain distinct protocol and user
  observations; one visible success does not establish the remaining gates.

## 2026-09-05 — second EPS1 image without reflashing

- **Changed variable:** S2 HTML image and new client session, same firmware and
  direct USB workflow. No agent-triggered reset, unplug, flashing or pin changes.
- **Measurement:** client exit 0, `stream=committed`; refresh limiter unchanged.
- **Visible acceptance:** after the pending observation, the user confirmed
  «так» to seeing `STREAM USB — 2` / `ТЕСТ S2`. Both S1 and S2 are physically
  confirmed, including a repeat update without reflashing.
- **Unknown:** long-run reliability, interrupted-transfer recovery, partial
  refresh and energy consumption remain untested on hardware.
- **Guard:** retain the two distinct image confirmations; do not generalize
  two successful updates into completion of the remaining acceptance gates.

## 2026-09-05 — controlled EPS1 interruption before Commit

- **Changed variable:** host client deliberately stops after one acknowledged
  Data record, then drops DTR/closes. Pico firmware and wiring unchanged.
- **Measurement:** physical client exit 0 with `stream=interrupted after first
  acknowledged chunk; no commit sent; closing DTR`. Requested S1 while S2 was
  visible; no completed replacement was sent.
- **Software guard:** explicit Commit rejection in probe; tests cover the
  interruption, port cleanup, real error propagation and host-fixture recovery.
  Race tests and quality task PASS (17 seconds).
- **Visible acceptance:** the user confirmed «так»: S2 remained unchanged.
  The incomplete first-plane upload did not replace the visible image.
- **Unknown:** physical recovery upload and measured PWR shutdown; other
  interruption points remain untested. Do not generalize this single case.

## 2026-09-05 — EPS1 recovery upload after controlled interruption

- **Changed variable:** new complete S3 image/session after interrupted upload.
  Same firmware and wiring; no agent-triggered reset or reflash.
- **Measurement:** direct USB client exit 0, `stream=committed`, limiter unchanged.
- **Visible acceptance:** the user confirmed «так» to seeing `STREAM USB — 3`
  / `ТЕСТ S3`. Recovery after the early first-plane interruption succeeded
  with a complete new upload, without an agent-triggered reset or reflash.
- **Unknown:** other interruption points, power-loss recovery, long-run
  reliability and energy consumption remain separate acceptance gates.
- **Guard:** preserve both protocol and visible evidence; this specific
  recovery scenario is accepted, not every possible failure mode.

## 2026-09-05 — EPS1 interruption in the second plane

- **Changed variable:** host probe cuts the upload after the first acknowledged
  fragment in pass 1, following a complete pass 0. Same target firmware/wiring.
- **Measurement:** client exit 0, `stream=interrupted pass=1 after first
  acknowledged chunk; no commit sent; closing DTR`. 48,000 + 100 logical bytes
  acknowledged using the current chunk size. Requested S1 while S3 was visible.
- **Software guard:** tests verify plane byte counts and no Commit; race tests
  and quality task PASS (16 seconds), changed coverage 91.2%.
- **Visible acceptance:** user confirmed «так»: `STREAM USB — 3` / `ТЕСТ S3`
  remained unchanged after the second-plane interruption.
- **Unknown:** recovery after this later cut and PWR voltage remain unverified.
  No additional refresh sent; other interruption points remain separate tests.

## 2026-09-05 — S4 recovery after second-plane interruption

- **Changed variable:** complete S4 upload after the later cut; same firmware,
  wiring and USB path, no agent-triggered reboot or reflash.
- **Measurement:** client exit 0, `stream=committed`, refresh limiter unchanged.
- **Visible acceptance:** the user confirmed «є» to seeing `STREAM USB — 4`
  / `ТЕСТ S4`. Recovery after the tested second-plane interruption succeeded.
- **Unknown:** other interruption points, power-loss recovery, long-run
  reliability and actual energy/peak RAM remain separate gates.
- **Guard:** four distinct frames and both tested recovery scenarios are
  physically confirmed; do not generalize this to every failure mode.

## 2026-09-05 — manager integration, hardware untouched

- **Scope:** opt-in loopback HTTPS → scene queue → Blitz → supervised EPS1 USB.
  Pico stays on the same streaming UF2 and visible S4 checkpoint.
- **Facts:** bounded API, restart-qualified conditional writes and one
  Screen-level pump implemented. No public deployment or physical transfer.
- **Resolved defect:** review found that ownership per Pump let two pumps for
  one Screen bypass shared cadence/ambiguous-result stopping. Ownership moved
  to Screen; a distinct-pump regression test covers the corrected behavior.
- **Correction:** serial Close alone does not prove a blocked write wakes.
  Pinned serial v1.8.0 explicitly wakes reads; Unix Write uses a blocking
  syscall. Manager now supervises a direct raw USB child with a deadline.
- **Measurements:** task PASS 17s, changed coverage 92.4%, total 89.8%, focused
  races pass. Live TLS smoke returned 401/202/412, exact stored HTML and
  `current=1,in_flight=0,confirmed=0`, then clean SIGTERM. It deliberately had
  no USB device and stopped before startup cooldown.
- **Guards:** no retry after ambiguity, no public arbitrary HTML before renderer
  isolation gates, no physical-success claim from HTTP/fake sink/build results.
- **Unknown/next:** combined manager physical test requires permission for the
  new macOS localhost server. No reflash or rewiring planned.
- **Commands, artifacts and sources:**
  `experiments/03-remote-epaper/docs/screen-manager-usb.md`.

## 2026-09-06 — S5 manager acceptance: client TLS compatibility

- **Scope:** user authorized the new macOS loopback manager and one S5 image.
  Existing hashes match; Pico serial node exists; firmware/wiring unchanged.
- **Failure before submission:** native curl (LibreSSL/3.3.6) exited 35 during
  TLS handshake. Server: `peer doesn't support any of the certificate's
  signature algorithms`. Certificate was the prior Ed25519 smoke fixture.
- **Fact:** failure precedes the HTTP GET; no HTML submission or USB refresh.
- **Next single variable:** replace only the local test certificate with RSA
  2048 for this client. Keep TLS 1.3, explicit CA verification and localhost.
  Do not disable verification, downgrade TLS or attribute this to the Pico.
- **Guard:** verify the actual macOS client/server handshake, not just the
  container client. Runtime success after replacement is not yet recorded.

**Follow-up:** with only the local certificate changed to RSA 2048, the same
macOS curl completed verified HTTPS GET and PUT; manager returned
`{"revision":1}` at approximately 00:39:53 Europe/Kyiv. The original failure
is resolved at the TLS/API boundary. The standard 180s startup cooldown remains;
physical delivery and visible acceptance are still pending at this checkpoint.

## 2026-09-06 — S5 terminal manager delivery and visible acceptance

- **Changed variable:** delivery through native localhost HTTPS manager rather
  than standalone USB CLI. Same streaming firmware/hash, direct USB port,
  container Blitz wrapper, wiring and 180s guard. User explicitly authorized it.
- **Input:** one `acceptance-5.html`, heading `MANAGER S5`, badge
  `ТЕСТ S5 · 06.09.2026`; preview inspected. Badge is a fixed test identifier.
- **Measurement:** after startup cooldown, status became
  `confirmed=1,current=1,in_flight=0` by 00:43:25 Europe/Kyiv. Matching terminal
  EPS1 success passed through the supervised client back to the manager.
- **Cleanup:** native manager stopped with SIGINT, exit 0. No extra image,
  retry, reflash or agent-triggered Pico reboot; no background service left.
- **Initial boundary:** terminal success initially left visible S5 output
  unverified; S4 remained the last user-confirmed frame at that moment.
- **Measurement, follow-up:** the user replied «так» to seeing `MANAGER S5`
  and `ТЕСТ S5 · 06.09.2026`. S5 is now the current known-good visible
  checkpoint for the combined manager → Blitz → USB → Pico → panel path.
- **Unknown:** Wi-Fi, partial and energy are not accepted here.
- **Guard:** distinguish queued revision, terminal confirmed revision and
  visible output; do not use this test to claim all firmware requirements done.

## 2026-09-06 — bounded manager PNG input, hardware unchanged

- **Scope:** isolated `rasterasset` decoder, not a new image on the panel.
  S5, its firmware, wiring, native client and refresh guard remain unchanged.
- **Fact:** reused standard Go PNG/base64 APIs; no dependency, public listener,
  worker protocol, image fetcher or MCU decoder was added.
- **Resolved defect:** excess base64 `=` padding could reduce a preliminary
  length estimate and allocate before rejecting. A failing allocation test
  reproduced it; an explicit padding-count guard now rejects without allocation.
- **Measurement:** unit/race coverage 100%; two bounded fuzz runs passed;
  independent review had no Required finding; task gate PASS 17s, changed
  coverage 92.6%, total 89.8%, existing USB/Wi-Fi builds pass.
- **Unknown/next:** connect bounded per-scene assets to actual Blitz, verify
  image layout/alpha and process resources. This does not establish `<img>`
  support in the live manager, new physical acceptance or energy savings.
- **Guard and sources:**
  `experiments/03-remote-epaper/docs/manager-image-assets.md`.

## 2026-09-06 — PNG integration finds a Blitz layout blocker, not a Pico fault

- **Scope/change:** approved `golang.org/x/net v0.58.0` for manager-side HTML
  parsing. Added scene budgets, deduplicated PNG → straight RGBA8 assets, a
  streaming local BZR2 encoder, bounded Rust decoder and resource injection.
  No renderer dependency, firmware, wiring, USB delivery or panel timing change.
- **Observed:** the 2×2 raster is loaded, positioned at (2,1), but has layout
  0×0 and produces no pixels. A separate explicit 4×4 `contain` box becomes
  3×2 instead of retaining its box and clipping. Half-alpha composition passes.
- **Rejected hypothesis:** loading resources before resolve versus between two
  resolves gives the same failures. Independent review reproduced them and
  confirmed that the public resource API invalidates layout correctly.
- **Diagnosis:** pinned Blitz beta.1 clamps replaced-element size to available
  space. This explains the 4×4 reduction; the 0×0 case is consistent with the
  same mechanism but exact measure inputs are not yet observed. This is a
  host-only renderer failure, not evidence of a new cable/panel defect.
- **Candidate:** upstream PR #605's sizing correction is present in published
  beta.2, inspected read-only. Need separate approval to evaluate the dependency
  update; do not claim it fixes both tests before executing them.
- **Additional guards:** tests reproduced HTML parser-mode mismatches for
  `noscript` and XHTML doctypes. Matching scripting-disabled mode and rejecting
  non-HTML5 doctypes prevent unvalidated image interpretation through those paths.
  Exhausted aggregate pixel budget now reports a limit, not configuration error.
- **Verification:** Go adapter race coverage 100%; markup fuzz 237,555 executions
  passes. Rust: 13 pass, 2 image-layout failures retained, all-target Clippy passes.
  Go task gate PASS 83s, changed coverage 93.1%, total 90.0%; USB/Wi-Fi builds
  pass without flashing. Pixel expectations were not changed to match the defect;
  the Go gate does not run or supersede the failing Rust suite.
- **Safety boundary:** the executable remains BZR1-only; new BZR2 image requests
  fail closed. Do not enable them until real renderer pixels and Go → binary
  integration pass. S5 remains the known-good visible result; no Pico touched.
- **Details/sources:**
  `experiments/03-remote-epaper/docs/manager-image-assets.md`;
  https://github.com/DioxusLabs/blitz/pull/605 and
  https://www.w3.org/TR/css-images-3/#the-object-fit.

## 2026-09-06 — approved Blitz beta.2 evaluation did not fix the pixel failures

- **Changed variable:** exact Blitz crates beta.1 → beta.2 with compatible
  AnyRender 0.13.0 / CPU adapter 0.17.0 and the Cargo-resolved rendering graph.
  User approved this dependency evaluation. No vendor patch, feature enabling,
  USB action or Pico change.
- **Measurement:** existing image tests reproduced both failures on beta.1.
  Beta.2 compiled in 1m35s and returns the same wrong packed rows: blank for
  intrinsic absolute 2×2; `[128,128,0,0,0,0,0,0]` for contain/clipping.
  Alpha/metadata tests pass. Assertions are unchanged.
- **Correction:** PR #605's presence was only a candidate, not proof of repair.
  Beta.2 does not resolve these two fixtures. Keep BZR2 disabled while localizing
  the actual sizing inputs; do not claim the upgrade fixed image support.
- **Migration guard:** beta.2 also auto-selects XML from root `xmlns`. A new
  failing Go test reproduced that unguarded input; reject root namespace
  declarations to retain the validated HTML parser mode.
- **Follow-up measurement:** temporary own-adapter tracing confirms CSS 4×4,
  no maximum, `item_is_replaced=true`, but unrounded layout 3×1.5. Absolute
  layout stays 0×0 at (2,1). Final cache entries have definite known dimensions;
  wrong first-pass values reaching the PR605 branch remain an inference.
  Trace code was removed; registry/vendor code was never changed.
- **Final checks:** full Rust suite 13 pass / 2 unchanged failures; Clippy and
  formatting pass. Go task gate PASS 49s, changed 93.1%, total 90.0%, adapter race
  coverage 100%. Existing USB/Wi-Fi builds pass without flashing. Real Go →
  BZR1 → beta.2 preview renders readable Ukrainian text in
  `blitz-probe/target/preview-beta2.png`; exact old/new pixel equivalence was not
  established. Image BZR2 activation and physical acceptance remain blocked.
- **Next candidate, unapplied:** remove only the two available-space fallback
  operations in beta.2 replaced-layout `max_size`, preserving explicit CSS
  min/max and ratio handling. No merged upstream correction was found in the
  bounded review. Require approval for a version-pinned dependency patch and
  additional min/max/zero-space regression tests before accepting it.

## 2026-09-06 — image bugs documented; requested test deferral not yet applied

- **User decision:** preserve the two known image-layout reproductions but stop
  routine execution, then continue independent work instead of patching Blitz.
- **Blocked action:** automated edit review rejected reasoned `#[ignore]` plus
  the exact `CONSTRAINTS.md` exception because the standing rule forbids skipped
  tests. No such change was applied; both original tests and assertions remain
  active. Do not retry through a different execution path.
- **Completed:** full synthetic reports `BLITZ-IMG-001/002`, expected and actual
  packed rows, exact dependencies, facts versus hypotheses, related upstream
  issues and rerun/reactivation conditions in
  `experiments/03-remote-epaper/docs/blitz-image-layout-bugs.md`.
- **New guard / measurement:** `blitz-probe/tests/image_activation.rs` sends a
  valid BZR2 request to the real executable: exit 1, exact VERSION diagnostic,
  empty stdout. Focused test PASS (1 test, no skips), format and locked offline
  all-target Clippy PASS. Independent review found no Required issue.
- **Scope:** no production code, dependency, Pico, wiring, USB operation or
  panel-policy changes. Known image tests were not rerun here; their previous
  failures remain open. No claim that the full suite is green.
- **Next:** obtain explicit confirmation for the bounded test exception;
  independent plan step 2 work is text/box viewport and CSS fixtures, followed
  by protected full-refresh timestamp composition. Image activation still needs
  corrected pixels and end-to-end tests; S5 remains the visible checkpoint.

## 2026-09-06 — user clarifies image support stays enabled despite two layout bugs

- **Clarification:** the user explicitly rejected keeping image support off:
  "ні, підтримка зображень має бути, але ці тести лише пропустити". This
  supersedes the earlier deferral condition, not the recorded bug evidence.
- **Applied scope:** exact `CONSTRAINTS.md` exception and reasoned ignores for
  `images_use_intrinsic_size_and_position` and
  `image_contain_and_clipping_use_final_layout` only. Original expected pixels
  are preserved; both bugs remain unresolved. Re-evaluate on the next renderer
  dependency/layout change; no other tests or limits may be suppressed.
- **Activation RED:** real executable test submits a valid BZR2 image request
  and expects exact BZM1 image pixels. It fails with exit 1 instead of 0 on the
  BZR1-only binary. Companion malformed-input and BZR1 compatibility cases pass.
- **Next implementation:** reuse the existing bounded BZR1/BZR2 decoder in the
  executable, then verify pixel output and Go handoff. No vendor patch, firmware,
  hardware update, new network fetcher or public renderer access.
- **Activation GREEN:** `main.rs` now uses the shared bounded `read_request`
  decoder and the existing renderer. All three real-executable tests pass:
  exact BZR2 image pixels, no frame on malformed BZR2 and BZR1 compatibility.
  The prior unconditional image-rejection test was replaced to match the user's
  clarified contract, not silently skipped.
- **Go handoff preview:** the first `testdata/blitz-image.html` run produced
  both scaled checkerboards through Go PNG decode → BZR2 → real Blitz, but its
  text was absent. The fixture had not selected the supplied Go Regular font;
  verify its font declaration against the existing readable BZR1 fixture before
  treating the complete scene as accepted. No physical update was attempted.
- **Font correction verified:** selecting `font-family:Go` in that fixture,
  matching the supplied font, produces readable Ukrainian text plus both
  128×128 checkerboards in `blitz-probe/target/preview-image.png`. This proves
  local Go → BZR2 → real Blitz handoff, not default font fallback or panel output.
- **Review correction:** the aggregate-limit test's second asset lacked RGBA
  data, so EOF could mask relaxed limit enforcement. Completed that pixel;
  the packet is now otherwise valid. No production limit change was needed.
- **Final checks:** full locked offline Rust suite 16 pass / exactly 2 named
  ignores / 0 fail after the review correction; format and all-target Clippy
  pass. Go task gate PASS 62s, changed coverage 93.1%, total 90.0%, zero lint
  issues; existing file-length debt remains visible. USB/Wi-Fi TinyGo builds
  pass without flashing. Known image bugs, broader CSS/public-input/physical
  acceptance stay open. Current plan and capability docs reflect image enabled,
  tests deferred; earlier blocked-activation entries remain historical evidence.

## 2026-09-06 — real-worker viewport and box capability fixtures

- **Scope:** manager-side pinned Blitz beta.2 → `render_mono`; no Pico, firmware,
  image-layout implementation, dependency or refresh-policy change.
- **Measurement:** first four active characterization tests pass: absolute
  bottom/right anchoring at 64×48, 296×128, 800×480 and odd 81×49 viewports;
  definite percentage boxes; margin/padding offsets; percentage width constrained
  by min/max. Full packed pixels, including row-padding bits, match the expected
  rectangles. These measure existing behavior, not a newly repaired defect.
- **Decision:** keep the fixtures as capability guards and extend them to
  inline alignment, clipping, relative flow and box sizing. No new test ignores;
  the two approved image-layout exceptions are unchanged.
- **Sources:** CSS 2.2 containing blocks, box model and min/max width rules;
  exact links are alongside the fixtures in `blitz-probe/tests/viewport.rs`.
- **Second measurement:** all nine fixtures pass after adding content-/border-box
  sizing, relative-position flow reservation, `display:none`, hidden versus
  visible overflow, and left/center/right adjacent inline-block alignment.
  Alignment uses font-free boxes; glyph readability still needs its own preview.
- **Preview measurement:** real Go → BZR1 → Blitz renders the new trusted
  `testdata/blitz-viewport.html` into `target/preview-viewport.png`. Visual check:
  readable Ukrainian including Ґ/Є/І/Ї, left/center/right text and bars, borders,
  percentage-sized content and a bottom-right local-preview label. No USB.
- **Review:** no Required finding. Addressed two optional precision points:
  narrowed the overflow test name to the clipping it actually proves, and added
  a smaller positioned-parent percentage fixture to distinguish parent from
  viewport resolution. No engine behavior, resource limits or exceptions changed.
- **Final verification:** ten viewport tests pass; full locked offline Rust
  suite 26 pass / exactly two approved image ignores / 0 fail. Format check and
  all-target Clippy pass. Commands ran only in existing container `1611f88ac6e1`.
  Go/firmware sources did not change; no new full Go gate or physical acceptance
  is claimed for this test-only increment. Current scope and remaining gaps are
  recorded in `docs/blitz-viewport-css.md` within the experiment.

## 2026-09-06 — isolated confirmed full-refresh timestamp

- **Scope:** manager-side `refreshstamp`, no live EPS1/Pico/panel change.
- **Boundary found:** current sender returns only terminal success/error; it
  does not report physical refresh start/completion instants. Do not fabricate
  those from HTML arrival or renderer timing. Reserved-region layout diagnostics
  are also still open; a pixel guard cannot detect white-on-white occupancy.
- **Decision:** first implement independent label/confirmation state and bounded
  corner composition, with explicit trusted instants and provisioned timezone
  supplied by the caller. Keep live integration and maintenance scheduling separate.
- **RED:** new state tests fail to compile because `Tracker`, `New` and typed
  cycle/time errors do not exist yet. They require candidate != confirmed,
  matching completion, partial preservation and full resync after failure.
- **State GREEN:** matching full-cycle confirmation and abort/history tests pass.
- **Composition RED:** corner/ownership tests fail because `Bounds`, `Paint` and
  `ErrOccupied` are not implemented. They require validation before mutation and
  preservation of outside logical pixels plus odd-width/stride padding bits.
- **Component GREEN:** race-enabled Go tests pass with 100% statement coverage.
  Independent font-drawer pixels match the badge; stale completion, full resync,
  repeated local DST hour and invalid clock/ID cases pass. `Paint` measures zero
  allocations in the focused test, not across the complete renderer pipeline.
- **Next check:** opt-in synthetic stamp in the local PNG preview, preserving
  existing three-argument usage and never confirming a physical cycle.
- **Preview RED:** valid five-argument invocation fails with the old usage
  error. Added error-path guards that preserve an existing output artifact.
- **Preview GREEN:** focused race-enabled component/CLI tests pass, including
  invalid timestamps/timezones and occupied-corner failures without truncating
  existing output. The opt-in preview stages a candidate but never confirms it.
- **Real preview:** Go → BZR2 → locked Blitz → stamp → PNG completes with the
  existing image fixture. Visual inspection confirms both images, Ukrainian
  text and the synthetic `2026-09-06 15:34` label in its 120×21 bottom-right
  rectangle. No USB, firmware flash, device acknowledgement or physical refresh.
- **Quality correction:** first task gate stopped on complexity limits in three
  new test functions (11/13/13, maximum 10). Split independent configuration,
  candidate, completion and backward-clock scenarios into focused tests; retained
  assertions and existing lint thresholds. No production behavior changed.
- **Final verification:** task gate PASS 17s, zero lint issues, changed coverage
  93.5%, total 90.1%; `refreshstamp` remains 100%. USB build flash/RAM
  237724/105564 bytes; Wi-Fi 699804/112156 bytes, unchanged from the previous
  checkpoint. Six existing file-length debts remain reported. Independent
  read-only component/preview review found no required or remaining optional
  finding. No firmware flashed; real timestamp integration, layout occupancy
  diagnostics and panel acceptance are still pending. Image support and the
  two approved skipped reproduction tests are untouched.

## 2026-09-06 — real-worker text and CSS capability guards

- **Scope:** pinned Blitz beta.2 manager rendering; no firmware, panel timing,
  transport, dependencies or image exceptions changed.
- **Measurement:** four active tests pass through `render_mono` with the exact
  Go Regular font from the existing locked `x/image` dependency. Normal-space
  collapsing, automatic word wrapping, nowrap overflow and preserved preformatted
  line breaks match their explicit reference renderings. Non-empty row guards
  prevent an absent font from making equal blank frames appear successful.
- **Decision:** these characterize current output, not newly implemented CSS.
  Extend to clipping and remaining text/cascade boundaries. No font vendoring
  or new download: the test resolves the cached Go module with networking off.
- **Sources:** CSS Text white-space and wrapping, CSS Overflow text-overflow,
  CSS Cascade sorting; stable official links accompany tests and the checkpoint.
- **Second measurement:** pre-wrap/forced-break equivalence, exact horizontal
  and vertical glyph clipping and inherited text style fixtures pass. Clipping
  is checked against an independently masked visible frame, including partial
  bytes and mid-glyph rows, not just an absence of outside pixels.
- **Review correction:** the preformatted whitespace test mixed spaces with a
  newline, so newline preservation could mask collapsed spaces. Add a separate
  same-line double-space oracle; keep the existing newline assertion intact.
- **Third measurement:** nine text tests pass, including the corrected space
  oracle, long-word `anywhere`/`break-word` behavior at a fixed width and glyph
  alignment/integer translation. This does not prove min-content sizing modes.
- **Source finding:** locked Blitz maps `pre-line`/`break-spaces` incompletely;
  no ellipsis paint implementation was found. Keep those capabilities unaccepted
  and inspect a real local text preview before drawing further conclusions.
- **Cascade measurement:** four fixtures pass for selector specificity,
  compound/descendant selectors, source order and important versus normal inline
  precedence. A raw-worker test confirms invalid CSS is silently discarded;
  successful rendering must not stand in for the product's typed profile gate.
- **Initial text preview:** Go → real Blitz shows readable Ukrainian, wrapping,
  clipping, pre-wrap, emergency long-word breaks and white text on black. The
  `text-overflow:ellipsis` card looks clipped just like the control, with no
  ellipsis. The absolute bottom footer is absent; this fixture omitted the
  definite root/body height used by prior accepted viewport fixtures. Check
  that single fixture difference next; do not claim general auto-height support.
- **Viewport correction verified:** adding only explicit 100% root/body width
  and height restores the bottom footer, consistent with earlier viewport
  fixtures. The ellipsis card still matches ordinary clipping visually; it is
  an unresolved capability gap, not a completed feature or a new test ignore.
- **Review:** repeated-space correction accepted; strengthened the optional
  glyph-alignment guard with left < center < right ink bounds, retaining pixel
  translation assertions. No production implementation was changed.
- **Final Rust verification:** 39 pass / exactly two existing image ignores /
  zero fail; all-target Clippy with warnings denied and format check pass after
  the alignment correction. Thirteen new tests are active. Text/cascade scope,
  sources and unresolved capabilities are recorded in `docs/blitz-text-css.md`
  within the experiment; the next implementation is typed CSS profile validation.
- **Wider verification:** Go task gate PASS 16s, zero lint issues, changed
  coverage 93.5%, total 90.1%. USB/Wi-Fi TinyGo builds pass with unchanged size;
  six existing file-length debts are still reported, not suppressed. No commit,
  push, firmware flash or physical acceptance in this increment.

## 2026-09-06 — strict manager CSS declaration boundary

- **Scope:** isolated manager grammar validation, not a new Pico renderer or
  public HTML gate. Raw Blitz/image rendering remains unchanged.
- **Fact:** locked Blitz beta.2 has no configurable CSS error reporter. Stylo
  suppresses some invalid duplicate/vendor-prefixed declarations; cssparser
  recovers unclosed blocks, strings and comments at EOF. Reporter silence or a
  successful render therefore cannot prove strict input acceptance.
- **Decision:** reuse locked cssparser/Stylo declaration parsing with a bounded
  structural preflight, fresh state per declaration and source-free typed byte
  ranges. Keep native capability validation separate. Contract and source links:
  `experiments/03-remote-epaper/docs/css-declaration-validation.md`.
- **RED:** new integration tests fail to compile because `csscheck` does not
  exist. They require valid CSS grammar, invalid duplicate rejection, strict
  malformed-input handling and UTF-8 byte diagnostics. No existing tests removed.
- **First GREEN:** four new Rust tests pass. Real Stylo validates declarations
  and rejects unknown properties/invalid duplicate values; the bounded token
  walk rejects EOF recovery and unmatched delimiters. Error ranges retain
  non-ASCII byte accuracy without logging source content. Exact already-locked
  cssparser/Stylo/traits/URL packages are exposed directly; no renderer upgrade.
- **Next verification:** exact byte/token/depth/declaration boundaries and a
  truncation corpus, then independent review and existing renderer regressions.
- **Boundary verification:** all eleven focused tests pass, including inclusive
  32 KiB / 4096 token / 16 depth / 256 declaration limits, non-finite numeric
  tokens, escaped delimiters, CRLF string continuations and deterministic UTF-8
  truncations. This corpus is not exhaustive fuzzing. No test skips added.
- **Review / regression RED:** the first full Rust suite passes (50 tests, two
  existing image ignores), but independent review found a missed EOF escape:
  an identifier ending in one backslash becomes a replacement character and can
  still pass Stylo. A new focused test reproduces this. Review also found that
  nested closing delimiters were not counted toward the token budget; add exact
  nested-boundary guards before final acceptance. Existing green tests were
  insufficient for these two cases, not evidence to waive them.
- **Corrections GREEN:** both review findings reproduced independently with
  failing tests. All thirteen CSS tests now pass after rejecting odd terminal
  escapes and counting actual nested closers. Doubled-backslash controls stay
  valid; the over-budget closing delimiter has an exact diagnostic span.
- **Final verification:** independent re-review approves both fixes with no
  remaining required/optional finding. Full Rust suite: 52 pass, exactly two
  approved image ignores, zero failures; fmt and all-target Clippy with warnings
  denied pass. Cargo still has 209 dependency packages plus this root package.
- **Wider verification:** Go task gate PASS 18s, zero lint issues, changed Go
  coverage 93.5%, total 90.1%. These percentages do not measure Rust coverage.
  USB flash/RAM 237724/105564 bytes; Wi-Fi 699804/112156, unchanged. Six existing
  file-length debts remain reported. No Pico flash/update, commit or push.
- **Remaining boundary:** this grammar module is not called by the raw worker.
  Implement native value acceptance, stylesheet/HTML discovery, aggregate
  budgets and typed worker errors before calling it a public-input gate. Image
  support and the S5 physical checkpoint remain unchanged.

## 2026-09-06 — CSS/HTML validation integration

- **Scope:** continue server-side validation through real HTML/worker errors;
  preserve image delivery, S5 firmware and the two approved image-test ignores.
- **Fact:** locked Blitz decodes HTML entities again in `<style>` text and starts
  parsing CSS during DOM construction. A preflight must match its effective CSS,
  not blindly treat a separate HTML parser's output as identical. Native profile
  acceptance and this beta behavior must stay explicit.
- **Stylesheet RED:** four new tests fail because `validate_stylesheet` is not
  implemented. They require selector/value grammar, strict at-rule/nesting
  rejection, whole-source offsets and a shared declaration budget.
- **Stylesheet GREEN:** all 17 CSS tests pass. Qualified rules reuse Stylo's
  selector parser, initial `@charset` cannot bypass explicit rejection, and
  declaration counts are shared across rules. Review identified forgiving
  selector branches as a separate remaining guard to add before activation.
- **HTML discovery RED:** three Go tests fail to compile because style source
  collection is absent. The adapter will use its existing parsed/serialized tree,
  assign document-order element ordinals and reject ambiguous style contexts
  before PNG decoding. No additional DOM parser or renderer process is needed.
- **HTML discovery GREEN:** three focused Go tests pass for the canonical tree,
  decoded inline attributes, incompatible style contexts and the 256-source
  limit. `<style>` containing `&` is deliberately rejected until its semantics
  can be normalized safely; no source is silently changed to appease Blitz.
- **Selector RED:** a new real-Stylo test reproduces invalid branches accepted
  inside forgiving `:is`/`:where` lists. Review also found `:has` requires an
  explicit recursive visitor and `&` can parse outside nesting. Reject recovered
  and parent components using the existing typed selector AST.
- **Selector follow-up:** invalid-branch cases now reject, but one nested valid
  selector control still fails. Investigate the locked engine rather than drop
  the positive assertion. The public worker remains unchanged meanwhile.
- **Document-budget RED:** two new tests fail to compile for the missing shared
  `Validator`; per-fragment limits must not reset for each inline/style source.
- **Document-budget GREEN:** both aggregate-budget tests pass. The selector
  control failure is now explained by pinned Stylo's explicit `parse_has=false`,
  not a validator regression. The originally assumed positive `:has` assertion
  is corrected to an active unsupported-engine rejection; nested positive
  `:not(:is(...))` remains. No skips or vendor changes. This capability was not
  part of the accepted basic selector profile.
- **Boundary verification:** all 21 CSS tests pass with the corrected pinned
  selector capability. Go aggregate-byte overflow reproduced with a failing
  boundary test before restoring its guard.
- **Checked IPC RED:** four BZR3/BZE1 tests fail to compile because `read_job`
  does not exist. The bounded local contract is recorded in
  `docs/blitz-checked-ipc.md`; no Pico wire or hardware change is involved.
- **Checked Rust GREEN / Go RED:** four real parser/wire tests pass, including
  truncations, metadata, source-free offsets and aggregate CSS bounds. Five new
  Go integration tests fail for the missing shared `renderdiag` package; typed
  validation failures must remain distinct from worker/runtime failures.
- **Process verification:** real Rust subprocess returns exact BZE1/exit 0 for
  bad CSS with a deliberately invalid font, proving rejection precedes rendering.
  Valid checked CSS plus an image still produces the expected BZM1 pixels. Go
  adapter tests pass for BZR3 and malformed/typed replies. Independent Rust review
  found no required defect; its requested process coverage is now present.
- **Manager recovery RED:** three new tests fail for the missing failure status.
  Status must answer which revision failed, its source-free location, and which
  revision remains confirmed. A rejected scene must not retry or kill the pump;
  transport/runtime failures must retain their existing stop/resync behavior.
- **Recovery GREEN / gate:** focused Go tests pass, including stale diagnostic
  rejection, copied status, no retry and corrected-scene delivery. Full Rust:
  66 passes, exactly two existing image ignores. Go task gate stops at two new
  test complexity violations; split test phases without removing assertions.
- **Go compatibility RED:** full gate then caught existing exact `ErrInput`
  assertions in the private encoder. Keep that contract and attach the new
  recoverable document classification at `Worker.Render`, the manager boundary.
  Invalid administrator viewport configuration still returns its original error.
- **Real adapter preview:** Go → BZR3 → Blitz renders `blitz-image.html` with
  both checkerboards and readable Ukrainian text, visually inspected in
  `blitz-probe/target/checked-image.png`. No USB operation or panel claim.
- **Review RED:** although the Go task gate passed in 20s (93.5% changed,
  90.2% total coverage), independent review found that a concurrent new submit
  could hide a fatal renderer failure behind `ErrSuperseded`. A new combined
  test reproduced a second renderer call after the first crash. Keep fatal
  renderer and frame-contract failures fatal even when their scene is stale.
- **Race recovery GREEN:** new fatal/stale and invalid-frame/stale regressions
  pass. Focused manager/adapter/diagnostic race suites pass. Recoverable stale
  CSS errors remain discardable; invalid frames are checked before supersession.
- **Reviewed checkpoint:** Go task gate PASS 19s after the state-transition
  helper split; zero new lint issues, 93.5% changed coverage, 90.2% total. USB
  flash/RAM 237724/105564 bytes and Wi-Fi 699804/112156 remain unchanged. Final
  independent Go re-review reports no required code finding. Image-enabled ADR
  correction and required inline-block entry are now recorded.
- **New renderer acceptance RED:** official Blitz status declares static
  containing-block semantics unsupported. A new active exact-pixel test now
  reproduces this on pinned beta.2: a 2×2 absolute child inside a static wrapper
  lands at (8,6), but CSS requires (20,12), relative to the nearest positioned
  ancestor. This is not a Pico, protocol, CSS-validation or clipping failure.
  The earlier ten viewport tests did not contain an intervening static wrapper.
  No ignore or expectation weakening was added; Rust acceptance is now RED.
  Resolve renderer strategy explicitly before claiming the required profile.
- **Independent confirmation:** review validates both the spec and rectangle
  oracle (20,12 versus measured 8,6), without image/font/AA ambiguity. Official
  status describes the same limitation; related closed upstream #690 is not
  proof of this nested case being fixed. Detailed reproduction, remaining text
  gaps and the required scope decision are in `docs/blitz-positioning-gap.md`.
  Following stop-the-line, do not add more features over this failing gate.
- **Final verification at pause:** full Rust `--no-fail-fast` completes with
  66 passes, exactly one failure (the new static-wrapper test), and the same
  two approved image ignores. Formatting passes; the last all-target Clippy
  passes. Go task gate and the final focused race rerun pass. No hardware or
  dependency mutation, commit or push. Request a renderer-remediation scope
  decision rather than silently waive another required CSS behavior.

### 2026-09-06 — approved engine-bug deferral, BLITZ-POS-001

- **Decision:** the user requested documenting and skipping confirmed engine
  defects without fixing the engine. This supersedes the preceding pause.
- **Reconfirmed before change:** the isolated static-wrapper test still fails
  on pinned Blitz beta.2: actual (8,6), required (20,12). Official Blitz status
  also documents immediate-parent absolute positioning; this is not a Pico fault.
- **Changed variable:** add a reasoned ignore only to
  `static_ancestor_does_not_capture_absolute_containing_block`. Its body and
  expected pixels remain intact. Root `CONSTRAINTS.md` records approval, owner,
  expiry and re-enable procedure; `docs/blitz-positioning-gap.md` holds the report.
- **Boundaries:** no engine patch, dependency upgrade, runtime workaround,
  firmware flash or USB operation. Two existing image exceptions are unchanged;
  other tests remain active. Full CSS conformance is still not accepted.
- **Verification:** full default Rust suite: 66 pass, zero fail, exactly three
  approved ignores. Explicit `--ignored --exact` still reproduces BLITZ-POS-001;
  it was not fixed. Formatting and all-target Clippy pass. Go task gate PASS 17s,
  zero new lint, 93.5% changed/90.2% total coverage, same firmware sizes and six
  reported file-length debts. Only the skip annotation and documentation changed.

### 2026-09-06 — isolated native CSS declaration policy

- **Scope:** server-side validator only; keep BZR3 grammar behavior, renderer
  pin, three approved engine-test ignores and visible S5 checkpoint unchanged.
- **Decision:** reuse Stylo's parsed property IDs and expanded declarations,
  with a separate native mode and specified-value limits. This is a core policy
  increment, not full HTML/fallback/computed-layout acceptance or a public sandbox.
- **RED:** five new tests fail to compile because `Validator::native()` and
  native rejection codes do not exist yet. They cover ordinary dashboard CSS,
  out-of-profile declarations, duplicate overrides and source-free UTF-8 spans.
- **Boundary:** broad resetting shorthands and font/URL assets need subsequent
  policies. Do not switch the working manager to this incomplete core profile.
- **Parser observation:** the existing grammar-only validator already rejects
  `display:grid` as `InvalidValue` on this pin. A new test had incorrectly
  assumed it would reach native rejection. Preserve grammar-first behavior and
  test that exact refusal separately; do not change the engine or suppress it.
- **Bounds RED:** three new resource/abuse groups pass; the specified-limit
  test exposes acceptance beyond a native numeric bound. Percentage opacity
  must use its own normalized 0..1 range, not the general percentage cap.
- **Core GREEN:** all nine native property/value/resource tests pass after the
  opacity correction. Grammar mode retains its prior behavior. Expanded
  longhands are checked individually; defaults inherited from broad shorthands
  cannot enter through an unlisted source property.
- **Runtime / review:** exact real Blitz output after native validation and
  UTF-8 truncation cases pass. An explicit checked-wire test proves BZR3 remains
  grammar-only. Full Rust suite passes with 78 active tests and the same three
  approved engine ignores. Independent read-only review finds no Required issue;
  add alias/HSL controls and document unitless HSL hue. Native diagnostics are
  kept in their owning module to avoid extending the grammar callback further.
- **Final gates:** alias/HSL controls pass; Rust formatting and all-target
  Clippy deny-warnings pass. Go task gate PASS 16s, zero new lint,
  93.5% changed/90.2% total Go coverage, unchanged USB/Wi-Fi firmware sizes.
  Six existing file-length debts remain visible. Contract and remaining
  integration work are in `docs/native-css-declarations.md` within the
  experiment. No additional ignores, engine patch, flash, USB activity or
  hardware acceptance; BZR3 still uses grammar-only validation.

### 2026-09-06 — native selector admission and fallback boundary evidence

- **Scope / decision:** extend only the opt-in native CSS validator with the
  required tag/class/ID/compound/descendant selector profile. Keep BZR3 grammar
  mode, dependencies, firmware, three approved ignores and S5 unchanged.
- **Sources:** exact locked selectors 0.40.0 AST/iteration and Stylo 0.20.0
  parser; W3C Selectors 4 structure and WHATWG style-element semantics.
- **RED:** five new selector policy/diagnostic/budget tests fail to compile
  because `UnsupportedSelector` does not exist. The tests distinguish grammar
  errors from valid CSS outside the native capability profile.
- **Boundary:** a bitmap marker is not stylesheet scope or permission for
  active content. Actual target matching and inherited/composited effects must
  be checked before relaxing property policy inside any fallback area.
- **Policy RED / cause:** four tests pass; the negative matrix exposes `*|div`
  being admitted. Pinned selectors intentionally drops a redundant any-namespace
  prefix when no default namespace exists. This is a normal parser optimization,
  not an engine bug or reason to skip the test. Strict AST grammar validation
  needs a source-token capability gate to preserve the requested syntax boundary.
- **Test-oracle correction:** the expanded matrix reaches `:nth-child(... of
  .card)`, `:has(.card)` and `p::first-line`, which this standalone grammar already
  rejects. Keep all exact cases active under grammar-first `InvalidSelector`, not native
  `UnsupportedSelector`.
  No parser settings, required semantics or skip annotations change.
- **Selector GREEN / review:** all five new tests pass. Independent read-only
  review finds no Required production issue; add escaped literal `|`/`*`
  controls. The token gate adds no dependency or independent selector parser.
- **Runtime GREEN:** real Blitz preserves expected specificity/source order and
  escaped-name matching after native admission. A style inside a marked bitmap
  subtree changes an outside box from 4px to 10px: the source-parent shortcut
  would be incorrect. Focused selector/render/BZR3 tests pass; no native activation
  or fallback implementation is inferred. Contract and pending ownership gates
  are recorded in the experiment's `docs/native-css-selectors.md`.
- **Final gates:** full Rust suite passes with 86 active tests and the same
  three approved ignores. Formatting, all-target Clippy deny-warnings and diff
  whitespace checks pass. Go task gate PASS 18s, zero new lint issues,
  93.5% changed/90.2% total Go coverage; USB/Wi-Fi firmware sizes unchanged.
  Six pre-existing file-length debts remain visible. No flash, USB transfer,
  dependency update, commit, push or new physical acceptance.

### 2026-09-06 — CSS targets and actual DOM bitmap ownership

- **Scope:** isolated manager-side ownership analysis, not CSS waivers, subtree
  extraction, BZR3 activation or firmware changes. Use the already parsed Blitz
  DOM and its selector matcher; never infer ownership from HTML source nesting.
- **Sources:** locked Blitz 0.3.0-beta.2 `query_selector.rs`, `node/node.rs`,
  `node/element.rs` and Stylo 0.20.0 author-origin selector parser. Actual DOM
  children and outermost bitmap ancestor govern ownership; layout children do not.
- **RED:** seven new real-DOM tests fail with unresolved `domscope` import.
  They cover cross-owner selectors, nested markers, parser repair, global style
  effects, escaped names, invalid markers/foreign trees and fail-closed reuse.
- **Limits/remaining gates:** bounded node/depth/query/target inventory. This
  accepts a previously parsed local document; it is not an untrusted HTML entry
  point, a CPU guarantee, or authorization for unmatched selectors/fallback CSS.
- **First GREEN:** all seven actual-tree tests pass using Blitz's existing
  `Node::matches_selector_raw`, parsing once per query. No engine patch or ignore.
  Add exact budget boundaries, duplicate-ID/new-scene and source-free errors next.
- **Defensive-tree RED / review:** a new test shows an invalid root parent was
  accepted. Independent review identifies Blitz `TNode::owner_doc` walking parent
  links without a cycle guard; do not invoke matching on that corrupted tree.
  Validate the document root, traversal IDs and every parent/child link first.
  This is our preflight guard, not an engine patch or a disabled reproduction.
- **Additional oracle correction:** optional no-doctype test expected browser
  quirks matching (`.card` matching `class="Card"`) but got zero targets. Exact
  source shows Blitz `TDocument::quirks_mode` always returns `NoQuirks`, even
  though the HTML sink records parser mode separately. The ownership contract
  requires actual renderer consistency, not browser quirks conformance. Keep
  both source variants active, pin case-sensitive results and compare with
  Blitz's own query APIs. Do not patch the engine, add an ignore or claim browser
  compatibility. The initial mistaken expectation is retained in this record.
- **Final GREEN / review:** all 103 active Rust tests pass, zero failures,
  exactly the same three approved ignores. Seventeen new tests exercise this
  increment. Independent final review accepts the root guard and source-backed
  oracle correction with no remaining Required findings. Formatting, all-target
  Clippy deny-warnings and diff whitespace checks pass.
- **Repository gates:** Go task PASS 16s, zero new lint, changed/total Go coverage
  93.5%/90.2%. USB flash/RAM 237724/105564; Wi-Fi 699804/112156, unchanged.
  Six untouched legacy file-length debts remain reported. No engine/dependency
  patch, new ignore, firmware flash, USB transfer, commit or push. S5 remains
  the last physical acceptance; rule/declaration integration and actual bitmap
  extraction are still open in `docs/css-target-ownership.md` in the experiment.

### 2026-09-06 — bind CSS declarations to actual scene targets

- **Scope:** opt-in native-core rule/inline binding on the same Blitz DOM.
  Preserve declaration spans and share targets per rule, without cascade
  reconstruction, fallback property waivers, BZR3 activation or firmware changes.
- **Sources:** locked cssparser 0.37.0 `DeclarationParser`, `QualifiedRuleParser`
  and Stylo 0.20.0 declaration parsing. Official docs.rs retrieval failed;
  exact installed upstream trait documentation and source were inspected.
- **RED:** four real-DOM binding tests fail because `StyleKind` and `styles()`
  are absent. They cover mixed native/bitmap rule targets, inline ownership,
  unchanged native validation, source-local diagnostics and source-free errors.
- **Decision:** retain original specified declaration byte spans before cascade
  discards overridden entries. Default validation must not allocate metadata
  vectors; only the opt-in analysis collects them.
- **Parser compatibility GREEN:** after adding span collection, existing strict
  grammar/native/checked-wire suites pass. Collection reuses the original parser
  callbacks; normal validation discards metadata without vector allocations.
- **Binding GREEN:** all four first tests pass. Rule targets are independent of
  stylesheet parent; inline targets preserve actual bitmap ownership. Invalid
  later declarations identify their DOM source and close the whole analysis.
  Next verify exact source/rule/query/target/CSS budget boundaries and source context.
- **Limits GREEN / review RED:** all seven initial limit tests pass. Review
  finds source-count checks happened after copying a source (total CSS bytes
  remained bounded, but the stated pre-copy count guard was inaccurate). A new
  test proves the 257th stylesheet reached context parsing before SourceLimit.
  Check source slots before inspecting/copying sheet or inline CSS; retain
  cumulative byte checks before appending text. No bound or assertion is weakened.
- **Source-slot GREEN:** the precedence test passes after the pre-copy check;
  independent review confirms the fix. Total eight source/limit tests now exist.
- **Fixture corrections:** the new span fixture first expected a trimmed
  selector; the existing cssparser prelude range includes its trailing space.
  Pin the exact original bytes instead. The paint fixture first used the broad
  `background` shorthand, which the documented native core rejects. Use its
  supported `background-color` longhand for this nonmutation test; do not relax
  the property gate. Neither correction changes a required renderer assertion
  or one of the three approved engine reproductions.
- **Final GREEN / Measurement:** all 117 active Rust tests pass, zero failures
  and exactly the same three approved engine ignores. Fourteen added tests
  include same-document resolve/paint equality plus independent exact RGBA
  expectations. Formatting and all-target Clippy deny-warnings pass. Go task
  PASS 19s, zero new lint, changed/total Go coverage 93.5%/90.2%. USB flash/RAM
  237724/105564; Wi-Fi 699804/112156, unchanged. Six existing Go file-length
  debts remain reported. Tracked diff and explicit untracked whitespace checks pass.
- **Review / Decision:** independent review accepts the source-slot fix,
  source/target boundaries and final tests; its remaining stale README summary
  is corrected. Simplification review found no useful further refactor. New
  contract `experiments/03-remote-epaper/docs/css-style-bindings.md` records
  source order, budgets, error precedence, evidence and pending integration.
- **Unknown / next gate:** local DOM identities are not yet checked against
  Go/BZR3 source metadata. Native activation, fallback property authorization,
  computed bounds and composited pixel extraction remain open. No engine patch,
  dependency change, new ignore, flash, USB transfer, commit or push. Physical
  checkpoint remains S5; these manager tests are not new panel acceptance.

### 2026-09-06 — checked Go/Blitz CSS source identity

- **Fact / scope:** local declaration bindings pass, but BZR3 still trusts the
  association between manifest CSS and HTML. It does not send the Go element
  ordinal. Connect these boundaries without enabling incomplete native policy.
- **Decision:** local BZR4 adds the source ordinal and supports an explicit
  empty manifest; verify count/kind/ordinal/text against the same Blitz document
  before layout/paint. No duplicate-content search, second rendered document or
  protocol downgrade. An identity mismatch fails without a frame; no physical
  update or changed device protocol. Contract: experiment
  `docs/css-source-identity.md`. Tests/review are required before activation.
- **Identity RED→GREEN:** three real-DOM tests initially fail on the absent
  `SourceIdentity`/`verify_sources` API, then pass with bounded source comparison.
  Missing/extra/reordered/kind/text/ordinal mismatches close analysis, identical
  CSS on different elements stays distinct, and HTML-decoded inline CSS matches.
  Existing 13 binding/limit tests also pass. Source identity does not relax or
  activate native property policy. Next add the versioned wire boundary.
- **Source evidence:** current official x/net v0.58.0 security documentation
  requires reserialization after parsed-content trust decisions. Installed
  Blitz beta.2 HTML sink uses scripting=false/drop_doctype and element-tree
  normalization; its document module additionally decodes style RAWTEXT.
  Official docs.rs BaseDocument retrieval failed; exact installed source read.
- **Review RED→GREEN / refinement:** passing identity through `SceneScope::new`
  wrongly rejects unknown bitmap markers even when CSS matches. A real-DOM
  reproduction returns InvalidMarker. Source identity now uses a stateless
  verifier, shared structural traversal/source collection and marker-ignore
  policy, not a native scope or unused matching counters. Five identity tests
  and 22 related ownership/binding tests pass. Strict native marker validation
  remains unchanged. Identity returns no unused ownership/location vector.
- **Wire RED→GREEN:** three BZR4 reader tests first return VERSION, then pass
  after adding the explicit version and bounded ordinal field. Six existing
  BZR3 wire tests still pass. Empty BZR4 manifests remain distinguishable from
  raw probes. Next require identity in the real worker before any frame output;
  decoding alone is not acceptance.
- **Worker RED→GREEN:** the mismatch process fixture first returns a frame for
  an omitted manifest. The checked render path now requires a validated-job
  borrow, verifies identity on the actual document, then injects images and
  resolves/paints it. Five BZR4 process tests pass: mismatch refusal, explicit
  empty manifest, grammar-before-font rejection, broad grammar and exact image
  output. Existing BZR3 process/wire and composition tests also pass. Next switch
  Go emission to BZR4, retaining raw probes only in Rust compatibility tests.
- **Go RED→GREEN:** image/CSS/text wire tests first fail on old magic/counts,
  then pass with BZR4 emitted for every prepared scene. Ordinals accompany exact
  CSS bytes; an explicit empty manifest accompanies image-only/text-only scenes.
  Focused Go worker/manager/diagnostic suites pass. Next exercise actual Go →
  rebuilt worker output and parser-normalized source identity.

- **Real integration / measurement:** production Go `blitz-preview` with BZR4
  succeeds for `testdata/blitz-image.html` and new `blitz-identity.html` using the
  rebuilt worker. Both PNGs were visually inspected: Ukrainian labels and image
  content appear; the repaired-table text appears outside its table. Four Go
  corpus cases pin manifest stability after serialization (implied paragraphs,
  foster parenting, adoption-agency repair, scripting-disabled noscript).
  Exact actual-DOM 256-source acceptance / 257-source refusal also passes.
  These checks prove CSS identity, not general typography/layout equivalence.
- **Review / measurement:** independent BZR4 review found no required defect.
  Rust full suite now passes 131 active tests, zero failures and the same three
  approved engine ignores; all-target Clippy deny-warnings passes. No vendor
  patch, extra skip or firmware/hardware change. Go full gates follow.

- **Go quality RED / simplification:** the new ordinal assertion brought the
  changed wire test to cyclomatic complexity 11 (limit 10). Keep every field
  check as one exact expected-record comparison with independent ordinals 3/5,
  instead of separate branches tied to encoder metadata. No lint suppression
  or removed assertion. Repeat task/race gates after this test-only refinement.

- **Final BZR4 gates / measurement:** task PASS 16s, no new lint issues,
  changed Go coverage 93.5%, total 90.2%; focused worker/manager/diagnostic race
  tests pass. USB flash/RAM 237724/105564 and Wi-Fi 699804/112156 are unchanged.
  Rust 131 active passes, 3 approved ignores, formatting/Clippy pass. The
  checked IPC/plan/profile docs now distinguish implemented identity from
  pending native/fallback and public-input policy. No flash, USB transfer,
  dependency upgrade, commit or push; last physical checkpoint remains S5.

### 2026-09-06 — active HTML rejection before worker launch

- **Fact / scope:** ADR-009 forbids scripts, event handlers, forms and active
  content in native and bitmap paths. Go currently bounds HTML/images but does
  not enforce this rule on ordinary elements. Add a parsed-tree gate before
  image decoding or worker startup; do not claim a full HTML sanitizer, URL/CSS
  capability policy or OS sandbox. No engine or firmware changes.
- **Source / decision:** WHATWG event-handler content attributes and meta pragma
  directives describe the active surfaces. Inspect canonical parsed elements,
  not substrings: comments, escaped examples and data-* metadata stay inert.
  Reject on* attributes conservatively, even if empty or unknown to this pin.
  Bitmap markers never waive this global prohibition. Existing source-free
  InvalidDocument diagnostics preserve manager recovery semantics.

- **Active-content RED→GREEN:** negative fixtures initially show accepted script,
  event, form, embed/media and meta-pragmas; the missing-worker probe reports a
  process failure, proving startup was attempted. `validateStaticElement` now
  rejects during the existing bounded traversal, before asset decoding. Focused
  worker/manager suites pass. Static comments, escaped code, data-* attributes,
  metadata and scripting-disabled noscript text remain accepted. Add exact empty
  attribute and before-PNG tests; independent review and complete Go gate follow.

- **Active gate lint RED:** adding the independent policy check made existing
  image traversal complexity 11. Separate single-image admission from whole-tree
  traversal; retain all source/count/namespace/depth assertions. No suppression,
  budget change or extra traversal; repeat complete gates after this refinement.

- **Review RED / correction:** `<details>/<summary>` escaped the initial tag
  gate. Exact pinned Blitz pointer handling toggles the `open` DOM attribute;
  review identified this as policy mismatch, not demonstrated execution in our
  one-shot worker. New closed/open details and dialog fixtures reproduce
  acceptance. Reject these interactive containers as well, without special
  bitmap permissions. Previous 17s Go gate passed before these additional cases;
  final acceptance requires rerunning after the correction.

- **Final active gate / measurement:** corrected details/dialog cases pass;
  task PASS 17s, no new lint, 93.6% changed / 90.2% total Go coverage; gate
  statements 23/23 covered and focused race tests pass. No dependency, engine
  patch, additional ignore or firmware change. Full profile/OS isolation remain.
- **S6 preparation:** added a synthetic two-image HTML acceptance fixture.
  Preview rendering succeeded but writing first failed because its new output
  directory did not exist. Create only the ignored S6 artifact directory, then
  repeat preview/build; no hardware action occurred. Preserve S5 artifacts.

- **S6 visual inspection:** text and two images appear, but the corner badge is
  absent with an auto-height static body. The accepted S5 fixture explicitly
  sized and positioned its body; BLITZ-POS-001 documents immediate-parent
  positioning. Inference: this preview is consistent with that known boundary,
  not new panel evidence. Author S6 with an explicit full-viewport positioned
  body, retaining intended corner text. No renderer rewrite, engine change,
  changed regression assertion or extra skip; general CSS acceptance is unchanged.

- **Final review / S6 readiness:** reviewer confirms the details/summary/dialog
  finding closed with no required remaining finding. Updated S6 preview visibly
  contains heading, both PNG copies and readable corner label. macOS manager
  and USB binaries cross-build in-container; known-good UF2 hash is unchanged.
  Tracked diff and explicit untracked-source whitespace checks pass. Artifacts
  and next acceptance are recorded in `docs/screen-manager-usb.md` within the
  experiment. No host service/USB command executed: explicit macOS execution
  permission is needed for this next physical step. S5 remains accepted.

### 2026-09-06 — captured S6 bitmap replay

- **Fact:** user authorized capture, inspection and one direct S6 bitmap replay.
  Existing `cmd/blitz-preview` uses manager Screen -> checked BZR4 -> real Blitz.
  A diagnostic pipe captured its exact output before the existing decoder;
  the PNG preview is decoded from those same returned bytes, without a stamp.
- **Measurement:** `build/stream/s6/replay.bzm` is 48,008 bytes, BZM1 header
  `42 5a 4d 31 20 03 e0 01` (800x480). SHA-256:
  `85255ce0e5e2849b029b30885aeadea36eb6de49cd46445c857a9db193468011`.
  `replay.png` visibly contains heading, two checker images and S6 corner badge;
  SHA-256 `6d3051007a88156f4c3f72bf9d4c6a27bae86171692f87c7e6dbb0e3eca9a970`
  also matches the earlier inspected S6 PNG exactly.
- **Measurement:** frameio, blitz-preview and epaperstream focused tests pass.
  S6 USB client and firmware artifact hashes match the successful raw control.
- **Boundary:** this is a newly captured render, not recovered bytes from the
  earlier failed physical send, whose bitmap was not retained. Send this
  immutable BZM1 through the same raw client as the successful control, with
  no renderer/manager running during USB, no firmware/wiring/cadence changes.
  Acceptance initially awaited observation; the user's confirmation follows.
- **Measurement, USB:** after the 17:05:09 UTC preflight, invoked exactly once:
  `./build/stream/s6/epaperstream-macos-arm64 --raw /dev/cu.usbmodem2101 < ./build/stream/s6/replay.bzm`.
  By 17:05:31 UTC the client exited 0 with
  `stream=committed (verify visible pixels separately)`. User was told to watch
  immediately before invocation. No retry; process stopped.
- **Measurement, visible:** user then confirmed «є зображенння» when asked
  whether S6 appeared instead of the half-black/half-white control. S6 replay
  has physical acceptance, separately from PNG inspection and protocol reply.
  This does not retroactively accept the earlier failed live-manager send or
  identify its cause. No software fix, rewiring or extra refresh was performed.

### 2026-09-06 — bitmap-only control transfer

- **Fact:** generated `build/stream/raw-control.bzm` directly as BZM1, 800x480,
  48,000 pixel bytes plus 8-byte header. No HTML, CSS, fonts, Blitz, manager,
  firmware replacement or wiring change. Left half black, right half white.
- **Measurement:** existing `frameio.Read` accepted geometry and length; all
  384,000 pixels were checked against the half-plane predicate. Exactly 192,000
  black and 192,000 white pixels; decoded PNG preview visually inspected.
  BZM1 SHA-256:
  `da78c255621494fd363998ddea4f6b5343c8c795ce325d8e0ad5750e20b33676`.
- **Fact:** one direct native invocation of the existing S6 USB binary with
  `--raw /dev/cu.usbmodem2101 < build/stream/raw-control.bzm`; no retries.
  Client SHA-256:
  `12fc2cba747cc1a5f704f5e213d9c43f85a509c9d9e7563c3c22635d9dfc5a4c`.
  Firmware artifact hash remains the S5 accepted hash recorded above.
- **Measurement:** by 17:00:09 UTC, client exited 0 with
  `stream=committed (verify visible pixels separately)`; process is stopped.
  User was warned immediately before the send to watch for a refresh.
- **Measurement, visible:** after the initially pending observation, user
  confirmed «є»: the half-black/half-white control appeared. The complete raw
  path physically succeeded. No additional frame was sent after confirmation.
- **Unknown:** cause of S4/S6 failure and intermittent-contact reliability.
  Rendering was bypassed, but this alone does not prove a renderer defect.

### 2026-09-06 — S4 resend failed visible acceptance; raw control requested

- **Measurement:** user reported that S4 did not appear despite the client's
  `stream=committed`. This supersedes the pending observation below.
- **Fact:** the S4 HTML was rendered again using the current Blitz worker;
  no archived previously accepted bitmap was replayed. This was explicitly
  clarified to the user. The result does not uniquely implicate wiring.
- **Decision:** user requested a bitmap-only upload. Prepare an exact 800x480
  half-black/half-white static BZM1 artifact, validate all pixels through the
  existing frame reader, then send once using the existing raw USB mode.
  No HTML, Blitz, manager, firmware change, pin change or cadence override.
- **Guard:** new input content removes rendering as a dependency, but still
  exercises the same USB/firmware/HAT/panel path. Only visible acceptance can
  establish that physical refresh worked in this attempt; no automatic retries.

### 2026-09-06 — user-requested S4 control resend after S6

- **Fact:** user did not watch the S6 refresh, then requested S4 to check the
  physical path. Sent exactly one existing `build/stream/acceptance-4.html`
  with the standalone macOS client and container Blitz worker, directly through
  `/dev/cu.usbmodem2101`. No manager, reset, reflash, rewiring or limiter change.
- **Measurement:** existing Dev Container was running; firmware artifact still
  hashes to `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
  S4 HTML SHA-256 is
  `320c88ae78d0864040b70d090c98a59ef341ab90d3eb2f07c0ba4f0a04f4efe6`;
  client SHA-256 is
  `0f4b1d3a801627583e823f441f4cfe2ef2e570637c0152fcde1710575d26e388`.
  Transfer started after 16:51:01 UTC and returned exit 0,
  `stream=committed (verify visible pixels separately)`.
- **Unknown:** visible output awaits the user. Expected `STREAM USB — 4` and
  `ТЕСТ S4 · 05.09.2026`; the old date is an intentional fixed test label.
- **Guard:** user was told to watch immediately before invoking the client.
  No retry. A visible S4 would establish that this path worked for this upload,
  not prove permanent wiring reliability or identify the S6 failure cause.
  This control bypasses the S6 manager and uses its older standalone client;
  it is not a single-variable isolation of the complete S6 pipeline.

### 2026-09-06 — S6 visibility correction and read-only diagnosis

- User reported “ні, s5” after terminal `confirmed=1`: S5 remains visible.
  This corrects the pending acceptance recorded below; S6 is not physically
  accepted. No further frame, reflash, reset or wiring change was attempted.
- Fresh random USB epochs and per-pass digest checks make a simple reused ID=1
  explanation unsupported by inspected code. Four focused protocol/worker
  package test suites pass; those are not evidence of visible output.
- `panel.waitIdle` returns success for an already-high BUSY input. The streaming
  device does not export its panel observer events. Software completion therefore
  cannot distinguish every failure to execute a physical refresh. Cause remains
  unknown; do not infer a PWR fault or change pins from this result.
- Next diagnostic must distinguish whether a refresh cycle actually occurred;
  preserve S5 and do not silently resend or replace the working firmware.

### 2026-09-06 — authorized direct S6 image delivery

- **Approval / scope:** user approved starting the native macOS loopback manager
  and one direct USB S6 update, without reflash. Serial node
  `/dev/cu.usbmodem2101` exists; matched Dev Container `1611f88ac6e1` is running.
  Initial sandboxed Docker inspection was denied; authorized escalated read
  succeeded. No container recreation or USB attach/detach was needed.
- **Preflight / facts:** S6 manager and USB binary hashes match the prepared
  record; streaming UF2 hash is unchanged. Existing RSA localhost certificate
  covers 127.0.0.1 and is valid at this run; API token/private key are mode 0600.
  No secret contents exposed or TLS verification bypassed.
- **Measurement, UTC:** manager started at approximately 16:39:30 on
  `127.0.0.1:19443`. By 16:40:09, one conditional PUT of
  `testdata/stream-image-s6.html` returned revision 1; status was
  `current=1,in_flight=0,confirmed=0` during its unchanged 180s startup guard.
  No second submission, forced reset, reflash, rewiring or interval change.
  Terminal completion and visible output remain pending at this point.
- **Measurement, terminal success:** at 16:42:59 UTC (19:42:59 Europe/Kyiv),
  authenticated status returned `current=1,in_flight=0,confirmed=1`. The matched
  terminal EPS1 result passed through direct native USB to the manager. The
  manager then stopped with SIGINT, exit 0. Exactly one scene was submitted;
  no automatic retry, reflash, Pico reset or background manager was left running.
- **Boundary:** S6 now has runtime protocol acceptance for Go/BZR4 → Blitz →
  native USB → unchanged streaming firmware. Visible heading, both image copies
  and corner label still require the user's observation. Do not promote S6 over
  S5 as the known-good visible checkpoint until that separate confirmation.

### 2026-09-07 — first native Go candidate flash, USB result uncertain

- **Approval / scope:** user rebuilt the Dev Container, connected Pico, then
  explicitly authorized direct macOS port discovery, candidate UF2 flashing
  and one HTML submission. No OrbStack attachment or wiring change authorized
  or performed. Changed variable: accepted EPS1 firmware to the new unified
  TinyGo EPS2 candidate; physical acceptance is not yet established.
- **Fact:** matching container `2b739d1fc4c5` mounts the same project, resolves
  Go 1.26.8/TinyGo 0.41.1 and passes `go test ./screenwire`. Candidate UF2 is
  `experiments/03-remote-epaper/build/native-candidate.gEnGoa/screen-device-pico2w.uf2`,
  SHA-256 `d6c5b0726bc08c4c9ed1bff4594c1685ae24779151ccd45a3851e7a28c01c5ef`.
  Native sender SHA-256 is
  `1e40d9731e0e8e785e87d91e8444b4a3e2a5014cce571cb0b1b209e38f1f66cd`.
  The retained accepted EPS1 UF2 still hashes to
  `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
- **Measurement:** macOS `/dev/cu.usbmodem101`; one Generic CDC device,
  VID/PID `2e8a:000a`. TinyGo 0.41.1 `src/machine/usb/cdc/usbcdc.go` confirms
  bootloader entry at 1200 baud with DTR low. The authorized
  `stty -f /dev/cu.usbmodem101 1200 hupcl` exited 0; RP2350 INFO_UF2 appeared
  and explicitly identified Model Raspberry Pi RP2350 / Board-ID RP2350.
- **Measurement:** exactly one `cp` of the candidate UF2 to
  `/Volumes/RP2350/screen-device-pico2w.uf2` exited 1 with
  `Device not configured`. The BOOTSEL volume disappeared and the expected CDC
  port did not reappear during a bounded 15-second check. A subsequent USB bus
  inventory found no Pico on macOS; the container also had no Raspberry Pi USB
  device or serial node. Status recorded at 2026-09-07 09:01:16 UTC.
- **Unknown:** whether the image was partially or fully written, whether its
  startup failed, or whether USB connection/ownership changed. Neither the copy
  error nor absent enumeration proves a panel, HAT, solder or GPIO fault.
  No EPS2 Hello/Health or HTML/image submission has yet been attempted.
- **Decision / guard:** do not blindly copy again or alter the driver. Ask the
  user for physical BOOTSEL reconnect, identify the bootloader again, then decide
  the recovery/check action from observable state. No automated retry, flash
  erase, pin/FPC/HAT change or Attach/Detach. Known-good visible S6 remains the
  historical recovery checkpoint, not the current firmware claim.
- **Sources:** exact TinyGo CDC implementation above; current hardware-source
  hierarchy and accepted wiring unchanged. Candidate limits and complete
  software evidence: `experiments/03-remote-epaper/docs/native-candidate-acceptance.md`.
  Unlike historical text diagnostics, the new candidate is request/reply EPS2;
  a competing passive CDC reader must not consume its protocol responses.

### 2026-09-07 — manual BOOTSEL retry copied, runtime USB still absent

- **Changed variable:** user physically reconnected with BOOTSEL. The same
  candidate UF2 (`d6c5b072…c01c5ef`) was copied once after re-reading
  INFO_UF2: Raspberry Pi RP2350 / Board-ID RP2350. No firmware rebuild,
  flash erase, wiring/HAT/FPC change or OrbStack Attach/Detach.
- **Measurement:** this `cp` exited 0. The boot volume disappeared, but no
  `/dev/cu.usbmodem*` port appeared during a bounded 10-second discovery.
  Subsequent macOS USB inventory and container USB/serial inventory found no
  Pico. This differs from the first attempt's copy error; both are retained.
- **Boundary:** a successful file copy is not proof that the candidate runs.
  No EPS2 Hello/Health, HTML submission or display update was attempted.
  The cause of absent runtime USB remains unknown; do not infer a display,
  GPIO, solder or renderer fault from it.
- **Decision / next check:** ask for an ordinary USB disconnect/reconnect
  **without BOOTSEL**, keeping batteries disconnected and wiring unchanged.
  This distinguishes cold startup from bootloader handoff before changing code
  or reflashing. USB absence currently prevents an on-device status query.
- **Source / guard:** TinyGo 0.41.1 `src/runtime/runtime_rp2.go` initializes
  serial in runtime `init`; our `cmd/screen-device/main.go` performs panel IO,
  identity/flash boot and application composition before its service loop.
  These are investigation boundaries, not a proven startup root cause.
  Existing accepted EPS1/S6 recovery artifacts remain unchanged; E5 stays open.

### 2026-09-07 — user distinguishes ROM BOOTSEL from normal startup

- **User observation:** the device appears when connected with BOOTSEL, but
  is not seen when connected without it. This is the user's report, not a fresh
  host USB capture; the previous automated copy/discovery results stand.
- **Verified source:** Raspberry Pi documents BOOTSEL as ROM-resident,
  independent of the installed flash application:
  https://www.raspberrypi.com/documentation/microcontrollers/pico-series.html#reset-flash-memory.
  Enumeration there proves that the bootloader can communicate over USB, not
  that our application, external flash or panel is healthy.
- **Read-only code check:** `scripts/native-release.sh` builds `./cmd/screen-device`
  with `-target=pico2-w -scheduler=tasks`. Installed TinyGo 0.41.1 target chain
  is `pico2-w -> pico2 -> rp2350`; the latter specifies `serial: usb`.
  No accidental non-wireless target or deliberately disabled USB was found
  in this build path. Our main services EPS2 only after board/identity/boot
  composition; those operations are investigation candidates, not proven faults.
- **Inference / decision:** focus on normal boot and runtime USB availability;
  do not change HAT, GPIO or FPC or blame HTML rendering. Next useful controlled
  comparison is the retained known-good firmware versus this candidate, then
  a minimal startup reproduction if needed. Do not repeat the same candidate
  copy as a diagnosis. No device operation or firmware/code change in this check.
- **Unknown:** exact failing startup stage and whether the candidate fully
  executes. EPS2 and visible acceptance remain untested; recovery artifact and
  immutable candidate remain preserved.

### 2026-09-07 — retained EPS1 firmware restores runtime USB

- **Approval / changed variable:** user approved restoring the known-good
  firmware for comparison, then confirmed physical BOOTSEL reconnect.
  Only the installed application was changed; no wiring/HAT/FPC adjustment,
  OrbStack Attach/Detach, flash-wide erase, provisioning or image submission.
- **Precondition / measurement:** INFO_UF2 identified Raspberry Pi RP2350 /
  Board-ID RP2350. Container SHA-256 verification of
  `experiments/03-remote-epaper/build/stream/stream-device.uf2` matched the
  accepted artifact:
  `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
- **Action:** one direct macOS copy of that exact file to
  `/Volumes/RP2350/stream-device.uf2`; `cp` exited 0. No rebuild or automatic
  reflash retry. Candidate `d6c5b072…c01c5ef` remains unchanged on disk.
- **Measurement:** `/dev/cu.usbmodem1101` appeared in the first discovery check.
  Subsequent macOS IOUSB inventory showed a newly enumerated device at
  `01100000`. Result recorded at 2026-09-07 09:22:58 UTC.
- **Inference:** runtime USB enumeration works with the retained firmware.
  Against the absent USB after the candidate, this strongly directs debugging
  to candidate startup/build/runtime or persistent-state interaction, not an
  assumption that the screen, HAT or cable needs replacement. A single recovery
  does not prove the precise failing line or rule out every intermittent fault.
- **Boundary / next work:** USB enumeration only, not an EPS1 protocol or new
  visible-image acceptance. No binary status request, HTML or bitmap was sent.
  Keep this recovered firmware installed while reducing the candidate's early
  startup path offline. Do not promote E5 or disturb the panel to diagnose USB.

### 2026-09-07 — prepare gated startup diagnostic, not a candidate fix

- **Scope:** user asked to continue after EPS1 restored USB. Production source,
  immutable candidate and known-good UF2 stay unchanged. New isolated
  `cmd/startup-probe` reuses actual board/UID/boot/USB/radio constructors;
  no panel refresh, radio Init/Join or network worker is started.
- **Static evidence:** exact TinyGo 0.41.1 CDC Write queues bytes; a printed
  BEGIN alone is not proof the host received it before a fatal call. Probe
  requires `n -> observed BEGIN -> r -> DONE/FAIL`, one stage at a time.
  Candidate ELF places flash_do_cmd and direct helpers in SRAM; not evidence
  that flash operations return correctly. Detailed sources and limits:
  `experiments/03-remote-epaper/docs/startup-probe.md`.
- **Regression guards:** RED/GREEN verified prepare does not execute hardware;
  action/output failures halt and cannot repeat a durable epoch write.
  Independent review found surviving RX bytes across DTR reconnect. Corrected
  by latching STOP after an observed established-session disconnect, and added
  a regression test. This trusted diagnostic is not an authentication boundary.
- **Software evidence:** focused race tests have 100% sequencer statement
  coverage; hardware main remains a target-only boundary. Focused lint and
  TinyGo pico2-w/tasks build pass. First full task gate passed in 68s before
  the final reconnect guard; final task gate passed in 28s after that change.
  Changed coverage 91.7%, total 92.2%; no checker suppression or skipped tests.
- **Prepared UF2:** `build/startup-probe-v1.uf2`, SHA-256
  `07755cc061d0770b937c0e9750529a941f3c42bc02d10e7fb1d5badafd0f2a9e`.
  No claim of a root cause or hardware result yet. Next action is controlled
  native USB flashing and one persistent text session; no automatic step replay.

### 2026-09-07 — startup probe stops responding at flash UID

- **Changed variable:** installed the isolated probe UF2 `07755cc0…f0f2a9e`
  instead of retained EPS1. One native 1200-baud boot request, verified RP2350
  INFO_UF2, then one direct macOS copy; exit 0. CDC `usbmodem1101` returned.
  Candidate package, wiring, HAT switches and FPC remained unchanged.
- **Measurement:** one native serial client received READY. Each action was
  separately host-acknowledged after its BEGIN line. Observed sequence:
  `BEGIN panel_io`, `DONE panel_io`, `BEGIN flash_uid`; after the next `r`,
  no DONE or FAIL arrived during the bounded observation (still absent at
  09:43:46 UTC). The command was not replayed. Serial client subsequently closed.
- **Inference:** first observed nonreturning boundary is actual `pico2w.UID()`,
  which guards `machine.DeviceID()` with disabled interrupts. This narrows the
  candidate's early-startup investigation; it does not identify the exact CPU
  instruction or distinguish a fault, wait loop, or loss of USB service.
- **Not executed:** boot-session/epoch reservation, USB writer constructor,
  radio constructor/Init/Join, EPS2, HTML or panel refresh. This experiment did
  not write credentials or the epoch journal. No UID or secrets were logged.
- **Source check:** TinyGo v0.41.1 uses the shared RP2 ID command and RP2350
  ROM/QMI path. The Pico SDK also copies its XIP restore stub from BOOTRAM;
  that address alone is not evidence of a bug. Exact links and reproduction:
  `experiments/03-remote-epaper/docs/startup-probe.md`.
- **Guard / next action:** preserve this failing artifact and all tests;
  restore retained EPS1 before another physical experiment. Do not bypass
  identity, durable state or authentication to make the candidate boot. A
  reduced probe is not a full candidate runtime or visual acceptance result.

### 2026-09-07 — matching TinyGo UID bug found; manual recovery required

- **Fact / sources:** TinyGo issue https://github.com/tinygo-org/tinygo/issues/5408
  reports DeviceID hanging Tiny2350. Merged PR #5413 fixes it and #5412:
  https://github.com/tinygo-org/tinygo/pull/5413, commit
  `2747027ef261ed6c270818ffe32cb155fd3de15c`. The patch changes ROM lookup
  pointer size and flash CS control. Installed TinyGo 0.41.1 has the old code;
  stable 0.42.0 (published 2026-09-01) release notes include the fix:
  https://github.com/tinygo-org/tinygo/releases/tag/v0.42.0.
- **Inference:** strong upstream match for the observed UID boundary, but the
  issue's board differs and neither the exact internal failing instruction nor
  successful operation with 0.42.0 has been established on this Pico.
- **Recovery measurement:** one direct 1200-baud `stty` did not complete; it was
  interrupted, exit 130. No RP2350 volume appeared on the subsequent check.
  Serial client and reset process are closed. Do not leave queued host actions
  or repeat the reset. Manual BOOTSEL reconnect is now needed for EPS1 recovery.
- **Decision / guard:** no vendor patch, UID bypass, disabled test or security
  downgrade. Next toolchain comparison should use the official fixed release,
  persist its pin/prerequisites, preserve the failing probe and known-good UF2,
  and rerun software gates plus the same acknowledged physical probe. Do not
  auto-rebuild the container or count release notes as physical acceptance.

### 2026-09-07 — EPS1 recovery confirmed by Hello, official upgrade prepared

- **Approval / changed variable:** user confirmed manual BOOTSEL. INFO_UF2
  again identified Raspberry Pi RP2350. Retained `build/stream/stream-device.uf2`
  SHA-256 was rechecked in the existing Dev Container and matched
  `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
- **Measurement:** one direct macOS `cp` exited 1 with `could not copy extended
  attributes ... Operation not permitted`. Do not rewrite this as cp success.
  Runtime CDC returned as `/dev/cu.usbmodem1101`; no automatic reflash followed.
- **Verification:** a host-only Go diagnostic, built in the existing container
  from `build/eps1-hello-check/main.go`, reused the existing streamwire codec.
  It sent exactly one fresh-epoch Hello, validated reply identity and CRC, and
  exited 0: `protocol=EPS1 hello=ok state=idle width=800 height=480 max_chunk=100
  passes=2 full_interval_s=180 crc=ok`. Recorded by 09:57:50 UTC.
  It has a 3s read deadline and 10s whole-process limit; no Begin/Data/Commit,
  retry, image, or refresh was sent. `go test ./streamwire` passed.
- **Inference / boundary:** the metadata warning did not prevent recovery of
  a running EPS1 application. Hello establishes runtime protocol availability,
  not flash readback identity or a new visual acceptance. Keep S6 as the latest
  visible checkpoint. USB client is closed; no queued host writes remain.
- **Separate next change:** `.devcontainer/Dockerfile` now pins the official
  `ghcr.io/tinygo-org/tinygo:0.42.0`, retaining all other current dependencies.
  TinyGo's official Docker instructions list this exact image/tag. Existing
  Rust-removal changes were preserved. No local runtime patch or bypass.
- **Software check:** current-container `go test -race ./cmd/startup-probe
  ./streamwire ./streamapp` passed; scoped `git diff --check` passed. The
  version command still reports TinyGo 0.41.1 / Go 1.26.8 / LLVM 20.1.1.
- **Pending:** container recreation by the user, version/source verification,
  focused and full gates, unchanged startup-probe comparison, then candidate
  rebuild/physical acceptance. The current container remains 0.41.1; neither
  a 0.42 build nor resolution of the UID hang is claimed. No commit or push.

### 2026-09-07 — rebuilt container and unchanged 0.42.0 probe

- **Fact:** user confirmed rebuild; matching container `0b3e0dea34f7` reports
  TinyGo 0.42.0 / Go 1.27.0 / LLVM 22.1.4. Installed RP2350 source contains
  both upstream fixes: fixed 2-byte ROM table pointer size and QMI CS control.
- **Scope:** same source, target pico2-w and explicit tasks scheduler. This is
  a complete official toolchain comparison, not isolation of one patch: Go,
  LLVM and other TinyGo runtime changes also differ. No runtime patch or UID
  bypass. Existing failing probe and accepted EPS1 hashes still match.
- **Software evidence:** focused race tests for startup-probe, streamwire,
  streamapp and screenboot pass. New artifact
  `build/startup-probe-tinygo-0.42.0.uf2`, SHA-256
  `4eb8a3d51e52478d0eacf788d659d90362aa49e5c5c668cc7136f29387ac28ee`.
  TinyGo short report: code 364408, data 8, bss 10076 bytes; headline RAM 10084
  is not measured peak. Probe flash size grew versus 0.41.1; no efficiency
  improvement is claimed. Full task gate subsequently failed at lint; see below.
- **Next controlled action:** one direct native USB flash then the same
  n/observed-BEGIN/r sequence, starting with panel_io and flash_uid. No automatic
  replay. macOS copy will use `cp -X` to omit extended attributes/resource forks,
  as documented by Apple's cp manual:
  https://github.com/apple-oss-distributions/file_cmds/blob/main/cp/cp.1.
  This host metadata option does not change firmware bytes or panel behavior.

### 2026-09-07 — bundled Go 1.27 triggers pinned analyzer panic

- **Measurement:** first 0.42.0 task gate exited 3 before coverage/build:
  staticcheck's buildir panicked `unexpected expr: *ast.KeyValueExpr` while
  analyzing dependency package `poll`. Report
  `build/startup-tinygo-0.42.0-quality.log` is preserved. This is an analyzer
  failure, not a reported project lint violation. No new firmware was flashed.
- **Decision:** retain the project's already selected `go1.26.8` while upgrading
  TinyGo, rather than disabling lint or patching a vendor. `toolchain go1.26.8`
  in go.mod is a preference/minimum under auto selection, not an upper bound;
  new image's bundled Go 1.27 was selected. Official selection semantics:
  https://go.dev/doc/toolchain. No go.mod or quality-threshold change.
- **Persistence:** Dockerfile now has `GOTOOLCHAIN=go1.26.8` and `RUN go version`
  to prefetch/verify the selected toolchain. Current-container comparison uses
  explicit `docker exec -e GOTOOLCHAIN=go1.26.8`; no auto-rebuild or host install.
- **Verification:** same full task gate with Go 1.26.8 passed in 57s; report
  `build/startup-tinygo-0.42.0-go1.26.8-quality.log`. No tests/checkers disabled.
  Unchanged probe build passed: code 182816, data 8, bss 9900 bytes (not peak
  RAM). Artifact `build/startup-probe-tinygo-0.42.0-go1.26.8.uf2`, SHA-256
  `dec336eb94e4ad797fe47e26b7c1076b1d0218c7d6b63ab02d31ccf4108c9735`.
  This is the selected physical-comparison artifact. The Go 1.27-built probe
  remains on disk, not installed on Pico.

### 2026-09-07 — UID probe completes with official TinyGo 0.42.0

- **Changed variable:** official toolchain build, application source unchanged.
  Selected UF2 `dec336eb…8c9735` was flashed once after native 1200-baud reset
  and verified RP2350 INFO_UF2. `cp -X` exited 0; CDC usbmodem1101 returned.
- **Measurement:** one persistent native USB session observed READY and the
  acknowledged sequence `BEGIN panel_io` / `DONE panel_io`, then
  `BEGIN flash_uid` / `DONE flash_uid`. Each r was sent only after reading BEGIN.
- **Inference:** the observed UID hang is resolved in this controlled probe
  with the official new toolchain. This supports the matching upstream bug;
  no vendor patch or security bypass was necessary. It does not isolate which
  upstream change fixes it or establish complete EPS2/radio/pixel acceptance.
- **Boundary:** at this checkpoint boot-session reservation and later stages
  had not run. Next checks reuse the same acknowledged sequence, with one
  bounded epoch reservation and no credential mutation or display refresh.

### 2026-09-07 — next boundary: boot_session returns FAIL, not a hang

- **Measurement:** after successful UID on the unchanged 0.42/Go1.26.8 probe,
  host observed BEGIN boot_session, then sent r once and received FAIL
  boot_session. The probe halted; USB remained responsive. No retry, later
  constructor, radio join or refresh. Serial client was closed afterward.
- **Meaning:** `screenboot.New(machine.Flash, uid)` returned without a Lifetime.
  This is an explicit control-only recovery result, not proof of corrupt
  credentials or a failed erase. Its current API does not distinguish geometry,
  storage construction, epoch reservation or lifetime construction in the probe.
  Epoch reservation was attempted only if earlier preconditions passed; no
  claim that flash was written or remained unchanged is made without evidence.
- **Decision:** restore retained EPS1, inspect the first failing precondition,
  and add bounded numeric diagnostic detail if needed. Never erase provisioned
  data, invent an epoch, or weaken authentication to continue. The UID issue
  remains resolved for this probe; complete candidate acceptance remains open.

### 2026-09-07 — retained EPS1 recovery verified after boot-session failure

- **Measurement:** native 1200-baud reset succeeded; INFO_UF2 identified RP2350.
  Restored retained `a17ae2a2…a92185f` once with `cp -X`, exit 0. One native
  fresh-epoch EPS1 Hello returned idle, 800x480, max_chunk=100, passes=2,
  full_interval_s=180, CRC valid. No image sent and no new visible claim.
- **Next variable:** add fixed boot-failure labels to the diagnostic only.
  Same storage/admission policy, no erase, fake epoch, retry or wiring change.
  The installed source reports 256-byte writes and 4096-byte erases; these
  match our accepted geometry. Exact runtime fault remains to be measured.

### 2026-09-07 — fixed-cause startup probe v2 software gate

- **Change:** retain screenboot's private error and expose only whitelisted
  failure labels. Probe prints BOOT label before its existing FAIL. Normal
  startup/reservation/admission policy unchanged; no raw errors or flash bytes.
- **Verification:** RED failed for missing API; focused unit/race GREEN;
  full task PASS in 27s, changed/total coverage 91.7% (baseline 75.0%). Report
  `build/startup-probe-v2-quality.log`. No checks weakened or skipped.
- **Artifact:** TinyGo 0.42.0 / Go 1.26.8, code 184356/data 8/bss 9900
  (headline, not peak RAM); `build/startup-probe-v2-tinygo-0.42.0.uf2`, SHA-256
  `5df376e92a70241646de8132a50b92f345b314f1158420c31f0f81c78bef8794`.
  Next physical comparison changes only diagnostic detail, not toolchain or
  storage policy, using the same n/observed-BEGIN/r progression.

### 2026-09-07 — physical cause narrowed to invalid epoch journal

- **Measurement:** probe v2 `5df376e9…bef8794` copied once via native USB,
  exit 0; READY v2, DONE panel_io, DONE flash_uid, then acknowledged
  boot_session returned `BOOT epoch_corrupt` followed by `FAIL boot_session`.
  No retry, radio constructor/join, frame, credential operation or recovery
  erase was requested. Independent source review found no required changes.
- **Meaning:** valid geometry and nonzero UID passed; epochstore.Reserve
  returned ErrCorrupt. That error can mean an invalid/non-erased scanned record
  or a mismatched write verification. This probe does not distinguish those
  internal cases or date the invalid data. It is not proof of physical flash
  damage, a wiring defect, or which previous firmware wrote those bytes.
- **Security boundary:** never fall back to an older epoch or invent one.
  Existing recovery erases credential blocks first with readback, then epoch
  blocks, preventing counter reuse under a retained key. Requires explicit
  user approval, reboot, fresh-key enrollment and reset of associated host
  session/cache state. No such approval or erase occurred in this experiment.
- **Next:** restore known-good EPS1; request authorization before any destructive
  recovery. The official TinyGo upgrade has resolved the observed UID hang,
  but complete EPS2/WPA3/display acceptance remains open.

### 2026-09-07 — final recovery checkpoint after fixed-cause probe

- **Measurement:** closed native serial client; 1200-baud reset and RP2350
  identity check succeeded. One `cp -X` of retained EPS1 exited 0. One native
  Hello verified EPS1, idle, 800x480, max_chunk=100, passes=2,
  full_interval_s=180 and valid CRC. No panel refresh or recovery erase.
- **Handoff:** all changed code passed task gate and independent bounded review;
  git diff --check clean. New boot diagnostic is preserved, not installed.
  Await explicit approval before erasing the last four 4096-byte data blocks
  (credentials then epochs, verified separately). This is a proposed recovery
  experiment, not a claim that erasing will establish full firmware acceptance.

### 2026-09-07 — user authorizes bounded recovery of credentials and epochs

- **Authority:** user replied «так» to explicit deletion of the last four
  4096-byte data blocks (16 KiB), including saved Wi-Fi/authentication data and
  session journal. Re-enrollment requires fresh keys; no blanket flash erase,
  host enrollment deletion, wiring change or security bypass is authorized.
- **Preflight:** existing container `0b3e0dea34f7`, TinyGo 0.42.0 with explicit
  Go 1.26.8; focused recovery/provisioning unit/race tests pass. No source change
  since previous full task PASS. One native EPS1 Hello again verified idle/CRC.
- **Artifact:** unchanged unified composition rebuilt at
  `build/screen-device-tinygo-0.42.0-recovery.uf2`, SHA-256
  `8b844bb0f369989e6f6639dc04f8320cdf1b2853437e2520d6485c9622b69b4a`.
  Code 698168/data 16/bss 10092 bytes, not measured peak RAM. Frozen old package
  remains untouched. Use its verified native epaperprovision client; no keys
  in arguments, and no provisioning/rotation request in this experiment.
- **Plan:** confirm physical recovery response, execute one approved erase via
  existing USB control, require success/blank readback, reboot and inspect new
  session readiness. An uncertain erase is not blindly repeated.

### 2026-09-07 — unified 0.42.0 firmware reaches USB control-only recovery

- **Measurement:** RP2350 BOOTSEL identity verified; one native `cp -X` of
  unified `8b844bb0…2b69b4a` exited 0. Native provisioning inspect replied
  `code=5 state=0 generation=0`, empty public identity/configuration fields.
  The CLI exits 1 because CodeRecovery (5) is not success; this is an intact
  decoded response, not another startup/USB timeout.
- **Meaning:** credentials are reported blank; safe session initialization
  still requires recovery. This independently confirms the full composition
  can now reach its USB recovery loop with the official toolchain upgrade.
  Normal EPS2, network and display paths have not run.
- **Next:** one explicitly approved USB erase; firmware verifies credential
  blocks before touching epoch blocks. Do not interpret a missing ACK as
  permission to repeat the mutation.

### 2026-09-07 — approved 16 KiB recovery erase succeeds

- **Measurement:** exactly one native `epaperprovision -confirm-erase erase`
  returned `code=0 state=0 generation=0`, empty metadata, exit 0. The current
  recovery implementation returns this only after verifying all credential
  bytes, then all epoch bytes, as erased and reloading the blank store.
- **Removed:** last four 4096-byte data blocks only. Previous credential report
  was blank; journal contents are deliberately not backed up or reused. Erased
  on-device bytes cannot be recovered by this operation; fresh enrollment is
  required for Wi-Fi. No host enrollment/session files were deleted or reused.
- **Next:** reset/reload the exact same unified UF2 to reserve a fresh epoch.
  Current recovery boot deliberately stays control-only until reboot. USB-only
  screen acceptance does not require enrollment or any wireless credentials.

### 2026-09-07 — same firmware boots normally after verified recovery

- **Measurement:** native 1200-baud BOOTSEL transition and RP2350 identity check
  succeeded. Reloaded identical unified `8b844bb0…2b69b4a` (copy exit 0).
  Post-reboot inspect returned `code=0 state=0 generation=0`, exit 0, rather
  than the pre-recovery CodeRecovery=5. Credentials remain blank.
- **Inference:** the epoch admission failure is cleared after verified data
  reset; same firmware/toolchain now reaches the normal owner. This supports
  invalid prior journal state, not a persistent flash-read/write fault. Exact
  origin of invalid bytes remains unknown. No wireless enrollment/join or
  new display transaction has been requested yet.
- **Next:** obtain EPS2 Hello/Health using the native sender, without pixels.

### 2026-09-07 — first normal EPS2 Hello/Health on unified native firmware

- **Measurement:** native `epaperscreen --status` exited 0 and reported profile=1,
  version=1, size=800x480, passes=2, chunk=1000, minimum_full_ms=180000,
  remaining_ms=147871; network state=0, last_failure=0, failures=0 at uptime=32s.
  No pixels were sent. This is the first normal EPS2 response on the unified
  native candidate, not merely successful UF2 copying or EPS1 fallback.
- **Next authorized acceptance step:** one direct native HTML submission using
  retained package sender `1e40d973…f1f66cd` and dashboard HTML
  `5728807c…e3392a`. Exact source and synthetic stamped PNG inspected. Actual
  send stamps the clock after readiness; preview time is not sent as evidence.
  Keep 180s floor, no partial refresh, no wiring or enrollment change. Wait for
  confirmed protocol outcome, then separately ask user about visible pixels.

### 2026-09-07 — first native HTML send has unconfirmed outcome

- **Measurement:** single native sender invocation ended at its five-minute
  context deadline, exit 1: `screen outcome unconfirmed; no automatic replay
  was attempted` and `context deadline exceeded`. No success line or visible
  confirmation. The operation included the unchanged 180s cold/full cadence.
- **Boundary:** recovery/normal EPS2 Hello are proven, full HTML delivery is
  not. Current CLI output does not identify the last Data/Commit/Query boundary.
  A timeout cannot distinguish lost reply from an operation not applied. No
  automatic or manual second image submission, erase or wiring change.
- **Next:** read-only state/health after the sender and supervised child close;
  preserve the unconfirmed attempt before deciding whether firmware recovery
  or bounded transport diagnostics are needed. Never call this pixel success.

### 2026-09-07 — post-timeout control works; expose existing transfer evidence

- **Measurement:** after the unconfirmed sender exited, native provisioning
  inspect still returned code=0/blank, and EPS2 status returned 800x480,
  remaining_ms=0, network failures=0 at uptime=469s. No reboot occurred.
- **Gap:** the existing CLI dropped already-decoded Status and Diagnostic
  fields from its text output. Add current state/pass/offset/current-image and
  fixed panel numeric evidence to `--status`; no firmware, wire, policy, retry,
  key logging or pixel action change. This snapshot does not reconstruct a lost
  ACK or assert an image is visibly present.
- **Verification:** missing-output RED, focused unit/race GREEN; task PASS23s,
  changed/total 91.7%. Report `build/status-evidence-quality.log`. Native host
  artifact `build/epaperscreen-status-v2-darwin-arm64`, SHA-256
  `f952c6ac526cd3dd9dcc1fb10c49699496cb0b4c37534ce7d8623c0a9c43aa97`.
  Next action is read-only status, not another upload.

### 2026-09-07 — correction: Health zeros are not transfer evidence

- **Measurement:** enhanced status exited 0 at uptime=723s with
  operation=Health, reply_code=0, state/pass/offset=0, current_image=false and
  zero panel Diagnostic. No image operation or reboot was performed.
- **Contradiction resolved from source:** `screenlink.dispatch` intentionally
  returns an empty Result for Hello/Health. Thus those zeros cannot establish
  Idle, zero received bytes, no panel fault or whether a frame appeared. The
  previous CLI addition assumed valid envelope fields contained live evidence;
  that inference was wrong. Root cause of the upload remains unlocalized.
- **Correction/guard:** `--status` now states `transfer_evidence=unavailable`
  rather than exposing envelope zeros as a transfer snapshot. Regression checks
  reject those false-evidence labels and require the explicit boundary. The
  underlying Inspect contract is unchanged: no wire/firmware/auth changes.
  Exact Query requires the original retained lease/transaction; neither its
  loss nor zero Health fields permits blind replay. Preserve earlier failed
  attempt and test report as historical, not final accepted behavior.
- **Next:** return to the known-good physical firmware after this unconfirmed
  experiment. Ask for the visible observation independently; a future transfer
  needs stage-aware host diagnostics before another attempt, not more guesses
  from discovery/status metadata.

### 2026-09-07 — user confirms old picture; discovery remains healthy

- **Visible observation:** user replied «Ні, залишилось старе зображення» to
  the explicit native dashboard check. First native HTML attempt is therefore
  not visibly accepted, not merely a missing success log. Latest visible S6
  checkpoint remains unchanged.
- **Measurement:** corrected native status `065de9ae…2dc953e` exited 0 at
  uptime=964s, normal EPS2, network failures=0, transfer evidence explicitly
  unavailable. Full task PASS22s, changed/total91.7%, diff check clean. Source
  review now includes firmware dispatch and confirms the discovery limitation.
- **Next investigation:** actual TinyGo USB RX buffering and per-record reply
  boundaries. No blind replay, extra erase, fallback key or rewiring. Restore
  retained EPS1 before another physical experiment, preserving current findings.

### 2026-09-07 — retained EPS1 restored after native visible failure

- **Measurement:** 1200-baud transition and RP2350 identity succeeded. One
  direct copy of unchanged `build/stream/stream-device.uf2` completed exit 0;
  SHA-256 `a17ae2a2116c20f7af09d615c20ca7b006d42cb93d1a5240ff2c7eaa9a92185f`.
  Native EPS1 Hello returned idle, 800x480, max_chunk=100, passes=2,
  full_interval_s=180, crc=ok. No pixels submitted or wires moved.
- **Guard:** preserve failed native UF2/sender/HTML and all prior evidence.
  Only a tested transport change may precede another native image attempt.

### 2026-09-07 — USB RX capacity mismatch found in installed sources

- **Fact:** installed TinyGo 0.42.0 CDC `ring.go` fixes RX capacity at 512
  bytes. `usbcdc.go:190` accepts only `min(packet length, rx.Free())` bytes.
  Our EPS2 header is 32 bytes; adapter MaxChunk=1000 permits a 1032-byte
  record. The proxy writes a complete record before awaiting its reply.
- **Inference:** a burst arriving before the owner drains RX can lose bytes;
  successful short Hello/Health does not exercise this boundary. This is a
  reproducible transport-contract hazard, not yet a measured explanation of
  the failed physical send. Compression cannot protect incompressible chunks.
- **Decision:** test the actual sender with a bounded RX model; constrain USB
  data records to fit without increasing Pico RAM or adding timing sleeps.
  Preserve negotiated device capabilities and the independent network path.
  Preserve request-boundary errors through failed reconciliation, without
  logging private identifiers or replaying frames.
- **Sources:** installed `/usr/local/tinygo/src/machine/usb/cdc/ring.go` and
  `usbcdc.go`, release 0.42.0; upstream reference links:
  https://github.com/tinygo-org/tinygo/blob/v0.42.0/src/machine/usb/cdc/ring.go
  and https://github.com/tinygo-org/tinygo/blob/v0.42.0/src/machine/usb/cdc/usbcdc.go.
  Web raw fetch failed; the stated behavior was verified in installed source.

### 2026-09-07 — host USB cap and failure evidence pass software gates

- **Measurement:** independent reproduction failed raw/mixed 800x480 transfer
  at 1032-byte burst, 520 bytes dropped, zero commits. Compressed-only passed.
  After USB data cap=480 (header32, total512), all three pass through real
  proxy/session, both byte-exact 48,000-byte planes, one Commit, no dropped bytes.
- **Implementation:** additive `screenclient.NewWithMaxChunk`; USB factory
  uses480, generic/network New retains1024. Negotiated capabilities remain
  unchanged; min(device,transport) and limit survive reconnect/reset. Packed
  payload never expands. No Pico source, RAM, refresh guard or wiring changes.
  Exchange failures keep attempted operation/pass/offset (not ACK evidence);
  unsuccessful reconciliation preserves original and terminal errors.
- **Verification:** diagnostic RED→GREEN, focused race PASS, independent
  review no required findings. Initial task rejected one complex test function;
  split it without dropping assertions. Final task PASS23s, changed/total91.7%,
  no suppressed checks; `build/usb-rx-budget-quality-final.log`. Diff check clean.
- **Artifact:** `build/epaperscreen-usb-rx-480-darwin-arm64`, SHA-256
  `c8072286c5d6ceb2a0e5477e5fb1fc14499468d7dc79547454f0e3c80981fa74`.
  Firmware8b844bb0…2b69b4a and HTML5728807c…e3392a remain byte-identical.
- **Next physical experiment:** reinstall that same unified firmware, verify
  normal EPS2 discovery, then one same-HTML send with the bounded native host.
  Only transport chunking/error reporting differs from the failed native run.
  Keep180s floor, no erase/enrollment/rewiring/partial refresh. Physical outcome
  remains open; software reproduction alone does not establish causality.

### 2026-09-07 — USB transition now requires manual BOOTSEL reconnect

- **Measurement:** `stty -f /dev/cu.usbmodem1101 1200 hupcl` returned0. The
  subsequent exact RP2350 INFO_UF2 read found no volume; native discovery found
  neither RP-prefixed volume nor cu.usbmodem port. Container `picotool info`
  reported no accessible RP-series BOOTSEL device, no ttyACM port; only USB
  root-hub nodes remained. No attach/detach or system modification attempted.
- **Boundary:** command exit0 does not prove successful bootloader enumeration.
  No UF2 was copied after this transition and no second HTML was sent. Last
  installed firmware remains retained EPS1; USB visibility is currently unknown.
- **Decision:** stop physical work and request manual disconnect/reconnect with
  BOOTSEL held. Leave all HAT wiring alone. After enumeration, verify target,
  install retained native8b844bb0…2b69b4a and test bounded senderc8072286…981fa74
  once. No additional recovery erase is authorized or required by this evidence.

### 2026-09-07 — manual BOOTSEL succeeds; same native UF2 reinstalled

- **Measurement:** user confirmed reconnect. INFO_UF2 reports bootloader v1.0,
  Model Raspberry Pi RP2350, Board-ID RP2350. Existing Dev Container remains
  running. Fresh SHA-256 checks match firmware `8b844bb0…2b69b4a`, bounded
  sender `c8072286…981fa74`, and HTML `5728807c…e3392a` recorded above.
  One direct `cp -X` to `/Volumes/RP2350/screen-device.uf2` completed exit 0.
- **Boundary:** this proves completed copying, not application startup or
  visible pixels. No erase, provisioning, rewiring or image submission yet.
  Only sender chunking/error evidence changes versus the earlier failed native
  attempt. Next obtain EPS2 status, then one bounded native HTML submission.

### 2026-09-07 — bounded sender discovers normal EPS2 before image attempt

- **Measurement:** native bounded sender `--status /dev/cu.usbmodem1101`
  returned0, profile1/version1, 800x480, two passes, advertised chunk1000,
  minimum_full_ms180000, remaining_ms137327, uptime42s, network failures0.
  Transfer evidence explicitly unavailable for Hello/Health.
- **Decision:** submit the same dashboard HTML once using native sender
  `c8072286…981fa74`, timezone Europe/Kiev. Actual USB decoded portions are
  capped480 despite the unchanged device maximum1000. Keep one retained
  session across readiness, sending and outcome reconciliation. Timestamp is
  sampled after readiness; no synthetic preview timestamp is submitted.

### 2026-09-07 — bounded native HTML delivery visibly accepted

- **Measurement, protocol:** the one sender started at11:42:36 UTC completed
  exit0 by11:46:22 UTC with `screen=confirmed (verify visible pixels separately)`.
  No retry, second send or additional erase. Native firmware unchanged from
  the prior failed attempt; host is the tested bounded senderc8072286…981fa74.
- **Measurement, visible:** user independently answered «Так, новий dashboard
  видно» after being asked about the Ukrainian heading, two blocks and corner
  time. New image replaces the earlier S6. No photo requested or necessary.
- **Inference:** the bounded USB sender fixes this observed delivery scenario,
  consistent with the independently reproduced RX overflow. This is not a
  hardware byte trace, nor proof that every historical failure shared one cause.
  The earlier UID hang, invalid epoch journal and manual BOOTSEL enumeration
  recovery remain separate documented boundaries.
- **Decision/guard:** promote the exact native pairing to the current known-good
  USB checkpoint above. Keep it installed and stop sending. Tests, source links,
  hashes and unchanged fallback artifacts prevent regression to oversized USB
  bursts. Wi-Fi/provisioning, reconnect acceptance, partial refresh and measured
  peak RAM/energy remain separate open gates; do not claim full project completion.

### 2026-09-07 — native USB repeat-update acceptance started

- **Scope:** user asked to continue after the accepted native dashboard. Same
  Pico/HAT/wiring and installed unified UF2 `8b844bb0…2b69b4a`; sender
  `c8072286…981fa74` and retained UF2 hashes rechecked. No flashing, reset,
  credential mutation, timing change or Attach/Detach.
- **Measurement, preflight:** native Hello/Health on `/dev/cu.usbmodem1101`
  returned profile1/version1, 800x480, two passes, advertised chunk1000,
  minimum180000ms/remaining0, network state0/failure0/count0, uptime862s.
  Discovery is not retained transaction evidence or proof of current pixels.
- **Changed variable:** HTML text only, in a new immutable-input directory:
  `experiments/03-remote-epaper/build/native-repeat.TeMKNU/dashboard.html`,
  SHA-256 `6ff8dc0c9452f9b1d1b558279b68224398b8d26df836fa37f271cabf8d1d6d93`.
  Heading «Повторне USB-оновлення», card «Кадр №2». Styles unchanged.
  Native preview rendered without warnings and was visually inspected; its
  fixed timestamp is synthetic, not the transmitted full-cycle time.
- **Decision/guard:** one send with the accepted bounded sender and
  `-timezone Europe/Kiev`; preserve the fresh-client180s floor and exact
  retained reconciliation, never automatically replay an ambiguous update.
  The first visible checkpoint stays authoritative until separately confirmed.
- **Measurement, protocol:** one native sender started by 13:23:03 UTC and
  finished by 13:26:38 UTC, exit0: `screen=confirmed (verify visible pixels
  separately)`. This includes the unchanged cold-client floor, not a throughput
  measurement. No second attempt, reflash or extra update followed. Asked the
  user whether «Повторне USB-оновлення» / «Кадр №2» appeared; answer pending.
- **Unknown:** actual changed pixels/time, physical cable reconnect, WPA3 and
  energy remain unaccepted. A new serial client is not a physical cable test.

### 2026-09-07 — operator checkpoint and next manager preparation

- **Fact:** live instructions still pointed at frozen pre-fix executables and
  said the migration was not committed. Corrected current guidance to accepted
  UF2/sender and commit `66dc9a1`; historical package and failures preserved.
  The old manager also embeds its own USB parent client, so replacing only
  its serial-proxy binary does not add the 480-byte bound.
- **Measurement:** current-source CGO=0 Darwin/arm64 manager build with
  Go1.26.8 completed exit0 inside existing container `0b3e0dea34f7`:
  `experiments/03-remote-epaper/build/native-repeat.TeMKNU/epaper-manager-darwin-arm64`,
  SHA-256 `641e3664be591fd7983a53ba74785794d23de2fa620b93f3751d1c6fd5b2e628`.
  It was not started. Container has no published ports. Asked for permission
  before a temporary native Mac listener; no firewall/router changes or private
  credential access/enrollment were performed.
- **Open packaging gap:** `scripts/native-release.sh` still requires0.41.1;
  current container/accepted firmware use0.42.0. Record an E4 follow-up to update
  the pin and associated notices/source checks together, without a downgrade,
  weakening version checks or overwriting either accepted/frozen artifact set.
- **Verification/guard:** native preview and manager build exit0; authored
  `git diff --check` clean. No production-code edits, test suppressions, commit
  or push in this continuation. Test-suite results from `66dc9a1` remain prior
  evidence, not a fresh full-suite execution in this documentation-only slice.

### 2026-09-07 — repeat update fails visible acceptance despite confirmation

- **Measurement, visible:** user explicitly reports «ні, досі "Мій e-paper
  dashboard"». The requested «Повторне USB-оновлення» / «Кадр №2» did not appear.
  This resolves the preceding pending observation negatively; keep the original
  successful first-image checkpoint, not the second protocol-only result.
- **Fact:** the last attempt returned `screen=confirmed`, exit0 using the same
  accepted native UF2/sender and the new hashed HTML recorded above. No extra
  submission or reflashing between completion and this user report.
- **Unknown:** whether the controller received the intended SPI bytes, whether
  the actual refresh ran, and the physical cause. A protocol success alone
  cannot choose between lifecycle/state, GPIO/contact, controller or panel.
- **Decision/guard:** stop WPA3 progression and blind frame retries. Audit the
  exact confirmation-to-Commit/panel lifecycle and repeated-cycle tests first;
  no new flash, reset, wiring instruction or image during initial diagnosis.
  Fresh Hello/Health cannot reconstruct the closed client's private transaction
  evidence and must not be presented as a readback of its frame.
- **Fact, source audit:** `cmd/epaperscreen` reads the requested path after
  readiness and renders it without an HTML cache. `screenclient.Send` sends
  both passes and Commit; `streamrx` checks each full pass digest before Ready,
  then calls the physical sink before Complete. Fresh lease acquisition clears
  old completion proof. This source trace is not a trace of the physical run.
- **Fact, lifecycle:** `panel.Stream.Begin` calls `Driver.start` for each new
  transaction (HAT power cycle, hardware reset and initialization); Commit calls
  refresh and sleep, then releases power. No obvious second-cycle wake omission
  found. `waitIdle` accepts immediately HIGH after the documented command delay;
  the pinned official `EPD_7in5_V2.c` likewise does not require observing LOW.
  Do not add an undocumented mandatory LOW edge or change timings blindly.
- **Fact, observability gap:** `panel.IO.Observe` can report Samples/LowSamples,
  but `pico2w.PanelIO` does not install it. EPS2 exposes panel diagnostics only
  for errors; successful waits are not retained. A constant HIGH signal can
  therefore finish in software without distinguishing controller activity.
  This is an evidence limitation, not proof that BUSY was stuck on this run.
- **Unresolved historical difference:** the extra reset-release BUSY gate in
  experiments10/11 (entry8 above) is not in the current official-derived
  `panel.start`. The retained EPS1 source also uses that same panel package.
  Preserve this difference for targeted diagnosis; it is not a demonstrated
  new regression or authorization to rewrite the accepted lifecycle.
- **Sources:** unchanged local official driver at
  `experiments/05-waveshare-official-c/vendor/waveshare/EPD_7in5_V2.c`, pinned
  `c9bcd84db5adf5f085353649a8a5c31492bc5fb8`; current `SPEC-panel-driver.md`.
  Waveshare's live manual returned403; search-indexed official manual confirms
  the sleep/power-off recommendation, but no extra timing claim is based on it:
  https://www.waveshare.com/wiki/7.5inch_e-Paper_HAT_Manual.
- **Measurement, independent software characterization:**
  `screenusbhost/TestFreshEPS2ClientsRefreshDifferentFramesOnPersistentPanel`
  passes for both alternating LOW/HIGH and permanently HIGH BUSY. Two fresh
  clients (480-byte cap) send different 800x480 frames through real codec,
  persistent `waveshare75`/session/driver code into recording IO. Each frame
  produces exact inverted/original 48,000-byte planes, the complete command
  sequence, its own refresh and final power-off. Fresh lease Query does not
  inherit the preceding completion. Fake time preserves the 180-second floor.
  This bypasses the host Sender/proxy, DTR, OS serial and physical pins/pixels.
- **Inference boundary:** this test passed immediately; it did not reproduce
  or fix the physical failure. Keep it as a repeated-cycle/evidence-boundary
  regression guard, not proof the transport or hardware is healthy. Package
  lint reports zero issues. Main reviewed both test files; no firmware or
  production behavior changed. Next target evidence must separate actual BUSY
  activity and delivered PWR from a software-only successful write sequence.
- **Verification:** existing-container `GOTOOLCHAIN=go1.26.8
  ./scripts/quality.sh task` PASS in21s; total coverage91.7%, changed100.0%
  (no changed production lines). Report:
  `experiments/03-remote-epaper/build/repeat-panel-diagnostic-quality.log`.
  Software build outputs were produced, but nothing was flashed or sent to
  the Pico in this diagnostic turn. No commit/push or security-policy change.

### 2026-09-07 — flashing activity was not observed

- **User report:** «не спостерігав, але думаю що ні» when asked whether the
  panel flashed during the failed repeat update. Visible flashing is UNKNOWN,
  not an observed absence. The confirmed observation remains the old title
  «Мій e-paper dashboard» after the attempt.
- **Decision:** do not turn this uncertainty into a diagnosis of missing power,
  a stuck BUSY wire or failed controller. No new send/reset/flash or pin change.
- **Next diagnostic boundary:** a bounded retained USB report should capture
  the attempted refresh stage, BUSY sample/LOW counts and elapsed stage time.
  Preserve the existing lifecycle and success criteria while observing it;
  no invented mandatory LOW edge or blind retry. Distinguish commanded PWR
  state from voltage physically delivered to the HAT: firmware alone cannot
  verify the latter or optically confirm pixels. Preparing and validating that
  instrumentation is the next step; it is not installed on the Pico yet.

### 2026-09-07 — retained panel-cycle telemetry candidate

- **Changed variable:** optional cached EPS2 PanelTrace opcode12 and observer
  wiring in waveshare75. No panel commands, GPIO mapping, timing, cadence,
  pixel polarity, credentials or storage layout changed. No full-frame RAM.
- **Fact:** trace contains boot-local cycle/state, last phase/step, total cycle
  elapsed milliseconds, actual read/LOW counts for the existing three BUSY
  waits. Errors retain counts, even without BusyDone; reads outside waits are
  not included. Snapshot performs no GPIO, clock call, Tick or lease acquisition.
  Opening USB still has normal DTR/priority effects. No mandatory LOW edge.
- **Evidence boundary:** a completed trace is not SPI receipt, voltage at HAT,
  or visible pixels. It cannot recover telemetry from the already-failed run.
  Active elapsed is0; stopped duration includes upload, not per-stage duration.
- **Verification:** TDD failures preceded wire/provider/client/CLI/adapter
  implementation; independent recorder tests cover errors, synchronous Begin,
  abort, cached reads, saturation/regression and transparent IO. Two full-size
  cycles retain their own BUSY counts while preserving exact plane/command
  assertions. Focused race tests PASS. Quality task PASS26s, changed92.8%,
  total91.7%; subsequent direct client tests PASS. Read-only independent review
  found no required changes. This repairs observability, not the physical fault.
- **Candidate:** `experiments/03-remote-epaper/build/screen-device-panel-trace.uf2`,
  SHA256 `65ce3ced1e05b854a33ece3077791ec4f57b6707aabe49fdb91f2623e18fc66a`.
  TinyGo0.42.0, Go1.26.8, pico2-w, tasks scheduler. Flash700616,
  headline RAM10108 (not peak); flash is2432 bytes above accepted baseline.
  Sender `build/epaperscreen-panel-trace-darwin-arm64`, SHA256
  `98754795c1e6d4f31fae869bedcf948125db5fc99910a808349cc84ab3af58d9`.
  Both built in existing container; prior accepted artifacts preserved.
- **Preflight:** native macOS Hello/Health returned profile1/version1,800x480,
  two passes, chunk1000, minimum180000ms, remaining0, networkdisabled,
  uptime5977s; exit0. No frame was sent in this preflight.
- **Next:** one direct software BOOTSEL request; flash candidate only after
  verified RP2350 identity. If unavailable, stop for manual intervention.
  Then inspect unavailable trace before one controlled upload. Do not declare
  the previous optical failure resolved from protocol telemetry.
- **Sources:** unchanged pinned Waveshare sequence cited above; additive wire
  contract in `experiments/03-remote-epaper/docs/eps2-wire.md`. No commits/push.
- **Final software gate:** after adding direct client-package coverage, quality
  task PASS21s, changed96.1%, total91.8%. Report `build/panel-trace-quality.log`.
- **Flash:** native 1200-baud request exited0, INFO_UF2 identified Raspberry Pi
  RP2350; one direct `cp -X` of the candidate exited0. This proves copying,
  not boot, protocol or visible output. Runtime inspection follows separately.
- **Runtime measurement:** CDC1101 returned; first `--panel-status` exited0:
  version1/state0/cycle0, all counters0, no automatic panel cycle on boot.
  One controlled send of unchanged `build/native-repeat.TeMKNU/dashboard.html`
  (SHA256 `6ff8dc0c9452f9b1d1b558279b68224398b8d26df836fa37f271cabf8d1d6d93`)
  then returned `screen=confirmed`, exit0. The normal 180-second fresh-client
  floor remained active. No automatic replay or second upload was performed.
- **Measurement by14:52:15 UTC:** subsequent cached trace exited0:
  state2/completed, cycle1, phase6/power-off, step17/deep-sleep, total2054ms;
  power-on samples1/LOW0, refresh samples1/LOW0, power-off samples1/LOW0.
  The measured2054ms is the physical sink cycle including upload, not the
  client's preceding cadence wait and not solely panel refresh duration.
- **Inference boundary:** firmware observed immediate HIGH at each existing
  BUSY wait after command delays. This is now target evidence, not the previous
  host-only constant-HIGH characterization. It still does not establish what
  happened during unsampled delays, HAT power delivery, SPI receipt, wiring
  fault or visible pixels. Do not silently change the success condition.
- **Manual boundary:** ask whether «Повторне USB-оновлення» / «Кадр №2» is
  visible. Until answered, optical acceptance remains pending and no more
  images, pin changes, erase or Wi-Fi enrollment are attempted. Sender and
  inspection processes ended; no background flash/send remains active.

### 2026-09-07 — unchanged trace and user-requested repeat

- **User observation:** the diagnostic send left the old dashboard visible.
  HAT VCC measured about3.2V; PWR reportedly stayed0 during the subsequent
  attempted send. Its returned trace remained cycle1/2054ms, exactly as before.
- **Unknown:** a new physical sink cycle was not established. Neither the
  sender's success message nor this voltage reading proves a wiring fault.
  Investigate freshness of the USB result and lifecycle before further wiring
  changes. The user was told to remove probes and leave wires unchanged.
- **User request:** send another image now, using the previous direct USB path.
  No firmware, wiring, timing or software changes. Pre-send cached trace again
  reports state2/cycle1/2054ms, all three waits samples1/LOW0.
  Use the preserved panel-trace sender and native-repeat HTML identified above;
  one attempt only, then compare trace and ask for visible acceptance.

- **Result of requested repeat:** sender exited0 with `screen=confirmed`.
  Post-send cached trace now reports state2/cycle2/phase6/step17/2054ms;
  all three waits samples1/LOW0. Unlike the earlier unchanged report, cycle
  advanced from1 to2. This establishes a new recorded sink lifecycle, not
  electrical delivery or optical acceptance. User observation is pending.
  No additional send, reflash or wire change was performed.

- **Optical result:** user confirms «Мій e-paper dashboard» still visible
  after cycle2. New image not accepted despite recorded sink completion.
- **Next user-requested repeat:** user moved wires slightly and requested the
  same image again. No software/firmware change. Before sending, direct USB
  trace remains cycle2/2054ms, three waits samples1/LOW0. One send launched;
  user asked not to move wires during powered transfer.
- **Repeat result:** sender exit0/`screen=confirmed`; trace advanced2→3,
  state2/phase6/step17/2054ms, three waits samples1/LOW0. New recorded sink
  cycle confirmed; visible output remains pending user observation. No retry
  or background sender remains active.

### 2026-09-07 — source audit after third optical failure

- **User observation:** cycle3 still left «Мій e-paper dashboard» visible.
  Moving wires did not yield a visible update; it does not establish which
  connection, if any, is faulty. No hardware operations in this audit.
- **Artifact verification:** SHA256 of both first-accepted binaries and both
  panel-trace binaries matches the full hashes recorded above. Checked in the
  existing Dev Container. These files are available for a controlled baseline
  comparison, not a guarantee that they will reproduce the earlier success.
- **Source facts:** `pico2w.PanelIO` configures GP15 as output, starts it LOW,
  and maps SetPower directly to GP15.Set. Every `panel.Stream.Begin` calls
  resetPower/start; Commit calls finish and releases power. No omitted
  second-cycle initialization was found. Trace wrappers retain GPIO callbacks.
  Host runDelivery reads/renders the selected HTML after readiness, then sends
  both planes; trace cycles2 and3 establish new sink executions, unlike the
  earlier ambiguous unchanged cycle1 report.
- **Source limitations:** HIGH-only BUSY can complete in software without
  establishing SPI receipt. Current waitIdle reads immediately and returns on
  HIGH; pinned official C EPD_WaitUntilIdle delays5ms before each read and5ms
  after completion. Its misleading LOW comment conflicts with actual code,
  which waits for HIGH. Thus do not call the timing byte-for-byte identical.
  Both implementations lack a mandatory observed LOW edge. The historical
  experiments10/11 post-reset readiness gate is another known difference;
  neither difference is established as the present cause or newly introduced.
- **Verification:** existing tests PASS for panel, paneldiag, screenclient,
  screenlink, screenusbhost and waveshare75 with Go1.26.8 in running container.
  The repeated physical-sink host test excludes real Sender/proxy/DTR/macOS;
  separate tests cover those software pieces, not the complete real hardware
  boundary. Do not claim comprehensive target proof from the green suite.
- **Next controlled plan:** (1) restore the checksum-verified first-accepted
  firmware/sender pair and send the distinct second HTML without moving wires;
  treat reboot as part of this baseline intervention, not proof of a code fix.
  (2) If first update succeeds, repeat with another distinct frame to test
  lifecycle persistence. (3) If it fails, stop blind uploads; prepare bounded,
  stage-correlated GPIO/reset/BUSY evidence and only then request an observable
  PWR measurement at both endpoints. Do not use the existing endless
  `power-probe` loop without a separately reviewed bounded diagnostic plan.
  Do not alter reset waits or assume wiring failure before these boundaries.
- **Sources:** local pinned official EPD_7in5_V2.c, current panel/lifecycle.go,
  panel/stream.go, pico2w/board.go, paneldiag/trace.go and screenusbhost code.
  Live raw GitHub fetch failed (cache miss); official comparison used the
  preserved pinned vendor source, not an asserted fresh upstream revision.
- **Scope:** documentation only; no firmware/code changes, flash, USB send,
  credential operation, commit or push during this audit.

### 2026-09-07 — restore first-accepted baseline

- **User authorization:** continue the documented baseline comparison.
- **Flash facts:** direct native1200baud request exited0; INFO_UF2 identifies
  Model Raspberry Pi RP2350 / Board-ID RP2350. One `cp -X` of preserved
  `screen-device-tinygo-0.42.0-recovery.uf2` exited0. No erase, rebuild or
  rewiring. Artifact SHA256 verified in the preceding audit.
- **Runtime facts:** original `epaperscreen-usb-rx-480-darwin-arm64 --status`
  returned profile1/version1,800x480,passes2,chunk1000,minimum180000ms,
  remaining176389ms,uptime3s,networkdisabled. This confirms a fresh boot and
  EPS2 response, not visible output. Diagnostic trace UF2 is no longer installed.
- **Next:** one send of native-repeat.TeMKNU/dashboard.html with this exact
  original sender, unchanged cadence. No automatic resend; optical result pending.

- **Baseline send result:** exit1, `screen outcome unconfirmed; no automatic
  replay was attempted`; `operation=4 code=11 state=4 pass=0 offset=0 phase=2
  step=6 command=04 busy_known=true busy=false`. Source mapping: CodeHardware,
  PhasePowerOn/StepControllerPowerOn; driver waits for BUSY HIGH after0x04,
  returning ErrBusyTimeout when it stays LOW until its10s deadline. Failure
  occurs before pixel-plane upload, not during HTML rendering or final refresh.
- **Post-failure:** original sender --status exits0, uptime235s, network0,
  remaining0. USB/runtime remains responsive. No automatic replay or pending
  sender. Restored baseline remains installed.
- **Inference limit:** unlike prior diagnostic HIGH-only cycles, this run
  observed LOW at controller power-on. Baseline restore also rebooted the MCU;
  cannot attribute the difference solely to the binary, wires or controller.
  Does not prove power delivery or SPI receipt. Next investigation should
  isolate reset readiness and driven signals before another image attempt.

### 2026-09-07 — bounded existing pin-isolation diagnostic

- **Authorization:** continue until manual intervention is required; user
  confirms wires were not moved during baseline restore. No rewiring planned.
- **Question:** does BUSY read LOW before any SPI command, after power/reset?
- **Reuse:** unchanged cmd/spi-startup-probe, built with TinyGo0.42.0,
  pico2-w/tasks in existing container. UF2 build/spi-startup-isolation-0.42.uf2,
  SHA256 e9aa83bbfba66fb2002d53415951443985c5406885142d93106f544c7c649657.
  It performs one bounded GPIO sequence, powers HAT off, then repeats only
  cached text. No SPI commands, pixel transfer, Wi-Fi or flash storage erase.
- **Limits:** this is an isolation sequence, not the production lifecycle:
  RST begins LOW, PWR settles250ms, reset LOW20ms, and CS/DC/CLK/DIN outputs
  are enabled sequentially. Results apply to those conditions; no direct
  causality claim about the full EPS2 firmware. Counts observe discrete10ms
  samples, not all electrical transitions or delivered voltage.
- **Reader:** reused unchanged experiments/04-waveshare-reference/diagread,
  built for macOS/arm64 in container. It expects DONE, absent in this probe;
  timeout after65s is a reader sentinel mismatch, not a hardware failure.
- **Next:** verified software BOOTSEL, exact UF2 copy, then capture one cached
  report. Preserve baseline UF2; no automatic refresh retry.

- **Flash/runtime result:** software BOOTSEL and verified RP2350 copy exit0;
  native diagread captured repeated identical cached reports. BUSY HIGH/LOW:
  PWR-high50/0, RST-low20/0, RST-released100/0, after CS/DC/CLK/DIN20/0 each.
  Firmware reports HAT power off. Repeated lines are cached output, not repeated
  power cycles. No displayed image change is expected from this probe.
- **Boundary:** all sampled input levels were HIGH under this isolation
  sequence, including RST held LOW. This does not prove actual RST/PWR voltage
  at HAT or receipt of commands (none sent). Do not claim that0x04 uniquely
  caused the earlier LOW: boot and initialization conditions differ.
- **Manual boundary:** further isolation requires measured driven levels at
  both Pico and HAT, beginning with PWR during an explicitly synchronized,
  bounded test. Do not measure the now-powered-off HAT as if active. Ask user
  whether the multimeter is ready before preparing that measurement window.
  Current installed firmware is the pin-isolation probe, not the HTML receiver.

### 2026-09-07 — one-shot measurement window

- **User ready:** multimeter prepared; no probes on board yet.
- **Change:** existing cmd/power-probe endless5s alternating loop replaced
  with one DTR-triggered30s window per boot. PWR starts LOW; RST remains LOW;
  no SPI, display refresh, Wi-Fi or credential operations. DTR loss ends the
  window; reconnect never restarts it. Firmware checks deadline every10ms
  (scheduler latency applies); this is not an independent hardware failsafe.
- **Tests:** RED undefined powerProbe, then GREEN100% pure state coverage for
  no-reader/off, deadline, early disconnect and no replay. Host tests do not
  measure actual voltage. Full quality task PASS22s, changed92.2%, total91.8%;
  git diff --check clean. No production panel-driver changes.
- **Artifact:** build/power-probe-once-30s.uf2, TinyGo0.42.0/pico2-w/tasks,
  SHA256 f5cfa8e2c78031c407d7e05b30c88deef68ed43d6a9d85b2995d50e7e81953ff.
- **Boundary:** configure pins/hold-reset strategy reused from existing
  power-probe; only window control changed. Print commanded state, never claim
  measured HAT voltage. Native reader must not open until user ready.
- **Next:** verify RP2350 mount, flash once, leave CDC unopened until probes
  can be placed for the explicitly announced measurement window.

- **Installed:** INFO_UF2 confirmed RP2350; copy exit0. USB reader not opened;
  no measurement window deliberately triggered. Runtime acceptance pending
  opening the reader after user confirmation. Previous baseline preserved.

### 2026-09-07 — unsynchronized voltage oscillation, one-shot runtime check

- **User report before deliberate reader open:** about3.2→0→3.2→0 each second
  at the requested PWR/GND measurement. Exact contact points, ground continuity,
  timing and other CDC clients were not independently observed. Asked user to
  remove probes; do not infer a wire fault or old firmware from this alone.
- **Runtime check:** opened native diagread once with probes removed. Received
  `PWR commanded_high= true RST=LOW maximum_seconds=30`, then one false line
  and DONE. No repeating HIGH/LOW events observed. This matches the new probe
  application, not the old5s endless loop. GPIO voltage was not measured during
  this check; logs report requested output, not HAT readback or electrical proof.
- **Artifact:** retained UF2 hash rechecked and matches f5cfa8e…1953ff above.
  No flash, code change, wiring change or image transfer in this check.
- **Reader boundary:** DONE is CRLF; existing reader searches LF-only DONE,
  so it can report its65s timeout after receiving completion. That is not a
  firmware timeout or a second measurement window. Leave reader closed after
  capture. The one-shot window is now consumed; reconnect alone cannot rerun it.
- **Next manual boundary:** a new synchronized measurement should start at
  Pico GP15 relative to Pico GND, then compare at HAT, with a fresh test boot.
  Do not ask user to interpret now-LOW PWR as a failure. Oscillation cause
  remains unknown, including whether it reflects probe contact or power.

### 2026-09-08 — replacement Pico, baseline restore requested

- **Prior board measurement:** user reported stable3.2V at GP15 relative to
  Pico GND during the confirmed one-shot HIGH window. This applies only to
  that board/run, not voltage delivered at HAT.
- **HAT measurement attempts:** two reader sessions returned
  `device not configured` after HIGH. No voltage reading obtained. User later
  reported disconnect happens when red probe touches PWR on the HAT PCB,
  black on Pico GND, and confirmed DCV20/COM/V-ohms configuration. Exact cause
  unknown; stopped powered probing. No fine probes available. Unpowered
  continuity measurement was proposed but no result reported.
- **New user action:** replaced Pico (not HAT) and requested firmware plus
  image retry. Treat this as a new board; do not carry over GPIO measurements
  or assumptions about flash contents. User did not report changing HAT/panel.
- **Flash facts:** replacement appeared in BOOTSEL, INFO_UF2 Model Raspberry
  Pi RP2350 / Board-ID RP2350. This identifies chip family, not wireless variant;
  target remains the user's Pico2W project. First-accepted UF2 and sender hashes
  reverified against their recorded values in existing container.
  One direct cp -X of8b844bb0…b69b4a exited0; CDC1101 appeared. No erase,
  provisioning, rebuild or wire changes by assistant.
- **Preflight:** original sender --status launched; response pending.
  Do not send pixels before readiness; no image has been sent to replacement.

- **Replacement runtime result:** ordinary EPS2 --status ended with context
  deadline exceeded; no image sent. Read-only native epaperprovision diagnose
  then promptly returned code5/state0/generation0 and zero identity, rejected
  operation. Device is responsive to control protocol, not ready for EPS2.
  Source recovery path can withhold Lifetime after startup/storage failure;
  numeric response does not uniquely identify its underlying cause. Do not
  confuse this new boot boundary with the old board's panel BUSY timeout.
  No service-area erase or provisioning performed; flash contents of the
  replacement remain unknown. Native clients have exited.

- **Explicit recovery authorization:** user approved clearing replacement
  Pico's service area after warning about Wi-Fi/token/session loss. Invoked
  epaperprovision -port /dev/cu.usbmodem1101 -confirm-erase erase once.
  Exit0; response code0/state0/generation0, zero identity and empty settings.
  Recovery implementation erases and verifies four reserved tail blocks,
  not application firmware. No retries, full-chip erase or new enrollment.
  Old credentials/journal are not recoverable through this operation.
- **Next:** reboot required before reserving a new boot lifetime and trying
  EPS2. Successful erase is not evidence of a working display or resolved
  underlying startup cause. Request USB disconnect/reconnect without BOOTSEL;
  no image sent yet on replacement.

- **After user reboot:** original EPS2 --status exit0, profile1/version1,
  800x480,passes2,chunk1000,minimum180000ms,remaining158350ms,uptime21s,
  networkdisabled. Recovery no longer blocks EPS2 on this boot; underlying
  original recovery cause is not uniquely established by the successful erase.
- **First replacement-board image attempt:** original verified sender launched
  once with native-repeat.TeMKNU/dashboard.html and Europe/Kiev. No firmware
  or wiring change, no automatic retry, normal fresh-client180s floor retained.
  Transfer and optical result pending.
- **Replacement transfer result:** original sender exited0 with
  `screen=confirmed (verify visible pixels separately)`. No retry. This is
  protocol completion on the replacement Pico, not proof of changed pixels;
  ask user whether the distinct heading and frame2 are visible. Sender ended.

### 2026-09-08 — preserve Pico and add a separate Pi5/Pironman direction

- **Final optical result on replacement Pico:** user reports old dashboard
  despite protocol completion. The fault remains unresolved; no further blind
  send or positive physical acceptance claim. Replacement still has the exact
  first-accepted unified UF2, not the power probe or panel-trace candidate.
- **User decision:** retain existing Pico/TinyGo firmware; develop an additional
  direct GPIO/SPI Go version for Raspberry Pi5 in the user's Pironman homelab.
  This is a Linux application/backend, not firmware flashed onto Raspberry Pi.
  Preserve renderer/display module reuse without coupling Linux GPIO into Pico.
- **Before implementation:** identify exact Pironman variant and GPIO use,
  Pi5 OS/kernel and SPI access; verify40-pin orientation and HAT Rev2.3 support
  against official sources. Keep cooling/storage/homelab services intact.
  No SSH access, GPIO claim, service install or hardware connection performed.
- **Checkpoint request:** user explicitly requested commit and push of all
  accumulated diagnostic code, tests, reasoning, measurements and this direction.
  Do not describe the proposed Linux backend as implemented or display as fixed.

### 2026-09-08 — separate Pi 5 SPI5 overlay, offline only

- **Scope:** Pironman 5 and Waveshare 2.13inch e-Paper HAT Rev2.1 / V4,
  not the older separate 7.5-inch Driver HAT or Pico-specific 2.13 board.
- **User evidence:** kernel `6.18.39+rpt-rpi-2712`; GPIO devices 0/10/11/12/13
  with gpiochip4 pointing to gpiochip0; SPI devices 0.0/0.1/10.0, no SPI5.
  Permissions are root:gpio 0660 and root:spi 0660 respectively.
- **Fact:** official 6.18 branch retains GPIO13 in the SPI5 pin group and
  GPIO12 as stock CS. Base Pi 5 DTS labels SPI10 as bootloader EEPROM SPI.
  Existing SPI0/SPI10 are not candidates for display probing.
- **Changed variable:** project-only candidate overlay replaces SPI5 pinctrl
  with GPIO14/15 and kernel-owned CS16. No Pi configuration or wiring changed.
- **Software verification:** dtc compile plus fdtoverlay on a minimal fixture
  and resolved-property tests passed; SPI0/SPI10 fixture properties preserved.
  DTC 1.7.2 installed only in the current Dev Container, package persisted in
  Dockerfile. No container rebuild claimed.
- **Unknown:** matching installed base DTB, active overlays, actual line
  ownership, HAT electrical acceptance and visible display operation. No
  fixture or compile result is evidence of an electrically safe connection.
- **Guard:** tests must reject reintroducing the old GPIO13 pinctrl reference
  or CS12. GPIO16 must have one owner: kernel SPI, never simultaneous GPIO
  userspace control. No deployment until target review and explicit approval.
- **Artifacts/sources:** `experiments/03-remote-epaper/deploy/pi5/README.md`
  records source links, tests and remaining acceptance steps.

## 2026-09-08 — Pi 5 local service software completed

- **Scope:** separate ordinary-Go Linux ARM64 service for the photographed
  Waveshare 2.13inch e-Paper HAT Rev2.1/V4. Pico and 7.5-inch firmware untouched.
- **Decision:** user deferred physical/DTB checks until software was ready;
  implementation continued without repeated manual measurement requests.
- **Fact:** `localdisplay` reuses the native engine, bounds HTML/PNG decoding,
  reserves the timestamp corner, rotates once, serializes full updates, applies
  restart/completion cooldown, and exposes authenticated Unix HTTP status.
  `cmd/epaper-local` composes Linux GPIO/SPI, private credential and shutdown.
- **Review correction:** admitted invalid/cancelled inputs originally left
  the old attempt in status. `finishAttempt` now records a fresh ID and safe
  outcome; snapshot copies prevent caller mutation. Tests cover this boundary.
- **Verification:** actual Unix HTML/PNG POSTs, socket replacement, body-read
  disconnect, disconnect during hardware, subprocess SIGTERM and blocked-device
  shutdown timeout pass against explicit test devices. Full module race/vet
  and task gate passed; changed coverage 92.9%, total 91.9% in the final gate
  before the extra Unix PNG test. All retained TinyGo builds passed.
- **Security:** govulncheck v1.1.4 found no reachable known vulnerabilities in
  `cmd/epaper-local`; pinned in tools/go.mod. No credentials/body are logged.
  Independent review closed all required findings. No commit/push/deploy.
- **Unknown:** electrical/real-DTB acceptance, live Pi systemd sandbox, Docker
  permissions and visible pixels. `systemd-analyze` is absent in the development
  container; policy regression tests are not a substitute for target verification.
- **Guard/artifacts:** module `deploy/pi5/SERVICE.md`, package in `build/`,
  driver/adapter/service test suites and the ARM64 build in `quality.sh`.
  Do not describe mock `controller-complete` as successful physical output.

## 2026-09-08/09 — first Pi 5 full frame and live-update requirement

- **User measurements:** offline fdtoverlay merged the candidate with installed
  Pi5 DTB without errors. After custom overlay installation/reboot,
  `/dev/spidev5.0` exists; GPIO14 SPI5_SIO0,15 SPI5_SCLK,16 output HIGH,
  GPIO13 remained input. User reports Pironman appears normal, not an exhaustive
  peripheral test. No SPI0/SPI10 reassignment.
- **Physical mapping used:** HAT VCC -> Pi3V3 physical1; GND ->6; DIN ->
  GPIO14/TXD physical8; CLK ->GPIO15/RXD physical10; CS ->GPIO16 physical36;
  DC ->GPIO22 physical15; RST ->GPIO23 physical16; BUSY ->GPIO24 physical18.
  This is the eight-wire2.13HATRev2.1/V4, no PWR, not7.5Rev2.3 or Pico.
  The photo/case orientation caused confusion: labels, not top/bottom prose,
  identify pins. GPIO16 is not physical pin16. No powered rewiring authorized.
- **Software observations:** ARM64 binary starts; Unix API401 without token,
  authenticated idle status. An unavailable socket followed user stopping the
  service, not a proven network/driver defect. The first POST was refused by
  the conservative180s startup floor. Later user explicitly confirmed that the
  full HTML test works. This is optical acceptance of full mode only.
- **Requirement correction:** user needs frequent local terminal/status updates;
  the3-minute full-only floor is unsuitable. Transport is not the display-rate
  limit. Add a separate opt-in partial session, not full refresh every second.
- **Sources:** V4 specification revision4.0,p9 recommends full after5partial/
  fast operations: https://files.waveshare.com/upload/4/4e/2.13inch_e-Paper_V4_Specification.pdf.
  Official HEAD verified with git ls-remote:
  `a794fbc39656b0f93938d1ffb3fdc77eaed9e9fc`; exact EPD_2in13_V4.c uses
  Display_Base0x24/0x26 +f7, then Display_Partial0x24 +ff. No custom LUT.
- **Decision:** retain full-only constructor/CLI and preserve the old artifact
  `build/epaper-local-full-baseline`. New `-partial` accepts first base immediately,
  defaults1s aftercompletion,max5partials,fullage10m,idlesleep30s. Base invalid
  aftersleep/fault/restart. Clock stays at lastfull duringpartial. RAM upload
  remains4000bytes. Single ownership and bounded BUSY unchanged. HTTP429 remains
  explicit; no terminal emulator or queue claimed.
- **Software verification:** new tests failed before APIs existed, then
  passed. Full-only command sequence unchanged; exact base/partial payload,
  everyI/O fault, policy, idle/race/shutdown tests added. Independent different-
  model review found no required code defect; requested stronger payload tests
  were added. No commit/push/deployment, no Pi GPIO operation during this change.
- **Final software gate:** quality task PASS31s, changed93.8%, total92.0%;
  panel andlive policy100%. Full-module race andARM64help passed. Candidate
  `build/epaper-local-live` SHA256
  `6214aba3b5e8f6c5dc801aed856204acaf7d177a6f0d894a948a778507dbb49c`.
  Baseline retainsSHA256
  `41e1a7d7f4337eb0a44d8d27b93a40002f9cafd2d630d4a581cb8ad1bf5271b5`.
- **Unknown:** physical partial latency, ghosting, repeatedbase/partial/cleanup,
  idle-sleep/wake and measured energy. Do not call them verified from mock I/O.
  See `docs/pi5-service/SPEC-live-refresh.md` and operatorSERVICE.md.

## 2026-09-09 — Pi 5 V4 partial and idle recovery visually accepted

- **Scope:** Pi 5/Pironman 5, Waveshare 2.13-inch HAT Rev2.1/V4;
  existing SPI5 GPIO14/15/16, DC22/RST23/BUSY24 wiring unchanged.
  Operator installed `epaper-local-live` and started it in the foreground.
  Built default-partial candidate SHA256:
  `0b53161d083f3c70b6c51c32f082d1bc68b641ca0cac677cd1d1bcc6a5a17fcf`;
  target checksum was not independently read back.
- **Fact:** user confirmed READY -> RUNNING visibly changed. Result id2 was
  partial, 00:48:01.859667708 -> 00:48:02.537566362 +03:00.
- **Measurement:** ids4..8 were five partial completions (about0.66–0.67s),
  id9 full (about2.47s), id10 partial. User confirmed Step7 and the observed
  cleanup matched the logs. Controller reports alone were not the acceptance.
- **Fact:** after the requested idle period, id11 was full,
  00:51:16.242053969 -> 00:51:18.713039404 +03:00; user confirmed AWAKE.
  This accepts visible recovery after idle, not a measured sleep current.
- **Decision:** partial sessions are the CLI default; explicit `-partial=false`
  retains full-only fallback. Preserve five-partial guard, idle sleep, bounded
  BUSY and no overlapping writers. No firmware or wiring changes for systemd.
- **Unknown:** long-duration ghosting/wear, power consumption, systemd startup,
  reboot recovery and Docker client deployment. Next variable is only process
  supervision: stop foreground owner before starting the system service.

## 2026-09-09 — local Debian package, no new display firmware

- **Decision:** user requested one-command installation instead of manually
  copying binary, systemd and sysusers files. Native dpkg-deb package for ARM64;
  recipe and lifecycle tests: `experiments/03-remote-epaper/deploy/pi5/deb/`.
- **Fact:** payload copies the visually accepted `epaper-local-live` unchanged
  (SHA256 `0b53161d083f3c70b6c51c32f082d1bc68b641ca0cac677cd1d1bcc6a5a17fcf`).
  Unit path alone changes to `/usr/bin/epaper-local` to avoid package ownership
  under `/usr/local`. No token, overlay, boot config or hardware commands added.
- **Guard:** install does not enable/start; local `/etc` unit/sysusers shadows
  cause early refusal. Stop foreground ownership before enabling system unit.
  Credentials migrate separately and survive reinstall/remove/purge.
- **Verification:** actual dpkg installation/reinstallation/purge in a temporary
  root passed, with systemd-sysusers mocked and dependency checks bypassed only
  in that isolated test. Archive identity, private-data preservation, shadow
  refusal and byte-identical repeat packaging tests passed. Independent review
  found no required lifecycle/security defect. This is not Pi/systemd acceptance.
- **Artifact:** `build/epaper-local_arm64.deb`, version0.1.0~local.20260909.1,
  9078336bytes, SHA256
  `16bc0f897b626ed4f4c79b4d0f7f9186beae700855da9a9beba5a0ec63665bae`.
- **Unknown:** real APT dependency resolution, system account creation,
  systemd permissions and reboot startup remain operator-side checks. No
  deployment, GPIO write, commit or push performed by the agent.
- **Software gate:** task PASS44s, total coverage92.0%; final fast gate PASS13s
  and package race tests passed. No limits or assertions weakened.

## 2026-09-09 — correct systemd credential integration and Debian lifecycle

- **Fact:** package installed on Pi; service failed before hardware startup with
  `credential must be a private regular file`. Our ordinary-file permission
  guard rejected group bits, incompatible with systemd ACL masks. The exact Pi
  ACL has not been inspected; this documented incompatibility was reproduced
  using a real named-user ACL in container tests.
- **Fix:** native CREDENTIALS_DIRECTORY/token source; systemd controls access,
  Go checks regular file, bounded size/format and rejects a final symlink. An
  explicit token-file still uses strict private mode. No chmod workaround.
  `-check-credential` exits before renderer, socket and GPIO, without token output.
- **Package:** version0.1.0~local.20260909.2 uses debhelper compat13,
  dh_installsysusers and dh_installsystemd --no-enable --no-start. Generated
  preinst stops on upgrade. A minimal remove-stop hook is necessary because
  compat13 omits prerm stop with --no-start. Local override protection retained.
- **Verification:** real dpkg and sysusers in isolated root, ACL regression,
  ordinary-file rejection, no-GPIO credential probe and archive repeatability
  tests passed. Task gate PASS31s before final generated-hook assertion.
  Container dependencies persisted in Dockerfile and installed in the existing
  container; container was not rebuilt/replaced. PID1 is sh, not systemd.
- **Artifact:** build/epaper-local-systemd SHA256
  `9fbe51038c004ce01cb598c32938214ab9a7ff16459b0fd80312a7bd520ee776`;
  build/epaper-local_arm64.deb SHA256
  `0472e1aca9e73d509aa88407747203cf2e81da2fd43facdb6655ac8920982761`.
  Prior standalone live binary preserved. Driver/pinmap unchanged.
- **Remaining gate:** real systemd LoadCredential test on target using packaged
  `-check-credential`, then normal start and reboot acceptance. No live PID1
  result is claimed from isolated dpkg or ACL tests. No target deployment here.
- **Sources:** https://systemd.io/CREDENTIALS/ and systemd issue29435;
  Debian trixie dh_installsystemd/dh_installsysusers manuals; full links and
  target probe command in deploy/pi5/deb/README.md.

## 2026-09-12 — ESP32 Rev3 diagnostics and requested 7.5 V2 comparison

- **Fact:** user permits C and WPA2 for the new ESP32 port only. Existing Pico
  and Pi5 implementations remain separate. Small panel photo E154A79N204Q02/V2
  matches the reported 1.54-inch B V2 tricolor profile; optical acceptance absent.
- **Measurement:** direct macOS USB bridge `/dev/cu.usbmodem5B140746091` worked;
  standalone upstream espflasher v0.8.1 detected ESP32/4MB, wrote the diagnostic
  at 0x1000 without erase-all, verified MD5 6481b3a5db96b3bf329f952edf2050bf,
  then reset. Existing container could not see this serial device.
- **Fact:** operator reported A/B=B and ON. UART command `t` returned
  `CONTROLLER_COMPLETE: left black, right red, white border; confirm visually`.
  Operator saw no pattern. **Unknown:** panel health, ribbon contact and actual
  electrical drive; controller-complete is not proof of a working or broken panel.
- **Decision:** user requested the previously used Waveshare 7.5-inch 800x480
  monochrome V2 as a comparison. New `epaper75` build profile reuses the existing
  `panel.Driver`; it does not reinterpret the tricolor command set or rewrite
  Pico. This changes panel plus matching profile, not an isolated proof of cause.
- **Guard:** one explicit `t` per boot, no automatic refresh/retry, bounded BUSY,
  command/phase/sample diagnostics. A 48000-byte frame is reused for both planes.
  GPIO4 power cutoff remains unverified (Rev3 schematic R35 NC); the supply
  callback reports manual power, and failure requires physical disconnection.
- **Verification:** new trigger/pixel/failure tests passed (100% statement
  coverage), existing panel and tricolor tests passed with race detector.
  Task gate passed in 29s, changed coverage91.2%, total92.1%, including prior
  Pi5 changes and hardware-only lines as uncovered. Both ESP32 profiles build;
  preserved Pico build gates pass. Small-image artifact is not overwritten.
- **Artifact:** `experiments/12-esp32-epaper/panel75-check.bin`, SHA256
  `70b0d5f9b6b4b333c6af604e0c0df18fc9372ec1cc147d05570bf1dbd5b17908`.
  Compiler reports flash16280/staticRAM4712; this excludes heap framebuffer.
  Investigated unexpectedly smaller image: ELF includes main, Check.Handle,
  Driver.Refresh, writePlane, waitIdle and SPI Tx; size alone is not runtime proof.
- **Correction:** TinyGo 0.42.0 `tinygo flash` itself calls full-chip EraseFlash.
  Its earlier suggested use to preserve credentials was incorrect. Use the
  standalone pinned flasher without erase-all; README now explicitly warns.
- **Remaining:** 7.5 profile not yet flashed or visibly accepted on ESP32;
  confirm power-off panel replacement and exact board switch mapping before
  sending `t`. Full secured remote-display ESP32 firmware remains unfinished.
- **Sources:** `docs/esp32-port.md`, experiment12 README; unchanged vendor
  `experiments/05-waveshare-official-c/vendor/waveshare/EPD_7in5_V2.c` at
  Pico_ePaper_Code c9bcd84db5adf5f085353649a8a5c31492bc5fb8. ESP32 Wiki currently
  returns403: do not substitute old separate-HAT switch meanings.

## 2026-09-12 — ESP32 7.5-inch diagnostic visibly accepted

- **Fact:** exact experiment12 panel75-check.bin SHA256
  `70b0d5f9b6b4b333c6af604e0c0df18fc9372ec1cc147d05570bf1dbd5b17908`.
  ESP32 Driver Board Rev3, previous 7.5-inch V2 monochrome800x480 panel.
  SPI2 mode0/1MHz, CLK13/DIN14/CS15/DC27/RST26/BUSY25; SDI34 unused.
- **Measurement:** operator confirmed display selector A/ON and USB connected.
  Direct upstream espflasher wrote at0x1000 without erase-all, verified
  MD5 d4ad4b93fa5d1947e5ad600aae3327b2 and reset. Firmware reported
  `EP75-V2 CHECK v1 READY`. One t was sent; logs reached power-on BUSY wait.
  Final read output was not recovered; subsequent passive read was empty.
  Do not invent CONTROLLER_COMPLETE or successful sleep for this capture.
- **Measurement, optical:** user replied «є» to the explicit black rectangle
  left/on white question. This is the first accepted ESP32/7.5 visible frame.
  The small panel's failure does not establish that it is broken.
- **Correction:** second ON/OFF controls USB TO UART, not master power. Exact
  Rev3 label is ON:A OFF:B. An indexed wiki table and older schematics conflict
  on A/B resistance assignment. Record actual accepted A/ON, not a universally
  proven resistance mapping. Never transfer separate-HAT switch semantics.
- **Guard:** retain artifact and current assembly; one attempt per boot. No
  automatic repeat after the positive observation. No Wi-Fi, streaming,
  repeated-cycle, peak-memory or energy acceptance follows from this frame.
- **Next request:** user asked for the complete ESP32/7.5 receiver, reusing the
  extensive existing Go manager/protocol. Plan: module tasks/esp32-75.md.
  Fresh installed-source inspection confirms missing original-ESP32 Flash and
  nonpersistent espradio NVS shims. A new persistent backend needs explicit
  approval; no erase, provisioning or hardware operation occurred in this audit.

## 2026-09-12 — ESP32 persistent reservation approval and ROM-driver boundary

- **Fact:** user approved persistent storage after its purpose was explained.
  Do not ask for the same 16-KiB reservation approval again. Current target is
  still the complete ESP32/7.5 receiver with Go-manager EPS2/EPN2 delivery.
- **Change:** added portable `espflash` range guards and negative tests; recorded
  approval in experiment03 tasks/esp32-75.md. No native I/O calls or integration
  are present yet; the range guard alone does not protect an existing flasher.
- **Measurement:** failing-first tests, then `go test -race -cover ./espflash`
  passed at 100%; `quality.sh task` passed in 29s, changed91.4%, total92.1%,
  including preserved Pico and Pi5 build gates. No physical operation occurred.
- **Fact:** official ESP-IDF v5.4.2 commit
  f5c3654a1c2d2a01f7f67def7a0dc48e691f63c0 patches ESP32 ROM clear-BP behavior
  because a wrong high status byte can set protection bits. Direct ROM binding
  alone is therefore not an accepted native flash backend. Source and precise
  next-step boundary are recorded in tasks/esp32-75.md.
- **Unknown:** native flash persistence, cache/interrupt safety with espradio,
  UART session detection and full receiver resource fit remain unverified.
  ESP-IDF/C is a proposed alternative MCU runtime, not a silent replacement.
- **Guard:** preserve accepted diagnostic; no chip erase, keys, Wi-Fi or image
  update. Do not advertise these address checks as completed firmware.

## 2026-09-12 — approved ESP-IDF Python tooling, container configuration prepared

- **Fact:** user approved Python after explanation of the official ESP-IDF
  build dependency. Permission is limited to SDK tooling in the Dev Container;
  no Python server/device runtime, custom scripts, or host execution.
- **Change:** Dockerfile retains TinyGo/Pico, adds ESP-IDF v5.5.5 with commit
  b774170ff46c393eeb5e495ea37936038d3f4f4f verification, recursive submodules,
  documented prerequisites, ESP32-only tool selection and SDK version check.
  Root AGENTS.md and `.devcontainer/README.md` preserve scope and setup steps.
- **Verification:** official release, remote tag/commit, installer and package
  requirements inspected; `git diff --check` passed. Current container is still
  the old environment. No image rebuild, SDK install, Python execution or
  hardware command occurred. Runtime verification awaits the user's rebuild.
- **Guard:** do not label configuration review as an installed toolchain or a
  working receiver. Preserve accepted panel diagnostic and both Go applications.
- **Source:** https://github.com/espressif/esp-idf/releases/tag/v5.5.5 and
  https://docs.espressif.com/projects/esp-idf/en/v5.5.5/esp32/get-started/linux-macos-setup.html.

## 2026-09-12 — rebuilt ESP-IDF SDK verified by an upstream compile

- **Fact:** after earlier checks found no matching container, user opened the
  rebuilt project. Container e2b84686dae7 is running with the expected project
  label. ESP-IDF v5.5.5 reports the pinned b774170ff46c393eeb5e495ea37936038d3f4f4f.
- **Measurement:** official export/dependency check succeeds; Xtensa GCC14.2.0
  esp-14.2.0_20260121, Python3.13.5, Go1.26.8, TinyGo0.42.0 verified. Official
  Python tools were used inside the container under the recorded permission.
- **Measurement:** unchanged official hello_world builds for esp32 with isolated
  build/sdkconfig in /tmp/epaper-idf-check.63qyal; exit0, app0x21660 bytes and
  bootloader0x6610 bytes. Compilation is not device execution or pixel proof.
- **Guard:** generated example defaults to 2-MiB flash and is not our 4-MiB
  receiver; ignore its suggested flashing commands. No board write, erase,
  serial operation or Wi-Fi connection occurred. Accepted panel diagnostic stays.
- **Next:** actual storage/identity and protocol components; no completed ESP-IDF
  receiver exists yet. Reproduction is in `.devcontainer/README.md`.

### 2026-09-12 — native ESP32 storage and UART qualification

- **Fact:** new receiver staging project uses official ESP-IDF5.5.5 partition
  APIs; no ROM cache or interrupt workaround. Physical flash size must be4MiB,
  partition exactly0x3fc000/16KiB; normal app image does not overlap it.
- **Measurement:** C EPE1 journal tests passed520 reservations and rollover;
  independent ASan/UBSan review exercised committed-slot corruption, exhaustion,
  interrupted writes/erase and verification failure. Source tests are retained.
- **Measurement:** storage/UART application cross-compiles: app0x287e0,
  bootloader0x4790. No flash, UART command or panel operation was performed.
- **Review finding:** provisioning could acknowledge success despite a final
  reload that contradicted the requested mutation. Independent regression
  reproduced a no-op erase returning state1/code0 instead of state1/code3.
  Added authoritative postcondition checks, matching Go provision.Report.
- **Unknown:** complete EPS2/EPN2 transport, hardware storage durability,
  UART recovery, Wi-Fi and optical acceptance remain unfinished. The new build
  is qualification-only; it must not replace the accepted rectangle diagnostic
  as a claimed full receiver. Track remaining work in tasks/esp32-75.md.

### 2026-09-20 — integrated ESP32 receiver rechecked, not flashed

- **Fact:** preserved native receiver includes USB provisioning, EPS2 screen
  streaming, EPN2 encrypted outbound Wi-Fi, WPA2/CCMP admission and exact7.5 V2
  panel lifecycle. Pico/Pi5 paths are not replaced. Runtime is C/ESP-IDF under
  the approved exception; manager/renderer and host utilities remain Go.
- **Measurement:**16/16 ASan/UBSan test executables and Go-to-C interoperability
  tests pass again after reopening the existing Dev Container. Two full frames
  and provisioning lifecycle are verified against injected SPI/flash, not pixels.
- **Artifact:** `receiver/build-release/epaper_receiver.bin`, SHA256
  `084cc044e996ee70154665446c99ad6198bbe79f314e9158937a1cb31e5c85f4`,
  size0xbc7c0. Bootloader0x1000, partitions0x8000, app0x10000; reserved tail
  `0x3fc000..0x3fffff` is not a flash input. Host Go binaries are built.
- **Verification boundary:** prior full Go run passed lint, tests, race, vet,
  compilation, changed coverage91.0% and total92.1%. Full resource gate is NOT
  green: old Pico protocol reports heap at receiver.go:111 and codec.go:204.
  Same findings reproduced from untouched HEAD in an isolated temporary tree.
  Separate old HTML resource gate rejects an empty allocation report as
  malformed; this result is not accepted as a resource pass. No checker disabled.
- **Guards:** USB startup diagnostics survive unavailable storage/display;
  first panel failure survives cleanup, including BUSY sample attribution.
  Explicit ESP32 auth/IPv4 policy is enforced on requests and replies without
  weakening Pico's WPA3 policy. Independent startup/null-I/O tests are retained.
- **Unknown:** native firmware boot, Wi-Fi, USB bridge behavior, interruption
  recovery and visible panel output. No new host serial or flash command has
  run. A current port and host authorization are required for the next step.
- **Sources/runbook:** `experiments/12-esp32-epaper/receiver/README.md` contains
  architecture, limits, source links and source-verified direct multi-image
  Go-flasher command without whole-chip erase.

### 2026-09-20 — native ESP32 flashed; live EPS2 discovery confirmed

- **Authorization:** user explicitly approved direct macOS USB identification,
  flashing without whole-chip erase and sending a test image.
- **Measurement:** existing Go espflasher0.8.1 detected ESP32/4MiB at
  `/dev/cu.usbmodem5B140746091`. Multi-image writes succeeded at0x1000/0x8000/
  0x10000; each passed device MD5 verification. No erase-all, credential writes
  or Wi-Fi enrollment performed. Source artifact SHA256 remains the preceding
  entry; flasher applies flash-header parameters before device verification.
- **Device MD5:** bootloader `766a3f9198e969bf17f6f1a274b8d971`, partition table
  `f9487bda4b423ff4b5875cf0306fd355`, app `63c45c0eba9c2d2ce7547ba4e727c637`.
- **Observed exception:** first read-only EPCQ Inspect returned no data.
  Subsequent independent EPS2 Hello/Health succeeded without reflashing:
  profile1/version1,800×480,two passes,chunk1000,minimum180000ms,
  uptime75s,network state0/failures0. Initial Inspect failure cause is unknown;
  do not claim it proves hardware failure or a fixed UART startup issue.
- **Next:** `receiver/usb-acceptance.html` sent with existing macOS Go tool;
  delivery waits the advertised startup cooldown. Expected large label
  `ESP32 USB READY`, `TEST FRAME A / 2026-09-20`, corner timestamp.
  Completion and visual acceptance must be recorded separately.
- **Quality:** repeated task gate completed its checks but exceeded its accepted
  wall-clock budget:195seconds versus90, therefore exit1. Changed coverage91.0%,
  total92.1%; do not describe the timed gate as passing. Budget unchanged.

### 2026-09-20 — first integrated ESP32 USB frame acknowledged

- **Measurement:** direct macOS `epaperscreen-darwin-arm64` rendered
  `receiver/usb-acceptance.html`, waited device admission, delivered EPS2 and
  exited0 with `screen=confirmed (verify visible pixels separately)`.
- **Boundary:** this confirms the receiver/controller transaction, not visible
  pixels. User optical confirmation was requested. No retry or second frame
  was sent during the wait; Wi-Fi remains unprovisioned/unaccepted.
- **Unknown:** end-to-end elapsed time was longer than the initially expected
  remaining cooldown; no per-stage timing was captured, so do not attribute
  all delay to refresh, transfer or cooldown without further evidence.

### 2026-09-20 — integrated ESP32 USB image visually accepted

- **Fact:** user answered “так” to whether `ESP32 USB READY` and the time in
  the bottom-right corner were visible. This supersedes the preceding pending
  optical confirmation, not the unresolved initial Inspect exception.
- **Known-good checkpoint:** ESP32 Driver Board Rev3, original monochrome
  7.5 V2 800×480 panel, A/ON; CLK13, DIN14, CS15, DC27, RST26, BUSY25.
  No wiring change in this experiment. Integrated app SHA256:
  `084cc044e996ee70154665446c99ad6198bbe79f314e9158937a1cb31e5c85f4`.
  Boot/partition artifacts and verified write offsets are recorded above.
- **Measurement:** Go-rendered `receiver/usb-acceptance.html` sent directly
  through macOS USB using `epaperscreen-darwin-arm64`, exit0 and
  `screen=confirmed`; visible output independently confirmed by the user.
- **Decision / regression guard:** preserve this artifact and fixture as the
  integrated USB baseline; test Wi-Fi as a separate transport change. Do not
  rewire or reflash merely to configure the manager.
- **Unknown:** follow-up read-only panel-status output was not recovered;
  no diagnostic result is inferred. Wi-Fi provisioning, encrypted live delivery,
  interruption recovery, runtime power and repeated-update acceptance remain
  pending. Existing quality/resource-gate failures remain documented above.

### 2026-09-20 — macOS manager and USB Wi-Fi enrollment

- **Decision:** user chose the Mac for initial Wi-Fi acceptance and authorized
  LAN address discovery and native manager startup; router/firewall unchanged.
  Cross-built the macOS manager in the existing Dev Container. Private inputs,
  enrollment, API token and temporary localhost TLS certificate are under the
  ignored `.local-mac/` directory; no credentials belong in documentation.
- **Measurement:** first provisioning attempt returned no ACK. Preserved its
  `.pending` candidate. A separate read-only Inspect then reported canonical
  success, blank state0/generation0. A new attempt with a separate registry
  succeeded: code0/state1/generation1/auth2. No ambiguous candidate was promoted
  or deleted; no credential erase or whole-chip flash occurred.
- **Measurement:** native manager launched with Wi-Fi screen transport, trusted
  HTML, a LAN-only device listener and loopback-only HTTPS API. Firmware network
  fencing after provisioning requires restart before live Wi-Fi acceptance.
- **Unknown:** cause of the first missing ACK remains unresolved. Enrollment
  success does not establish Wi-Fi association, encrypted-session success or
  a visible wireless update; those are the next acceptance boundaries.

### 2026-09-20 — first encrypted Wi-Fi frame controller-confirmed

- **Fact:** user confirmed USB power reconnection after provisioning. No USB
  frame command ran in this experiment; the existing native Mac manager used
  Wi-Fi screen transport and USB-created enrollment.
- **Measurement:** authenticated localhost HTTPS accepted the single
  `receiver/wifi-acceptance.html` scene as revision1. Manager device TCP socket
  became established. Status progressed to current1/confirmed1/delivered1,
  in_flight0 and refresh_trusted=true. Full-refresh cycle1 started at
  2026-09-20T19:32:27.014049Z and completed at19:32:33.317413Z.
  These timestamps bound the manager cycle, not a measured panel-only waveform.
- **Inference:** confirmed delivery through the configured EPN2 transport
  establishes successful authenticated/encrypted protocol interoperability on
  hardware. It is not a full security audit or optical acceptance.
- **Decision / guard:** preserve USB fixture A and distinct Wi-Fi fixture B
  (`ESP32 WIFI READY`). No duplicate scene submitted while awaiting delivery;
  keep the existing 180-second admission policy unchanged.
- **Unknown:** user optical confirmation of Wi-Fi fixture B is pending;
  interruption, wrong-key, recovery and long-running/power gates remain open.

### 2026-09-20 — Wi-Fi frame visually accepted

- **Fact:** user answered “так” to the expected `ESP32 WIFI READY`,
  `TEST FRAME B` and bottom-right time. This resolves the preceding pending
  optical check; both integrated USB and encrypted Wi-Fi paths now have a
  user-confirmed visible result on the exact 7.5 V2 panel.
- **Checkpoint:** same firmware artifact, board, wiring and A/ON switches as
  the USB checkpoint above; changed transport to the native macOS Go manager
  with USB-provisioned credentials. Fixture: `receiver/wifi-acceptance.html`.
- **Boundary:** preserve the first missing USB ACK as unresolved history.
  No claim of completed recovery/security audit, resource gates or endurance
  testing. No extra refresh or firmware write was needed for this confirmation.

### 2026-09-20 — manager restart: TCP reconnect without board reset

- **Measurement:** before stopping the manager, status had no in-flight frame,
  confirmed1/delivered1 and maintenance cycle2 completed. Sent SIGTERM only to
  the identified manager process; it exited0. Started the same binary with the
  same enrollment/TLS configuration and addresses. ESP32 established a new TCP
  connection without board reset, reflash or USB serial access.
- **Fact:** new manager GET returned an empty scene and a new epoch/revision0.
  The current screen manager keeps the scene in memory; it does not restore the
  HTML across process restart. An upstream producer must resubmit the scene.
- **Boundary:** TCP reconnect alone does not establish encrypted frame delivery.
  Next single-variable check uses a distinct recovery fixture through the new
  manager; preserve existing admission/cooldown and do not infer pixel changes.

### 2026-09-20 — encrypted frame confirmed after manager restart

- **Measurement:** submitted `receiver/wifi-recovery.html` once to the new
  manager epoch. Status reached current1/confirmed1/delivered1,
  refresh_trusted=true; cycle1 started at2026-09-20T20:32:00.5802Z and completed
  at20:32:08.160675Z. No board reset, reflash or USB frame command was used.
- **Verification:** `go test ./screenhub ./screenclient ./manager` passed in
  the existing Dev Container. This is scoped verification, not a full gate.
- **Boundary:** encrypted delivery after a graceful manager restart passed.
  User optical confirmation of `WIFI RECONNECTED` / `TEST FRAME C` is pending.
  This does not test interruption during transfer, lost ACK, wrong key, board
  power failure or automatic scene persistence (the producer resubmitted HTML).

### 2026-09-20 — manager restart recovery visually confirmed

- **Fact:** user answered “так, є” to `WIFI RECONNECTED` / `TEST FRAME C`.
  This closes the preceding optical check: graceful manager restart followed
  by automatic ESP32 reconnect and a resubmitted frame worked visibly.
- **Checkpoint:** same integrated firmware, panel, wiring and switch settings;
  only the Mac manager was restarted. Preserve fixture C alongside USB A and
  Wi-Fi B. No ESP32 reset, reflash or USB frame was used for this test.
- **Boundary:** scene resubmission was explicit, not persistent-scene recovery.
  Interrupted transfers, lost ACK, prolonged outage and board power failure
  remain separate unaccepted hardware scenarios. Do not generalize this pass.

### 2026-09-20 — ESP32 power-cycle recovery controller-confirmed

- **Fact:** user confirmed USB power disconnection/reconnection without buttons;
  the existing Mac manager stayed running. No credential write, reflash or USB
  serial command was used. A new device TCP connection appeared.
- **Measurement:** submitted `receiver/wifi-power-recovery.html` once as
  revision2. Manager invalidated the previous confirmation while recovering
  (confirmed0, delivered1, refresh_trusted=false), then reported
  current2/confirmed2/delivered2, in_flight0, refresh_trusted=true.
  Cycle2 started at2026-09-20T20:38:02.159263Z and completed at20:38:10.596745Z.
- **Inference:** stored Wi-Fi configuration and device credentials survived this
  idle power cycle; encrypted delivery resumed with the same enrollment.
  No claim about power interruption during flash writes or an active refresh.
- **Boundary / guard:** keep the distinct fixture D and initial recovery
  invalidation evidence; user optical confirmation of `ESP32 POWER RESTORED`
  / `TEST FRAME D` remains pending. Existing cooldown was not bypassed.

### 2026-09-21 — idle board power-cycle recovery visually accepted

- **Fact:** user answered “так” to `ESP32 POWER RESTORED` / `TEST FRAME D`.
  This closes the pending optical check for the 2026-09-20 power-cycle test.
- **Checkpoint:** same firmware, wiring, panel and manager enrollment; board
  power was cycled while idle, then the manager submitted one distinct frame.
  Stored credentials, reconnection and visible delivery passed this scenario.
- **Boundary:** not evidence for interruption during flash writes or refresh,
  lost-ACK handling, endurance or measured energy savings. No extra device
  command or refresh was issued merely to record this confirmation.

### 2026-09-21 — first USB request: UART deadline investigation

- **Scope:** investigate the intermittent first Inspect/provision no-response
  without touching the accepted board firmware, credentials or live manager.
- **Fact:** `ep_uart_read` checks its clock again after consuming all requested
  bytes. The USB idle loop uses it to read one byte and treats false as no data.
  Therefore a successful read followed by delayed task scheduling can discard
  the first byte before packet framing starts.
- **Source:** pinned ESP-IDF `b774170ff46c393eeb5e495ea37936038d3f4f4f`,
  `components/esp_driver_uart/src/uart.c`, `uart_read_bytes`: consumed ring-buffer
  bytes are copied before the returned count. Public contract:
  https://github.com/espressif/esp-idf/blob/b774170ff46c393eeb5e495ea37936038d3f4f4f/components/esp_driver_uart/include/driver/uart.h.
- **Hypothesis:** this can explain a lost first request, but no device trace yet
  establishes it as the cause of the observed ACK failures. A deterministic
  host regression against the actual UART adapter is being prepared.
- **Decision:** distinguish bounded idle-byte polling from strict full-record
  deadline checking. Do not add blind provisioning retries, relax authentication
  or change display timings. Hardware verification remains a separate gate.

### 2026-09-21 — UART idle-byte loss reproduced and guarded

- **RED:** native test compiled actual `platform/uart.c` against deterministic
  SDK-boundary stubs. A read consumed `E`, then resumed at20001us for a20000us
  deadline; the original adapter returned false (CTest exit8).
- **Change:** added `ep_uart_poll_byte` with a bounded20ms SDK wait and count-based
  success; the USB idle loop now uses it. Complete-record `ep_uart_read` retains
  its deadline check. No packet format, panel timing, credentials or retry
  policy changed. Firmware has not yet been replaced on hardware.
- **GREEN:** all17 native ASan/UBSan tests pass, including byte preservation,
  empty/error reads and strict late-record rejection. Go-to-C interoperability
  passes. Separate `build-uart-fix` candidate keeps `build-release` intact.
- **Build:** actual ESP32 candidate builds, app0xbc7d0, SHA256
  `891b38e755082b9d55d72bcca0a93027a304a2bbeebdf8c41f64487216feb5c7`.
  Candidate sdkconfig and partition table match the accepted build exactly;
  original app SHA256 remains `084cc044e996ee70154665446c99ad6198bbe79f314e9158937a1cb31e5c85f4`.
- **Unknown:** deterministic proof establishes the adapter defect, not the
  actual cause of each historical USB no-response. On-device first-request
  checks remain required. SDK multi-fragment wait semantics also mean the
  record deadline check is not a hard bound on time spent inside the SDK read;
  do not claim this focused fix resolves that separate timing concern.

### 2026-09-21 — UART candidate failed hardware acceptance

- **Precondition:** manager reported in_flight0/confirmed2/delivered2; stopped
  the identified manager gracefully before serial access. User authorized the
  candidate flash. Same board, wiring and enrollment; no key writes.
- **Measurement:** app-only candidate write at0x10000 completed with device MD5
  `9501302a98d83af8eed45d2ced02cbd6`. First Inspect returned
  `provision: invalid configuration` (not the earlier no-data timeout). A
  separate read-only Inspect returned the same error; EPS2 status returned EOF.
  No frame or provisioning mutation was sent.
- **Unknown:** invalid configuration is a host validation error, not proof that
  stored credentials are corrupt. No raw response/boot trace captured yet.
  The relationship to the UART change, boot output and serial framing is not
  established; the candidate must not be marked hardware-accepted.
- **Decision:** roll back only the app to the retained known-good build-release,
  preserving credentials/epochs. Retain regression tests and candidate evidence
  for diagnosis. Rollback write verification and runtime recovery follow below.

### 2026-09-21 — known-good app restored after UART candidate test

- **Measurement:** app-only rollback succeeded, MD5
  `63c45c0eba9c2d2ce7547ba4e727c637`, matching the previously accepted image.
  No credential/epoch sectors were flash inputs. Manager restarted with the
  same enrollment; no new scene submitted during recovery.
- **Boundary:** flash verification is not runtime/optical acceptance. The
  candidate remains unaccepted and the first-request production blocker open.

### 2026-09-21 — USB response framing evidence on restored firmware

- **Setup:** stopped the empty Mac manager (current0/in_flight0); retained the
  restored known-good firmware. Built a temporary native Go Inspect probe in
  the Dev Container. Probe sends one read-only request, reads at most2048 bytes
  for at most12 seconds and prints only framing/CRC/public numeric status. No
  raw dump, SSID, password or key output; no provisioning writes or retries.
- **Measurement:** first probe wrote512 bytes and received768. A complete EPCR
  began at offset256: version2/operation1/state1/auth2/code0, CRC and canonical
  ESP32 decode valid, generation1, elapsed98ms. Second independent probe found
  EPCR at offset0, total512, same valid numeric result, elapsed94ms.
- **Inference:** extra bytes before a valid response explain this observed
  host framing/validation failure: the normal client expects its first512 bytes
  to be one EPCR. They do not establish credential corruption. The origin of
  the256-byte prefix was not captured; do not label it ROM output or a stale
  reply without further evidence. This does not prove the historical no-data
  timeout or acceptance of the UART candidate.
- **Safety:** do not blindly scan for a successful mutation ACK or retry a
  credential write. EPCQ currently has no per-request nonce; a stale valid ACK
  must not promote a new enrollment. A future synchronization fix needs bounded
  parsing, explicit operation semantics and stale-response regression tests.

### 2026-09-21 — correlated USB v3 software candidate

- **Decision:** preserve v2 for Pico and existing ESP32 tools; opt into ESP32
  v3 explicitly through `epaperprovision -target esp32 -usb-v3`. Every request
  carries a fresh 128-bit ID echoed by the response. Bounded scanning can skip
  noise/stale frames without accepting an older mutation ACK. No automatic
  retry, legacy fallback, credential migration or Wi-Fi protocol change.
- **Fact:** 18 native tests, the full remote-epaper Go suite and native Go/C
  interoperability (v2 and v3 provisioning/inspection/rotation/erase) passed.
  Scoped lint passed. New codec functions have 100% statement coverage;
  correlated execute has 94.1%, response scanner 100%. These are scoped
  coverage measurements, not the repository-wide quality gate result.
- **Gate:** `QUALITY_BASE_REF=HEAD scripts/quality.sh task` subsequently passed
  in 39 seconds: changed Go coverage 95.8%, total 92.1% against 75.0% baseline;
  lint, file-length guards and preserved Pico USB/Wi-Fi/screen builds passed.
- **Measurement:** ESP-IDF candidate `build-usb-v3/epaper_receiver.bin`, size
  `0xbc810`, SHA256
  `1becdd820456d4a14938139b35855f5e18075ff3f11260843ecbcabcce4f9aba`.
  Its sdkconfig and partition-table binary match build-release byte-for-byte.
  Known-good build-release SHA256 remains
  `084cc044e996ee70154665446c99ad6198bbe79f314e9158937a1cb31e5c85f4`.
- **Boundary:** no board access/flash in this increment. The candidate combines
  correlation and the prior idle-byte fix; neither fixes the unknown origin of
  the observed prefix. First-open behavior and hardware acceptance remain open.
- **Guard:** `receiver/USB-CORRELATION.md`, native provisioning correlation
  tests, Go codec/client/CLI tests and v2/v3 interoperability. Next physical
  step must start with an idle manager and read-only correlated Inspect, not a
  repeated provisioning write.

### 2026-09-21 — USB v3 first physical Inspect passed

- **Precondition:** user authorized flashing; manager reported current0 and
  in_flight0. Stopped the identified idle manager before serial access.
- **Measurement:** app-only write at `0x10000` completed on the detected ESP32
  with 4MiB flash. Device MD5 `9a34003549a8962688028ebcb67815a2` matched the
  candidate. No bootloader, partition table or credential/epoch sectors written.
- **Measurement:** first read-only `epaperprovision-v3 -target esp32 -usb-v3`
  Inspect after reset succeeded: code0/state1/generation1/auth2. No provisioning
  mutation, automatic retry or legacy fallback. Private metadata was filtered
  from tool output. This establishes first-attempt correlated USB acceptance
  for this run, not a statistical reliability guarantee.
- **Recovery:** restarted the same Mac manager/enrollment. HTTPS status ready,
  current0/in_flight0; no scene submitted. Visible pixels and Wi-Fi frame
  delivery on this candidate still require a separate acceptance check.

### 2026-09-21 — USB v3 Wi-Fi acceptance remained unconfirmed

- **Measurement:** manager accepted one existing `wifi-acceptance.html` fixture
  as revision1. TCP connections from the ESP32 were observed, including a
  reconnect, but status remained current1/in_flight0/confirmed0/delivered0.
  No repeated PUT or reduced refresh interval was used.
- **Fact:** manager adds a conservative 180-second floor after first screen
  connection; TCP establishment alone does not prove that handshake succeeded.
  Waiting did not produce delivery evidence, so cooldown is not a proven cause.
- **Measurement:** stopped idle manager and issued read-only EPS2 Health over
  USB. It succeeded: profile1/version1/800x480/passes2/chunk1000,
  minimum_full_ms180000/remaining_ms0; network state0/last_failure1/failures13,
  uptime548 seconds. USB ownership blocks networking, so state0 during this
  probe is expected. Last_failure1 maps to radio-join phase in `network.c`;
  it does not identify the AP/driver/event failure or prove a v3 regression.
- **Decision:** candidate passed first USB Inspect and EPS2 Health, not Wi-Fi
  frame acceptance. Restore retained build-release app only before further
  comparison; preserve enrollment. No claim of visible pixel change.

### 2026-09-21 — refresh priority software implementation and build guard

- **Decision:** user approved request metadata for urgent updates with a
  separate configurable budget, not an unconditional 180-second prohibition.
  Implemented HTTP revision metadata, urgent batching, manager/USB/Wi-Fi cadence
  and negotiated EPS2 BeginRefresh in the ESP32 receiver. Panel commands,
  wiring, BUSY waits, sleep, digest and ownership rules are unchanged.
- **Fact:** this ESP32 panel adapter is still full-only. Explicit partial is
  rejected before staging; no partial waveform was enabled. Full refresh
  defaults are normal180s / urgent30s when the operator opts in, not newly
  established manufacturer limits or guarantees.
- **Measurement:** initial quality gate passed in33s, changed Go coverage95.6%,
  total92.1%; native18/18 and legacy/new Go-to-C full-frame interop passed.
  Scoped race tests and `go vet ./...` passed. This is software evidence only.
- **Review:** independent scoped code review found no actionable correctness
  issue in priority/reconciliation/ownership; reviewer did not run hardware.
- **Build finding:** the first candidate inherited ignored root `sdkconfig`
  with PANIC_PRINT_REBOOT, unlike known-good build-release's SILENT_REBOOT.
  Partition binaries matched; panic configuration did not. That candidate is
  not approved for flashing. Use an isolated build-local sdkconfig from tracked
  defaults and enforce silent panic / no core dump in CMake so stale local
  config fails closed. No change to the retained build-release image.
- **Unknown:** the preceding rollback flasher session's final MD5 result was
  lost across session restoration; no running flasher was observed afterwards.
  Do not claim the currently installed image is verified build-release.
  The USB-v3 candidate's earlier Wi-Fi non-delivery remains unresolved; the
  priority change is not evidence that it fixes that network observation.
- **Guard:** `refresh-priority.md`, Go policy/API/client/transport tests, native
  `screen_test` and Go/C receiver interop. Physical acceptance must use matching
  updated manager/tools and firmware, preserving credential/epoch partitions.
- **Final software measurement:** quality task passed again in28s with changed
  coverage94.8%, total92.1%. Isolated SDK-config build completed: application
  `build-refresh-policy/epaper_receiver.bin`,772400bytes, SHA256
  `d96b45e427116f383d15faea7c0140b55b6caac2601fd099e262fedf49f73dbd`,
  MD5 `708b61f49df59df1f97933dca08c2aff`. Generated SDK header and partition
  binary now match build-release byte-for-byte. Unsafe root-config reconfigure
  was tested and fails with the explicit new CMake guard. macOS arm64 manager,
  EPS2 USB and provisioning tools built separately under `.local-mac/*-refresh`.
- **Boundary:** candidate has not been flashed. Requested confirmation of an
  idle connected board and permission for macOS host-tool execution under the
  user's host/container rules. No new credential write or key rotation.

### 2026-09-21 — priority candidate application flash and USB acceptance failure

- **Fact:** user confirmed connected idle ESP32/7.5 V2 and authorized native
  macOS USB tools. App-only flash at0x10000 completed; device MD5 verified
  `708b61f49df59df1f97933dca08c2aff`, then reset. Credentials and epoch sectors
  were not written. One block timeout retried successfully during flashing.
- **Measurement:** updated `epaperscreen-refresh --status` returned EOF twice
  on `/dev/cu.usbmodem5B140746091`; second error identified operation10/pass0/
  offset0. The port remained enumerated. No frame was submitted.
- **Unknown:** flash integrity does not establish running application health;
  EOF alone does not distinguish firmware failure from host serial lifecycle.
- **Decision:** restore retained build-release application before further
  experiments and compare the same read-only status probe. No wiring changes.

### 2026-09-21 — probe correction and verified rollback

- **Correction:** the two candidate probes above omitted the required `esp32:`
  endpoint prefix and selected Pico serial lifecycle. They are invalid evidence
  of candidate failure. The documented ESP32 prefix must be preserved in copied
  acceptance commands; added it explicitly to `refresh-priority.md`.
- **Measurement:** rollback application MD5 verified
  `63c45c0eba9c2d2ce7547ba4e727c637`; reset completed. The updated host tool's
  corrected `--status esp32:/dev/cu.usbmodem5B140746091` also returned EOF on
  the retained application. No frame was sent, no credentials changed.
- **Unknown:** USB framing/open/reset behavior remains unresolved. This does
  not establish a refresh-policy regression or panel/wiring failure.
- **Next boundary:** request a physical idle USB power removal/reconnection,
  then read-only ESP32 status before any further flash or image submission.
  The known-good application is restored, not newly optically accepted.

### 2026-09-21 — restored application USB status after physical reconnection

- **Fact:** user confirmed physical USB reconnection without buttons.
- **Measurement:** corrected ESP32 status probe succeeded: profile1/version1,
  800x480, two passes, chunk1000, minimum180000ms, remaining128558ms,
  uptime51s. Network state1/last_failure3/failures3 was reported; no network
  diagnosis or visible-frame acceptance follows from that status.
- **Inference:** reconnection restored this read-only USB exchange. It does not
  prove the exact cause of preceding EOF or a refresh-policy regression.
- **Decision:** compare the priority candidate using the correct ESP32 endpoint,
  app-only flashing and unchanged credentials. No frame submitted yet.

### 2026-09-21 — priority candidate reinstalled with corrected probe

- **Measurement:** app-only write completed, MD5 verified
  `708b61f49df59df1f97933dca08c2aff`; flasher reset completed. Correct ESP32
  status probe immediately returned EOF. No frame or credential mutation.
- **Inference:** the post-flasher result matches the retained application's
  pre-power-cycle symptom. Candidate acceptance remains incomplete; a cold
  USB reconnection is required for a like-for-like comparison, not another
  firmware change. Do not classify it as a policy regression on this evidence.
- **Next boundary:** physical USB reconnection, then read-only status. Retain
  the known-good rollback image; no further experiment before this comparison.

### 2026-09-21 — priority candidate responds after USB reconnection

- **Fact:** user confirmed another physical USB reconnection. Candidate remains
  the MD5-verified `708b61f49df59df1f97933dca08c2aff` application.
- **Measurement:** correct ESP32 status probe passed:800x480, two passes,
  chunk1000, remaining163748ms, uptime16s; network state6/last_failure3/failures2.
  This is USB protocol evidence only, not Wi-Fi or optical acceptance.
- **Measurement:** matching native Mac manager started in USB mode with
  `-refresh-policy -full-interval 180s -urgent-interval 30s`. One authenticated
  TLS PUT of `refresh-priority-acceptance.html` with urgent/full headers was
  accepted as revision1. Initial status current1/in_flight0/confirmed0/delivered0
  is pending, not display success. No duplicate PUT or credential writes.
- **Next boundary:** await terminal delivery and user observation of
  `PRIORITY READY / FRAME P1`; startup uncertainty guard remains active.

### 2026-09-21 — priority candidate USB urgent/full controller completion

- **Measurement:** the single urgent/full revision1 reached confirmed1/delivered1,
  in_flight0 and refresh_trusted=true. Full cycle1 started at
  `2026-09-21T12:33:59.652201Z` and completed at
  `2026-09-21T12:34:14.728313Z` (about15.08s). No duplicate PUT was issued.
- **Boundary:** this establishes updated manager/USB/ESP32 protocol completion
  for the candidate and negotiated policy path. Visible `PRIORITY READY / FRAME P1`
  still requires user confirmation. It does not prove urgent30s cadence across
  consecutive frames, Wi-Fi acceptance or partial refresh support.
- **State:** local USB manager remains running for the next acceptance step.
  No credential writes, wiring changes or git commit.

### 2026-09-21 — priority P1 visibly accepted; cadence comparison started

- **Fact:** user confirmed `PRIORITY READY / FRAME P1` visible on the panel.
  This closes the optical boundary for the candidate's first USB frame.
- **Measurement:** a single normal/full P2 PUT was accepted as revision2 at
  `2026-09-21T12:38:04Z`, after P1's180s normal budget had elapsed.
- **Plan:** only after P2 terminal completion, submit one urgent/full P3 and
  compare completion-to-start timing against configured30s/180s budgets.
  These operator budgets are not newly established manufacturer safety limits.

### 2026-09-21 — USB normal-to-urgent cadence measured

- **Measurement:** normal P2 completed at12:38:19.148367Z. Urgent P3 PUT was
  accepted as revision3 at12:38:36Z, before its30s eligibility time. P3 started
  at12:38:49.190982Z and completed at12:39:04.258320Z. All timestamps UTC.
- **Result:** observed completion-to-next-start gap30.043s, below normal180s
  but not below configured urgent30s. Status confirmed3/delivered3/in_flight0,
  refresh_trusted=true. No duplicate submissions, GPIO or credential changes.
- **Boundary:** this verifies negotiated urgent cadence for this USB sequence,
  not long-running panel safety, partial refresh or Wi-Fi delivery. User
  observation of `URGENT READY / FRAME P3` remains pending.

### 2026-09-21 — urgent P3 visually accepted

- **Fact:** user explicitly confirmed `FRAME P3` visible. Together with the
  recorded controller completion and30.043s inter-refresh gap, this closes
  optical acceptance for the tested USB normal-to-urgent sequence.
- **Boundary:** this is not acceptance of Wi-Fi priority delivery, prolonged
  operation, interruption recovery or partial refresh. Existing firmware
  remains full-only. No additional frame, flash or credential write was made
  for this confirmation.

### 2026-09-21 — encrypted Wi-Fi priority candidate controller completion

- **Fact:** stopped the identified idle USB test manager and started the matching
  Wi-Fi manager with existing enrollment, API token and TLS certificate. No
  provisioning, key rotation, flash, USB probe or wiring change in this test.
- **Measurement:** authenticated TLS urgent/full PUT of
  `refresh-priority-wifi.html` was accepted as revision1 at12:47:32Z. TCP peer
  was observed; TCP alone was not treated as authenticated delivery evidence.
- **Measurement:** status reached confirmed1/delivered1/in_flight0 and
  refresh_trusted=true. Cycle started12:50:15.703880Z and completed
  12:50:21.828593Z (UTC). Only one PUT was issued; startup guard was retained.
- **Boundary:** encrypted device-path/controller completion is established for
  this run with the priority candidate. User observation of `FRAME W1` remains
  pending. This does not prove repeated Wi-Fi urgent cadence, interruption
  recovery, long-running reliability or partial refresh. Earlier non-delivery
  reports remain historical facts, not retroactively diagnosed by this success.

### 2026-09-21 — encrypted Wi-Fi W1 visually accepted

- **Fact:** user confirmed `WIFI PRIORITY READY / FRAME W1` visible on the
  7.5-inch panel. This closes optical acceptance for the preceding single
  encrypted Wi-Fi urgent/full delivery using the priority candidate.
- **Accepted scope:** USB P1 and P3 visible, USB normal-to-urgent gap30.043s,
  and Wi-Fi W1 visible with matching controller completion. Existing keys
  were preserved. Candidate application MD5 remains
  `708b61f49df59df1f97933dca08c2aff`.
- **Remaining:** repeated Wi-Fi cadence, interruption/reconciliation,
  prolonged operation and post-flash USB lifecycle diagnosis. Full-only panel
  adapter remains unchanged; no production-completeness or partial claim.

### 2026-09-21 — priority candidate idle manager-restart recovery started

- **Precondition:** Wi-Fi W1 status confirmed1/delivered1/in_flight0. Stopped
  only the identified test manager with SIGTERM, then restarted identical
  arguments/enrollment. No board reset, USB opening, key or firmware writes.
- **Measurement:** new manager ETag namespace started at revision0. One
  urgent/full PUT of `refresh-priority-reconnect.html` was accepted as revision1
  at12:57:06Z. It is a producer resubmission, not automatic scene persistence.
- **Boundary:** tests reconnect after idle server restart, not power loss or
  interruption during staging/refresh. Await completion and visible FRAME W2.

### 2026-09-21 — idle Wi-Fi manager restart controller recovery passed

- **Measurement:** ESP32 reconnected to the replacement manager without board
  reset or USB access. W2 reached confirmed1/delivered1/in_flight0 and
  refresh_trusted=true. Cycle started12:59:35.562551Z and completed
  12:59:42.712571Z (UTC). Exactly one post-restart PUT was issued.
- **Result:** this run establishes authenticated delivery after an idle manager
  restart using unchanged enrollment. The producer supplied the scene again;
  automatic scene persistence and in-flight interruption are not established.
- **Boundary:** user observation of `WIFI RECONNECTED / FRAME W2` remains
  pending. Manager remains running; no additional update queued.

### 2026-09-21 — idle Wi-Fi manager restart visually accepted

- **Fact:** user confirmed `WIFI RECONNECTED / FRAME W2` visible. Combined
  with the preceding controller result, this closes visible acceptance for
  recovery after an idle manager restart with unchanged enrollment and no
  board reset. The scene was explicitly resubmitted by the producer.
- **Remaining boundary:** interruption during an active transfer/refresh,
  lost-ACK reconciliation and prolonged operation are still not physically
  accepted. No additional frame, reset, flash or key write for this confirmation.

### 2026-09-21 — active-interruption preflight; physical test not started

- **Measurement:** existing Dev Container race run passed selected screenhub,
  screenclient and manager regressions: lost Commit/Data replies, lost Commit
  ACK without new SPI, interrupted-update reconciliation, pending upload abort,
  and original-cycle confirmation without resend (`-count=1`).
- **Boundary:** these are software tests, not physical interruption acceptance.
  No frame was submitted and no live connection was cut in this preflight.
- **Safety:** receiver README requires physical power removal after unknown
  refresh or failed staging cleanup; EN/software reset is insufficient. A
  physical test needs an available operator for that fault boundary.
- **Method:** do not use timed process termination as evidence of a specific
  protocol cut. Retain the client transaction across a precisely identified
  transport cut; observe Query/reconciliation rather than submit blindly.
  No existing production CLI fault-injection control was identified during
  this preflight. A bounded test harness is needed before that experiment.

### 2026-09-21 — first terminal ACK-loss hardware probe: reconnect failed

- **Preparation:** test-only native harness added; default encrypted simulator
  and race checks passed. Quality task passed26s, changed coverage94.8%, total
 92.1%. Firmware and production manager code were not changed by this harness.
- **Precondition:** manager reported no in-flight work and a completed maintenance
  cycle; stopped only that manager. Operator confirmed available for power removal.
- **Measurement:** physical probe authenticated, waited180s, sent one full
  quadrant frame, validated terminal Commit CodeOK/Complete, deliberately hid
  that ACK from the retained client and closed the socket. On the next
  authenticated connection, Hello failed with TCP reset by peer. Test exited
  failure after201s; Query confirmation was not obtained. No second Send.
- **Boundary:** the test observer saw completion, so this is not an unknown
  physical-refresh outcome. Reconciliation acceptance failed. The test process
  ended and its client identity is no longer retained; never claim a later
  fresh process reconciled this transaction. Root cause of the TCP reset is
  unknown. No automatic repeat of this experiment or firmware rollback yet.
- **Next:** align the test's reconnect handling with existing Hub.WaitReady:
  bounded transient reconnects retaining the same client, never another Send;
  prove this behavior in the simulator before a separate new experiment.

### 2026-09-21 — ACK probe reconnect handling corrected

- **Change:** test-only harness now allows at most three transient Connect
  failures, preserving the original client/claim/pending identity and closing
  each failed stream. Uses the existing Hub transient classifier; no Send retry.
  Default simulator injects a failed reconnect and asserts one sink Commit,
  one original Commit request, zero reconnect Commits and one successful Query.
- **Measurement:** focused race tests passed; quality task passed25s, changed
  coverage94.8%, total92.1%. Mac test binary rebuilt in the Dev Container.
- **Experiment:** a separate new physical probe was explicitly launched after
  prior terminal completion was known. It waits at least180s before its one
  frame; it does not claim to recover the exited first probe's transaction.
- **Boundary:** receiver firmware and production manager remain unchanged.

### 2026-09-21 — terminal ACK-loss hardware reconciliation passed

- **Measurement:** second, separate live probe passed in242.99s. The observer
  validated terminal Commit CodeOK/Complete, hid it from the retained client,
  closed TCP, and reauthenticated. That same client then reconciled its pending
  identity: confirmed=true; Commit requests1, reconnect Commit requests0,
  Query requests1. No Send replay. This run needed no transient retry.
- **Result:** real ESP32 encrypted-path reconciliation after host-side terminal
  ACK loss is accepted at the protocol boundary. No independent physical
  refresh counter exists; do not claim request counts are such telemetry.
- **Boundary:** visual quadrant-pattern confirmation is pending. This test is
  not staging interruption, loss before refresh completion, power failure or
  manager process-crash persistence. First probe's TCP reset remains an
  unresolved historical observation; second success is not its root cause.
- **Recovery:** restarted the normal Wi-Fi manager with unchanged keys and
  arguments, no scene submission. No firmware change, power removal or git
  commit was needed for either probe. Operator remained available.

### 2026-09-21 — terminal ACK-loss pattern visually accepted

- **Fact:** user confirmed the four large black/white rectangles visible after
  the successful terminal ACK-loss probe. This closes the visual boundary for
  that run, alongside confirmed reconciliation and zero reconnect Commits.
- **Boundary:** visible pixels do not independently count refresh operations.
  Staging interruption, power failure and process-crash persistence remain
  separate unaccepted scenarios. No additional frame or device action was made.

### 2026-09-21 — staging interruption probe prepared and started

- **Scope:** extended only the test harness to cut after the first validated
  Data ACK in Receiving state, before any Commit. The pending client is kept
  across bounded reconnects; Query must be unconfirmed with no hardware error.
  No firmware, production manager, key or wiring change.
- **Software:** initial RED caught missing harness support; a test-fixture
  padding mistake was corrected using Frame.Clear for canonical black pixels.
  Simulated encrypted staging interruption passed under race detection with
  zero sink Commits, zero Commit requests and Query code0/state5. Existing
  terminal ACK scenario still passes. Quality task passed26s, changed
  coverage94.8%, total92.1%; native Mac test binary rebuilt in the container.
- **Physical precondition:** normal manager current0/in_flight0; stopped that
  process only. Explicit staging probe launched with existing enrollment and
  operator available. It retains the at-least180s initial guard and sends no
  automatic replacement frame. Prior quadrant pattern should remain visible.

### 2026-09-21 — physical staging interruption passed at protocol boundary

- **Measurement:** live staging test passed in185.42s. First Data ACK validated,
  TCP closed before Commit; same client reauthenticated and queried once.
  Result unconfirmed, code0/state5(Closed), no pending client transaction,
  original Commit requests0 and reconnect Commit requests0. No hardware error.
- **Result:** receiver accepted cleanup of this partially staged transaction
  without claiming a completed image. No retry Send or power removal occurred.
- **Boundary:** old quadrant-pattern preservation still awaits user observation.
  No claim of measured rail power removal, independent refresh counting,
  interruption inside physical refresh or prolonged fault endurance.
- **Recovery:** ordinary Wi-Fi manager restarted with the same keys and no
  scene submission. Firmware and wiring unchanged.

### 2026-09-21 — interrupted staging visually accepted

- **Fact:** user confirmed the four black/white rectangles remained unchanged
  after the staging interruption test. Together with zero Commit requests and
  unconfirmed code0/state5, this closes acceptance for this pre-Commit cut.
- **Boundary:** no independent panel refresh counter or power measurement;
  mid-refresh power loss and prolonged operation remain separate checks.
  No additional device action was performed for this confirmation.

### 2026-09-21 — new Wi-Fi frame after interrupted staging completed

- **Fact:** with the restored ordinary manager and unchanged ESP32 firmware,
  power and enrollment, submitted one normal/full `refresh-after-staging.html`
  fixture as revision1. This is a new complete frame, not a replay of the
  interrupted transaction. No board reset, USB access or key write.
- **Measurement:** accepted at13:52:47Z; cycle started13:52:47.740146Z and
  completed13:52:53.573722Z(UTC), about5.834s. Status confirmed1/delivered1,
  in_flight0, refresh_trusted=true. Only one PUT issued.
- **Boundary:** controller completion establishes subsequent delivery after
  staging cleanup. User observation of `RECOVERY READY / FRAME W3` is pending;
  this does not establish power-loss or mid-refresh interruption recovery.

### 2026-09-21 — post-staging recovery frame visually accepted

- **Fact:** user confirmed `RECOVERY READY / FRAME W3` visible. Together with
  controller completion, this verifies a subsequent complete Wi-Fi update
  after the tested pre-Commit interruption, without restarting ESP32.
- **Accepted sequence:** interrupted staging had zero Commit requests and
  preserved the old image; a separately submitted complete frame then displayed.
- **Boundary:** power-loss, mid-refresh interruption and prolonged operation
  remain separate acceptance gates. No device action for this confirmation.

### 2026-09-21 — idle power-cycle acceptance prepared

- **Precondition:** current1/confirmed1/delivered1/in_flight0,
  refresh_trusted=true. Latest maintenance cycle2 completed at
  `2026-09-21T14:02:59.720085Z`; no active panel update was reported.
- **Action:** SIGSTOP sent only to identified test manager PID17770 to prevent
  scheduled maintenance during the operator's power-disconnection window.
  Process, canonical scene and client state remain in memory; no restart,
  provisioning, flash or new frame submission.
- **Next:** operator removes/reconnects board USB power while idle, then resume
  this exact manager with SIGCONT. Do not leave it paused after the test.
  This tests idle power loss, not interruption during physical refresh.

### 2026-09-21 — idle power cycle: retained manager resumed

- **Fact:** operator confirmed USB power reconnection. Verified PID17770 was
  the same stopped test manager, then sent SIGCONT. No new process, scene PUT,
  serial access, provisioning or firmware write.
- **Measurement:** initial status retained old confirmed1; subsequent status
  changed to confirmed0/refresh_trusted=false while current1/delivered1 remained.
  A new ESP32 TCP connection was observed. Old cycle2 timestamps are historical,
  not evidence of post-power-cycle completion.
- **Inference:** retained manager invalidated its prior confirmed baseline and
  queued recovery. Await a new confirmed cycle; authentication/recovery success
  must not be inferred from TCP alone. No mid-refresh power loss was tested.

### 2026-09-21 — retained-manager idle power recovery completed

- **Measurement:** with no new PUT, same manager PID17770 recovered current1
  from confirmed0/untrusted to confirmed1/trusted. New cycle3 started
  `14:14:59.712876Z`, completed `14:15:05.446863Z` (UTC). This proves recovery
  of the in-memory scene after operator-confirmed idle ESP32 power cycle,
  not scene persistence across a manager process restart.
- **Separate optical check:** only after that completion, one urgent/full W4
  PUT was accepted as revision2 at14:16:36Z. Cycle4 started14:16:36.580743Z,
  completed14:16:42.303460Z; status confirmed2/delivered2/in_flight0/trusted.
  User observation of `POWER RECOVERY OK / FRAME W4` remains pending.
- **Boundary:** no credentials, firmware or wiring changed. Board power loss
  was while idle, not during physical refresh or flash writes. Manager is
  resumed/running, not left stopped. Prolonged reliability remains unaccepted.

### 2026-09-21 — idle power recovery W4 visually accepted

- **Fact:** user confirmed `POWER RECOVERY OK / FRAME W4` visible. The retained
  manager first recovered its previous scene without a PUT, then a separately
  submitted new frame completed and was visually accepted with unchanged keys.
- **Accepted scope:** idle ESP32 power cycle with the same resumed manager;
  not power interruption during refresh, credential writes or manager restart.
- **Remaining:** prolonged reliability and other fault boundaries are not
  established by this single run. No additional device action for confirmation.

Append or update an entry after each material experiment:

```text
Date:
Scope and exact hardware/firmware:
Changed variable:
Symptom or objective:
Facts:
Measurements:
Inference / root cause and confidence:
Decision:
Verification evidence:
Remaining unknowns:
Regression guard:
Sources and artifact:
```
