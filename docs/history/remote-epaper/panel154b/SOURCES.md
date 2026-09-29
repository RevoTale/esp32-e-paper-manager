# Waveshare 1.54-inch (B) V2

This is a separate 200x200 black/white/red driver. It does not replace any
existing Pico or Linux panel implementation.

- [Official controller implementation](https://github.com/waveshareteam/e-Paper/blob/a794fbc39656b0f93938d1ffb3fdc77eaed9e9fc/RaspberryPi_JetsonNano/c/lib/e-Paper/EPD_1in54b_V2.c).
- [Specification](https://files.waveshare.com/upload/9/9e/1.54inch-e-paper-b-v2-specification.pdf).
- [Manual](https://www.waveshare.com/wiki/1.54inch_e-Paper_Module_%28B%29_Manual).

Black input bits are inverted for command 0x24; red input bits are sent directly
to 0x26. This corresponds to the vendor's white clear (0xff, 0x00). Input has
no conflicting colored pixels. Data-entry/window order follows the vendor.
30-second BUSY timeout is a project safety budget, not a display specification.
No custom LUT, partial update or periodic automatic refresh is implemented.
After a failure the caller must report it; no blind retry or assumption of sleep.
Successful completion means controller completion only, not visible acceptance.

Command-sequence adaptation attribution: Copyright Waveshare team.
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
