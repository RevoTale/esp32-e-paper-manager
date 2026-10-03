package screenusbhost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == "--serial-proxy" {
		if os.Args[2] == "diagnostic" {
			_, _ = os.Stderr.WriteString("private-secret")
			os.Exit(23)
		}
		if os.Args[2] == "blocked" {
			time.Sleep(time.Hour)
		}
		_, err := io.Copy(os.Stdout, os.Stdin)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestOneSubprocessCarriesManyRecordsAndIsReaped(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 1)
	w, err := startProcess(executable, "echo", changed)
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 3; n++ {
		if _, err = w.Write([]byte("bounded")); err != nil {
			t.Fatal(err)
		}
		var b [7]byte
		if _, err = io.ReadFull(w, b[:]); err != nil || string(b[:]) != "bounded" {
			t.Fatal(err, b)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	assertReaped(t, w, changed)
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertReaped(t *testing.T, w worker, changed <-chan struct{}) {
	t.Helper()
	select {
	case <-w.Done():
	default:
		t.Fatal("not reaped")
	}
	select {
	case <-changed:
	case <-time.After(time.Second):
		t.Fatal("no lifecycle wake")
	}
}

func TestParentDeadlineKillsBlockedReadAndWrite(t *testing.T) {
	for _, write := range []bool{false, true} {
		t.Run(map[bool]string{false: "read", true: "write"}[write], func(t *testing.T) {
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			w, err := startProcess(executable, "blocked", make(chan struct{}, 1))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			err = operation(ctx, w, func() error {
				if write {
					_, err := w.Write(bytes.Repeat([]byte{1}, 1<<20))
					return err
				}
				var b [1]byte
				_, err := w.Read(b[:])
				return err
			})
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
			select {
			case <-w.Done():
			default:
				t.Fatal("deadline returned without reap")
			}
		})
	}
}

func TestMissingExecutableFailsWithoutWorker(t *testing.T) {
	if w, err := startProcess("/nonexistent/epaperscreen", "port", make(chan struct{}, 1)); w != nil || !errors.Is(err, ErrWorker) {
		t.Fatal(w, err)
	}
}
