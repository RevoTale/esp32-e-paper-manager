# Spec: image-conversion

Status: approved automatically by user policy on 2026-08-30.

## Objective

Convert trusted local PNG, JPEG, or GIF input into the exact 800×480,
MSB-left, 0=white/1=black panel frame on the host.

## Contract

- `Convert(dst, src, Options)` requires exactly 48,000 destination bytes and
  validates fit mode, rotation (0/90/180/270), and threshold before mutation.
- `contain` preserves the whole image on white letterbox; `cover` center-crops.
- Rotation is applied before geometry. Transparent pixels composite over white.
- Scaling uses official `golang.org/x/image/draw.ApproxBiLinear`; conversion is
  deterministic thresholding or bounded two-row Floyd–Steinberg diffusion.
- File decoding checks encoded size (32 MiB) and decoded dimensions/pixel count
  (50 megapixels) before allocating the decoded image. Format is detected from
  content, not extension. No URL fetching is supported.

## Verification

Pixel-exact tests cover polarity, MSB order, contain geometry, rotation,
dithering, invalid options without destination mutation, content-based decode,
and file/pixel limits. Run `go test ./frameimage`.

Source: https://pkg.go.dev/golang.org/x/image/draw
