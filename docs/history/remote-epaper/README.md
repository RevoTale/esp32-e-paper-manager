# Pico 2 W + Waveshare 7.5-inch V2 remote e-paper

## Поточний напрямок: native Go engine, 2026-09-07

HTML та inline CSS тепер обробляє **Go manager**, не Pico. Він рахує layout
під viewport, малює кадр і передає EPS2 Raw/PackBits. Одна generic TinyGo
прошивка `cmd/screen-device` приймає готові пікселі через USB або захищений
WPA3-зв’язок і передає їх у RAM контролера без повного framebuffer на Pico.
USB має пріоритет. Rust/Blitz більше не є активними залежностями.

Починай з [інструкції для нового candidate](docs/native-candidate.md),
[профілю HTML/CSS](SPEC-engine.md) та [плану міграції](tasks/pure-go-engine.md).
Готовий пакет і точні результати: [приймання candidate](docs/native-candidate-acceptance.md).
Нову прошивку ще потрібно фізично прийняти; успіх старих тестів не є доказом
її роботи на екрані. Старі артефакти збережено для повернення до перевіреного стану.

## Історичний buffered/HTML шлях

Опис і команди нижче стосуються попередньої реалізації `cmd/device`, а не
нового EPS2 candidate. Вони збережені разом із фактичними результатами bring-up.

Цей експеримент керує придбаним чорно-білим Waveshare 7.5″ V2 (800×480,
product 13504) через Waveshare e-Paper Driver HAT Rev2.3. TinyGo firmware
приймає bounded HTML через USB CDC або Pico-initiated encrypted manager link,
локально виконує HTML/CSS/layout/1-bit raster і безпечно оновлює панель.

Зафіксований тоді стан: software task gate проходить із 90.3% changed coverage;
USB-only і WPA3 TinyGo build компілюються. Фізичні 20 USB-передач, WPA3/reconnect,
енергетичні заміри й target IPv6 ще не прийняті. Pico не відкриває listener:
він сам підключається до trusted manager. Без USB-provisioned WPA3 config радіо
не вмикається.

2026-09-02 повний USB шлях фізично підтверджено: актуальна TinyGo firmware
отримала HTML, локально відрендерила його й показала новий кадр `USB UPDATE`.
Це перша прийнята передача, а не заміна ще не виконаної серії з 20 оновлень.
Історія складного bring-up та його регресійні правила збережені в
[`docs/epaper-debugging-history.md`](../../docs/epaper-debugging-history.md).

## Безпека перед підключенням

- Від’єднуй USB і батареї перед зміною GPIO або FPC.
- Не вставляй і не виймай FPC під живленням.
- На маленькому синьому адаптері відкриті срібні контакти FPC дивляться
  **догори, від плати**.
- HAT: `Display Config = B / 0.47R`, `Interface Config = 0 / 4-wire SPI`.
- Не подавай батарейну напругу на `3V3 OUT`.

## Підключення HAT до Pico 2 W

USB-роз’єм Pico зверху, написи на нижньому боці плати читаються нормально:

| HAT | Pico GPIO | Фізичний pin |
|---|---:|---:|
| `PWR` | `GP15` | 20 |
| `CS` | `GP17` | 22 |
| `CLK` | `GP18` | 24 |
| `DIN` | `GP19` | 25 |
| `DC` | `GP20` | 26 |
| `RST` | `GP21` | 27 |
| `BUSY` | `GP22` | 29 |
| `VCC` | `3V3 OUT` | 36 |
| `GND` | `GND` | 38 |

`GP16` (pin 21) лишається SPI0 SDI через вимогу TinyGo/RP2350, хоча дисплей
дані назад не передає. Якщо `PWR` досі на GP16, спочатку повністю знеструм Pico
і перестав лише цей провід на GP15.

Для 3×AA Ni-MH батарей: червоний провід має йти на `VSYS` (pin 39), чорний —
на `GND`. Оголені проводи без пайки або надійного клемника не використовуй:
коротке замикання тут реальніше за програмну помилку. Під час USB батареї поки
не підключай.

## Build у Dev Container

```sh
cd /workspaces/pico-sandbox/experiments/03-remote-epaper
go test ./...
go vet ./...
./scripts/check-protocol-resources.sh
tinygo build -target=pico2-w -scheduler=tasks -size=short \
  -o remote-epaper-usb.uf2 ./cmd/device
tinygo build -target=pico2-w -scheduler=tasks -tags=wifi -size=short \
  -o remote-epaper-wifi.uf2 ./cmd/device
go build -trimpath -o epaperhtml ./cmd/epaperhtml
go build -trimpath -o epaperprovision ./cmd/epaperprovision
go build -trimpath -o epaper-manager ./cmd/epaper-manager
```

Контейнер зафіксований на TinyGo 0.41.1 і Debian-based офіційному image.

## Quality gates

Pinned development tools live in the separate `tools/go.mod`; firmware
dependencies are unchanged. Run inside the Dev Container:

```sh
./scripts/quality.sh fast       # lint changed whole files, file limits, tests
./scripts/quality.sh task       # + coverage ratchets and Pico 2 W build
./scripts/quality.sh full       # + vet, race, module and resource checks
./scripts/quality.sh build      # isolated TinyGo firmware build
./scripts/quality.sh resources  # TinyGo size, stack and allocation report
```

Set `QUALITY_BASE_REF` when the comparison ref is not `origin/main`. Existing
oversized and lint debt is reported, but a modified legacy file must satisfy the
current limits instead of inheriting that debt.

Готові статичні клієнти для Raspberry Pi/Linux:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
  -o dist/epaperctl-linux-arm64 ./cmd/epaperctl
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath \
  -o dist/epaperctl-linux-armv7 ./cmd/epaperctl
```

Для 64-bit Raspberry Pi OS використовуй `epaperctl-linux-arm64`; для 32-bit —
`epaperctl-linux-armv7`. Після копіювання виконай `chmod +x FILE`.

## Захищена Wi-Fi прошивка

- Тільки WPA3-SAE, без WPA2/open downgrade.
- Generic UF2 не містить секретів; SSID, passphrase, manager endpoint,
  timezone, device ID/key задаються лише через USB `epaperprovision`.
- Pico не слухає вхідний порт. Він опитує manager кожні 30 секунд із backoff
  2 секунди…5 хвилин.
- Device link: HMAC-SHA-256 key separation + AES-256-GCM + strict counters.
- Публічний bearer token існує тільки між `epaperhtml` і TLS 1.3 manager та
  ніколи не надсилається Pico.
- USB має пріоритет; активний фізичний refresh не переривається.

Дивись [provisioning](docs/provisioning.md),
[security/network](docs/security-network.md) і [operations](docs/operations-v2.md).
Публічно відкривається лише HTTPS manager/VPN; порт на Pico не форвардиться.
Manager має dual-stack host networking, але target Pico IPv6 поки fail-closed
через неповну підтримку адресної конфігурації у доступному embedded stack.

## Flash через macOS

1. Повністю знеструм Pico.
2. Затисни `BOOTSEL`, під’єднай USB, відпусти кнопку.
3. Переконайся, що macOS змонтувала `/Volumes/RP2350`.
4. На macOS скопіюй потрібний generic UF2:

```sh
cp dist/remote-epaper-usb.uf2 /Volumes/RP2350/
# або WPA3 firmware для подальшого USB provisioning:
cp dist/remote-epaper-wifi.uf2 /Volumes/RP2350/
```

Після копіювання volume зникає і Pico запускає firmware. `picotool -f` не є
основним шляхом: TinyGo 0.41.1 CDC build не надав сумісний USB reset interface.

Щоб копіювання UF2, очікування нового CDC і post-flash сторінка виконались одним
кроком, спочатку збери macOS ARM64 client у Dev Container, а потім на macOS
запусти script із поточним CDC path:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags=directport -trimpath \
  -o dist/epaperhtml-darwin-arm64 ./cmd/epaperhtml
./scripts/flash-usb-macos.sh /dev/cu.usbmodem1101
```

Script не робить refresh на кожен reboot. Він надсилає малу сторінку
`Firmware installed` лише після нового UF2, появи CDC і успішного USB diagnostic
handshake. Очікування CDC не має прихованого deadline; `Ctrl-C` безпечно
перериває очікування.

## Поточний HTML client

Application-mode USB CDC доступний контейнеру без dedicated BOOTSEL attach.
Надіслати bounded HTML через USB:

```sh
./dist/epaperhtml dashboard.html
```

Перед HTML client виконує малий `diagnose` handshake. `usb=ready` підтверджує,
що firmware завантажилась, USB двосторонній, runtime панелі створений і сховище
provisioning читається. Помилка містить точний transport stage: `handshake` або
`update`; обидва етапи bounded до 20 секунд і закривають заблокований port.

Надіслати той самий HTML через HTTPS manager:

```sh
./dist/epaperhtml -manager https://manager.example \
  -token-file ./manager-api-token -device-id DEVICE_ID dashboard.html
```

## Legacy v1 frame client

За документацією OrbStack, application-mode serial/UART автоматично
форвардиться в Linux; dedicated `Attach` потрібен для BOOTSEL/libusb flashing.
Перевір порт:

```sh
./epaperctl -list
```

У вже запущеному privileged Dev Container kernel іноді бачить CDC ACM, але
device node ще відсутній. Перевір:

```sh
cat /sys/class/tty/ttyACM0/dev
```

Якщо команда повернула, наприклад, `166:0`, а `epaperctl -list` порожній,
створи node з цими точними числами:

```sh
mknod /dev/ttyACM0 c 166 0
chmod 660 /dev/ttyACM0
./epaperctl -list
```

Для поточного Pico цей шлях перевірено: клієнт бачить `/dev/ttyACM0` з
`VID=2E8A`, `PID=000A`. Після rebuild контейнера ручний node зникає.

Надіслати зображення; формат визначається з вмісту, а не розширення:

```sh
./epaperctl photo.png
./epaperctl -fit cover -rotate 90 photo.jpg
./epaperctl -dither=false -threshold 150 image.gif
```

Надіслати вже упакований 48,000-byte frame:

```sh
./epaperctl -raw frame.bin
```

Показати файл або output Linux/Raspberry Pi terminal:

```sh
./epaperctl -text status.txt
ip addr | ./epaperctl -text -
```

Текстовий режим має 114×36 символів, input ≤1 MiB. ANSI/OSC escape sequences
відкидаються і не виконуються.

Клієнт пише `queued`, коли frame повністю перевірений Pico. Перший refresh може
початися одразу; наступні старти firmware обмежує інтервалом 180 секунд для
зменшення зносу й ghosting.

## Physical acceptance

Спочатку використовуй окрему діагностичну прошивку, яка чекає USB CDC reader і
показує точний етап, команду, byte offset та BUSY evidence:

```sh
cd /workspaces/pico-sandbox/experiments/07-tinygo-waveshare-7in5-v2-diagnostic
tinygo build -target=pico2-w -scheduler=tasks -o diagnostic.uf2 .
```

Після її видимого checkerboard і `code=OK` послідовно проший:

```sh
picotool load -f -v panel-a.uf2
# візуально підтвердити pattern A; зачекати 180 s
picotool load -f -v panel-b.uf2
# підтвердити pattern B; зачекати 180 s
picotool load -f -v panel-checker.uf2
```

Після цього проший `dist/remote-epaper-usb.uf2` і виконай 20 послідовних
HTML-передач.
Без цієї перевірки драйвер не вважається hardware-accepted.

## Recovery

- Якщо клієнт пише `status=refreshed`, але на екрані старий кадр, це ще не
  успіх: e-paper зберігає попереднє зображення, а контролер не читає пікселі
  назад. Потрібен новий унікальний acceptance-кадр.
- Якщо checker не змінює екран і `BUSY` не показав жодного active-low стану,
  спочатку перевір живлення та контакти. Не переписуй драйвер навмання.
- Для перевірки `PWR` використовуй probe-прошивку з відомими HIGH/LOW
  інтервалами. HIGH має бути приблизно `3.2 V` і на `GP15`, і на контакті
  `PWR` HAT. Якщо він є біля Pico, але зникає на HAT — проблема в jumper або
  конекторі.
- Прозвонка без навантаження може виглядати нормально, бо щуп притискає слабкий
  female Dupont і тимчасово відновлює контакт. Перевіряй напругу на обох
  кінцях, не рухаючи з'єднання.
- Перед переставлянням або заміною jumper від'єднай USB і батареї.
- `RESET` перезапускає firmware, але не повторює стару невдалу передачу.
  Відкрий CDC знову і повтори `epaperctl`.
- Після обриву USB incomplete transfer скидається; committed queued frame
  лишається доступним для refresh.
- E-paper зберігає останнє зображення без живлення — це не означає, що Pico
  зараз працює.
- Два повторні спалахи LED у старому `02-epaper-smoke-test` означають загальний
  failure під час `Init` (часта причина — BUSY timeout), а не точний діагноз.
  `03-remote-epaper` виводить phase помилки в serial лише у pattern firmware.
  Постійне світло саме по собі не доводить успішний refresh. Перевір FPC,
  `BUSY`, `PWR`, switches і повторно надішли frame після знеструмлення.

## Джерела й внутрішні контракти

- [Waveshare Pico-ePaper-7.5](https://www.waveshare.com/wiki/Pico-ePaper-7.5)
- [Waveshare official Pico source](https://github.com/waveshareteam/Pico_ePaper_Code)
- [Waveshare e-Paper Driver HAT](https://www.waveshare.com/wiki/E-Paper_Driver_HAT)
- [TinyGo Pico 2 W](https://tinygo.org/docs/reference/microcontrollers/boards/pico2-w/)
- [TinyGo Pico W Wi-Fi / cyw43439](https://tinygo.org/docs/reference/microcontrollers/featured/pico-w/)
- [OrbStack USB devices](https://docs.orbstack.dev/features/usb)
- [Raspberry Pi picotool](https://github.com/raspberrypi/picotool)
- [`RESEARCH.md`](RESEARCH.md) — перевірені джерела та відомі конфлікти
- [`VERIFICATION.md`](VERIFICATION.md) — актуальні build/resource результати
- `SPEC-*.md` — затверджені модульні контракти
