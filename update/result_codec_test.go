package update

import (
	"errors"
	"testing"
)

func TestResultCodecRoundTripAndCorruption(t *testing.T) {
	accepted := Result{ID: ID{1}, Status: StatusAccepted, ContentHash: Digest{2}}
	rejected := Result{ID: ID{1}, Status: StatusRejected, Diagnostic: Diagnostic{
		Stage: StageParse, Code: CodeLimit, Limit: "nodes", Observed: 257, Maximum: 256,
	}}
	for _, original := range []Result{accepted, rejected} {
		wire := make([]byte, ResultEncodedSize)
		if size, err := EncodeResult(wire, original); err != nil || size != ResultEncodedSize {
			t.Fatalf("EncodeResult() size=%d error=%v", size, err)
		}
		decoded, err := DecodeResult(wire)
		if err != nil || decoded != original {
			t.Fatalf("DecodeResult() result=%#v error=%v", decoded, err)
		}
		wire[10] ^= 1
		if _, err := DecodeResult(wire); !errors.Is(err, ErrCodec) {
			t.Fatalf("DecodeResult(corrupt) error=%v", err)
		}
	}
}

func TestResultCodecRejectsBadStorageAndPadding(t *testing.T) {
	result := Result{ID: ID{1}, Status: StatusAccepted, ContentHash: Digest{2}}
	if _, err := EncodeResult(make([]byte, ResultEncodedSize-1), result); !errors.Is(err, ErrBuffer) {
		t.Fatalf("EncodeResult(storage) error=%v", err)
	}
	if _, err := EncodeResult(make([]byte, ResultEncodedSize), Result{}); !errors.Is(err, ErrResult) {
		t.Fatalf("EncodeResult(invalid) error=%v", err)
	}
	wire := make([]byte, ResultEncodedSize)
	_, _ = EncodeResult(wire, result)
	wire[99] = 1
	if _, err := DecodeResult(wire); !errors.Is(err, ErrCodec) {
		t.Fatalf("DecodeResult(padding) error=%v", err)
	}
}

func TestEncodedSizeFromHeader(t *testing.T) {
	request := testRequest(t)
	wire := make([]byte, EncodedSize(request))
	_, _ = Encode(wire, request)
	if size, err := EncodedSizeFromHeader(wire[:HeaderSize]); err != nil || size != len(wire) {
		t.Fatalf("EncodedSizeFromHeader() size=%d error=%v", size, err)
	}
	if _, err := EncodedSizeFromHeader(wire[:10]); !errors.Is(err, ErrCodec) {
		t.Fatalf("EncodedSizeFromHeader(short) error=%v", err)
	}
	wire[8], wire[9], wire[10], wire[11] = 0, 0, 0, 0
	if _, err := EncodedSizeFromHeader(wire[:HeaderSize]); !errors.Is(err, ErrCodec) {
		t.Fatalf("EncodedSizeFromHeader(length) error=%v", err)
	}
}
