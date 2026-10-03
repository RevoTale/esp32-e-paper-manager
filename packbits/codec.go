// Package packbits provides a bounded, allocation-free byte-run codec.
// Control semantics follow TIFF 6.0 section 9; this is an EPS2 chunk codec,
// not TIFF row/file decoding. The EPS2 subset forbids the unused 0x80 no-op.
// https://www.itu.int/itudoc/itu-t/com16/tiff-fx/docs/tiff6.pdf
package packbits

import "errors"

var ErrBlock = errors.New("packbits: invalid block or insufficient storage")

// Decode must fill dst exactly and consume all src. Both slices are caller-
// bounded and must not overlap. On failure dst is unspecified scratch: never
// publish it or forward it to a sink unless this function succeeds.
func Decode(dst, src []byte) error {
	if len(dst) == 0 {
		return ErrBlock
	}
	for len(src) > 0 {
		written, read, err := decodeRun(dst, src)
		if err != nil {
			return err
		}
		dst, src = dst[written:], src[read:]
	}
	if len(dst) != 0 {
		return ErrBlock
	}
	return nil
}

func decodeRun(dst, src []byte) (int, int, error) {
	control := src[0]
	if control == 128 {
		return 0, 0, ErrBlock
	}
	if control < 128 {
		n := int(control) + 1
		if n > len(src)-1 || n > len(dst) {
			return 0, 0, ErrBlock
		}
		copy(dst, src[1:1+n])
		return n, n + 1, nil
	}
	n := 257 - int(control)
	if len(src) < 2 || n > len(dst) {
		return 0, 0, ErrBlock
	}
	for i := range n {
		dst[i] = src[1]
	}
	return n, 2, nil
}

// Encode writes a deterministic greedy stream, returning ErrBlock if dst is
// insufficient. Callers may use a raw fallback; this does not promise globally
// optimal compression. It borrows disjoint storage and retains neither slice.
func Encode(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, ErrBlock
	}
	written := 0
	for len(src) > 0 {
		n := repeat(src)
		if n >= 3 {
			if len(dst)-written < 2 {
				return 0, ErrBlock
			}
			dst[written], dst[written+1] = byte(257-n), src[0]
			written += 2
		} else {
			n = literal(src)
			if len(dst)-written < n+1 {
				return 0, ErrBlock
			}
			dst[written] = byte(n - 1)
			copy(dst[written+1:], src[:n])
			written += n + 1
		}
		src = src[n:]
	}
	return written, nil
}

func repeat(src []byte) int {
	n := 1
	for n < min(len(src), 128) && src[n] == src[0] {
		n++
	}
	return n
}

func literal(src []byte) int {
	n := 1
	for n < min(len(src), 128) && repeat(src[n:]) < 3 {
		n++
	}
	return n
}
