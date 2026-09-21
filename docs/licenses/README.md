# Pinned toolchain notices

These are unmodified upstream license texts, not an independent legal review.
Run `sha256sum -c docs/licenses/SHA256SUMS` from the experiment directory.

- `TinyGo-0.41.1-LICENSE`: `LICENSE` from the verified
  `github.com/tinygo-org/tinygo@v0.41.1` Go module.
- `compiler-rt-builtins-LICENSE`: `lib/compiler-rt-builtins/LICENSE.TXT`
  from the qualified TinyGo 0.41.1 installation.
- `picolibc-b92edfda-COPYING.picolibc` and `picolibc-b92edfda-COPYING.NEWLIB`:
  [COPYING.picolibc](https://raw.githubusercontent.com/picolibc/picolibc/b92edfda8ac6853772d87cadaeeeaa21b78609b6/COPYING.picolibc)
  and [COPYING.NEWLIB](https://raw.githubusercontent.com/picolibc/picolibc/b92edfda8ac6853772d87cadaeeeaa21b78609b6/COPYING.NEWLIB)
  from TinyGo's pinned picolibc revision
  `b92edfda8ac6853772d87cadaeeeaa21b78609b6`.

The qualification inventory is `build/native-firmware-notices-input.txt`;
[firmware resources](../native-firmware-resources.md) records its scope.
The release collector checks these hashes before copying the texts. It also
collects the separately licensed CYW radio firmware notice from its selected
module. These records do not establish hardware acceptance.
