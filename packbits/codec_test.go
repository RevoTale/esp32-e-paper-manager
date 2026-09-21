package packbits

import (
	"bytes"
	"testing"
)

// Hand-authored PackBits control bytes, independent of our encoder.
func TestDecodeControlBoundaries(t *testing.T) {
	for _, tc := range []struct{ encoded, plain []byte }{
		{[]byte{0, 42}, []byte{42}},
		{[]byte{255, 42}, []byte{42, 42}},
		{[]byte{129, 42}, bytes.Repeat([]byte{42}, 128)},
		{append([]byte{127}, bytes.Repeat([]byte{42}, 128)...), bytes.Repeat([]byte{42}, 128)},
		{[]byte{1, 10, 20, 254, 30, 0, 40}, []byte{10, 20, 30, 30, 30, 40}},
	} {
		dst := make([]byte, len(tc.plain))
		if err := Decode(dst, tc.encoded); err != nil || !bytes.Equal(dst, tc.plain) {
			t.Fatalf("%x: %x, %v", tc.encoded, dst, err)
		}
	}
}

func TestDecodeRejectsMalformedOrMismatchedLength(t *testing.T) {
	for _, tc := range []struct {
		src []byte
		n   int
	}{
		{nil, 0}, {nil, 1}, {[]byte{0}, 1}, {[]byte{1, 4}, 2},
		{[]byte{255}, 2}, {[]byte{128}, 1}, {[]byte{0, 1, 128}, 1},
		{[]byte{0, 1}, 2}, {[]byte{255, 1}, 1}, {[]byte{0, 1}, 0},
		{[]byte{129, 1, 0, 2}, 128},
	} {
		if err := Decode(make([]byte, tc.n), tc.src); err == nil {
			t.Fatalf("accepted %x for %d bytes", tc.src, tc.n)
		}
	}
}

func TestEncodeRoundtripAndCapacity(t *testing.T) {
	for _, input := range [][]byte{
		{1}, {1, 2, 3}, {1, 1}, {1, 1, 1},
		{1, 2, 2, 3}, {1, 2, 2, 2, 3},
		bytes.Repeat([]byte{0}, 1024), bytes.Repeat([]byte{0, 1, 2}, 342),
	} {
		buf := make([]byte, len(input)*2)
		n, err := Encode(buf, input)
		if err != nil || n == 0 {
			t.Fatal(n, err)
		}
		out := make([]byte, len(input))
		if err := Decode(out, buf[:n]); err != nil || !bytes.Equal(out, input) {
			t.Fatalf("roundtrip: %v", err)
		}
		for cap := 0; cap < n; cap++ {
			if _, err := Encode(make([]byte, cap), input); err == nil {
				t.Fatalf("accepted capacity %d, required %d", cap, n)
			}
		}
	}
	if _, err := Encode(make([]byte, 8), nil); err == nil {
		t.Fatal("empty block accepted")
	}
}

func TestCodecAllocatesNoStorage(t *testing.T) {
	var src, encoded, decoded [1024]byte
	if got := testing.AllocsPerRun(100, func() {
		n, err := Encode(encoded[:], src[:])
		if err != nil || Decode(decoded[:], encoded[:n]) != nil {
			t.Fatal("codec failed")
		}
	}); got != 0 {
		t.Fatal("allocations", got)
	}
}

func FuzzCodec(f *testing.F) {
	f.Add([]byte{1, 2, 2, 2}, uint16(4))
	f.Add([]byte{129, 0}, uint16(128))
	f.Fuzz(func(t *testing.T, src []byte, length uint16) {
		if len(src) == 0 || len(src) > 1024 {
			return
		}
		_ = Decode(make([]byte, int(length)%1025), src)
		encoded, decoded := make([]byte, len(src)*2), make([]byte, len(src))
		n, err := Encode(encoded, src)
		if err != nil || Decode(decoded, encoded[:n]) != nil || !bytes.Equal(src, decoded) {
			t.Fatal("roundtrip")
		}
	})
}
