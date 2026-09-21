package streamrx

import (
	"errors"
	"testing"
	"time"
)

func TestConfigurationAndIntentBounds(t *testing.T) {
	base := Config{Epoch: 1, Width: 8, Height: 1, Passes: 2, MaxChunk: 32, Idle: time.Second, Total: time.Second}
	for field := 0; field < 8; field++ {
		bad := base
		invalidateConfig(&bad, field)
		if _, err := New(bad, new(sink)); !errors.Is(err, ErrConfig) {
			t.Fatal(err)
		}
	}
	if _, err := New(base, nil); !errors.Is(err, ErrConfig) {
		t.Fatal(err)
	}
}

func invalidateConfig(bad *Config, field int) {
	switch field {
	case 0:
		bad.Epoch = 0
	case 1:
		bad.Width = 0
	case 2:
		bad.Height = 0
	case 3:
		bad.Passes = 0
	case 4:
		bad.Passes = 3
	case 5:
		bad.MaxChunk = 0
	case 6:
		bad.MaxChunk = 4097
	case 7:
		bad.Total = 0
	}
}

func TestIntentBounds(t *testing.T) {
	r, s, intent := fixture(t)
	wrong := intent
	wrong.Epoch++
	if err := r.Begin(wrong, 0); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Begin(intent, 0); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
	if err := r.Write(wrong, 0, 0, []byte{128}, 0); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
	if s.begin != 1 || s.writes != 0 || s.aborts != 1 {
		t.Fatal("wrong identity reached sink")
	}
}

func TestDuplicateChunkAndPostCompleteClose(t *testing.T) {
	r, s, intent := fixture(t)
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(intent, 0, 0, []byte{128}, 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(intent, 0, 0, []byte{128}, 0); !errors.Is(err, ErrChunk) {
		t.Fatal(err)
	}
	if s.writes != 1 {
		t.Fatal("duplicate reached sink")
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if s.aborts != 1 {
		t.Fatal("repeated abort")
	}
}

func TestSuccessfulTransfersAllocateNothingAfterConstruction(t *testing.T) {
	r, _, intent := fixture(t)
	data := []byte{128, 1}
	var failure error
	allocs := testing.AllocsPerRun(100, func() {
		intent.ID++
		if failure = r.Begin(intent, 0); failure != nil {
			return
		}
		for pass := uint8(0); pass < 2; pass++ {
			if failure = r.Write(intent, pass, 0, data, 0); failure != nil {
				return
			}
		}
		failure = r.Commit(intent, 0)
	})
	if failure != nil || allocs != 0 {
		t.Fatal(failure, allocs)
	}
}
