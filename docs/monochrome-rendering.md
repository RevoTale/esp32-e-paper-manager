# Monochrome rendering

The manager's engine composes HTML, colors, images and alpha into RGBA first,
then converts the final viewport to packed monochrome pixels.

Default conversion uses a fixed luminance threshold of 128: dark pixels become
black, lighter pixels become white. This preserves solid dark text and fills
without adding a periodic halftone pattern. Text antialiasing still determines
edge coverage in the RGBA raster; no resizing or panel-coordinate changes occur.

Callers that deliberately want gray-tone approximation can use
`Result.FrameWithMode(ctx, engine.OrderedDither)`. This retains the earlier
display-anchored Bayer 4x4 pattern. It applies to the complete composited frame,
not just image tags. Threshold and ordered conversion are pixel-local, so
changing one area cannot propagate diffusion error into unrelated rows.

## Grain investigation — 2026-10-08

Symptom: regularly spaced missing squares in text and drawing.
Reproduction: a solid CSS #111 surface is RGBA (17,17,17,255), but the old
Bayer conversion produces a white pixel at every (x mod4=0,y mod4=0).
It loses 1/16 of fully covered dark ink. The artifact exists in the host mono
preview before USB, Wi-Fi or controller writes; it is not evidence of a panel
electrical fault. Other shades can lose more pixels.

Fix: make threshold conversion the dashboard default, keep halftoning explicit.
Tradeoff: threshold does not preserve gray-tone densities in photographs;
image-oriented consumers can select OrderedDither.
Regression tests cover real dark text, #111/#333/#555 solid fills, default gray
conversion, invalid modes and the existing anchored immutable dithering oracle.
Historical bilinear/clipping test geometry is unchanged; only its quantized
pixel oracle follows the new default.

This is a Go rendering change. Updating the manager/client renderer applies it;
the ESP32 firmware transports the resulting bits unchanged. Preview success
does not confirm the user's physical screen. No hardware frame was sent here.

Verification: the regression failed on the earlier conversion (#111 hole at
4,4), then passed after the change. The clean-candidate make quality completed
with exit0: three stable root coverage profiles at94.8%, changed coverage100%,
lint, race/vet, audit, native/interoperability, TinyGo, ESP32 and Linux builds.
Local evidence is ignored build/quality-grain-fix.log and the grain-debug
RGBA/mono/fixed PNGs. The original checkout's ignored soak harness is outside
this candidate; no quality rule or test exclusion was introduced.
