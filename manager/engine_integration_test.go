package manager

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamwire"
)

type engineSink struct {
	planes    [2][]byte
	committed bool
}

func (s *engineSink) Begin() error { s.planes = [2][]byte{}; return nil }
func (s *engineSink) Write(pass uint8, offset uint32, data []byte) error {
	if pass > 1 || int(offset) != len(s.planes[pass]) {
		return errors.New("invalid plane")
	}
	s.planes[pass] = append(s.planes[pass], data...)
	return nil
}
func (s *engineSink) Commit() error { s.committed = true; return nil }
func (*engineSink) Abort() error    { return nil }

type enginePort struct {
	link    *streamwire.Link
	replies bytes.Buffer
}

func (p *enginePort) Write(data []byte) (int, error) {
	err := p.link.Push(data, 0, func(reply []byte) error { _, err := p.replies.Write(reply); return err })
	return len(data), err
}
func (p *enginePort) Read(data []byte) (int, error) { return p.replies.Read(data) }

func TestHTTPSNativeEngineAndUSBCodecMatchPreview(t *testing.T) {
	size := display.Size{Width: 17, Height: 9}
	r, screen, api := nativeScreen(t, size)
	server := httptest.NewTLSServer(api)
	defer server.Close()
	etag := engineHTTPRequest(t, server, http.MethodGet, "", "", 200)
	source := `<div style="width:50vw;height:100vh;background:black"></div>`
	engineHTTPRequest(t, server, http.MethodPut, etag, source, 202)
	delivery, err := screen.RenderNext(context.Background(), time.Second, true)
	if err != nil || delivery.Revision != 1 {
		t.Fatalf("%+v %v", delivery, err)
	}
	preview, err := r.Render(context.Background(), size, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	sink := transmitEngineFrame(t, delivery.Frame)
	if !sink.committed {
		t.Fatal("USB receiver never committed")
	}
	for _, plane := range sink.planes {
		if !bytes.Equal(plane, preview.Bytes()) {
			t.Fatal("USB plane differs from native preview")
		}
	}
	if err = screen.Resolve(delivery.Revision, true); err != nil || screen.Status().Confirmed != 1 {
		t.Fatalf("completion %v %+v", err, screen.Status())
	}
}

func nativeScreen(t *testing.T, size display.Size) (*engine.Renderer, *Screen, *ScreenAPI) {
	t.Helper()
	r, err := engine.New()
	if err != nil {
		t.Fatal(err)
	}
	screen, err := NewScreen(r, size, renderbatch.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	api, err := NewScreenAPI(screen, []byte(strings.Repeat("t", 32)), [16]byte{1}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return r, screen, api
}

func transmitEngineFrame(t *testing.T, frame display.Frame) *engineSink {
	t.Helper()
	sink := new(engineSink)
	link, err := streamwire.NewLink(streamrx.Config{Epoch: 1, Width: uint16(frame.Size().Width), Height: uint16(frame.Size().Height), Passes: 2, MaxChunk: 3, Idle: time.Second, Total: time.Second}, sink, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err = streamwire.Send(&enginePort{link: link}, 7, frame); err != nil {
		t.Fatal(err)
	}
	return sink
}

func engineHTTPRequest(t *testing.T, server *httptest.Server, method, etag, source string, want int) string {
	t.Helper()
	request, err := http.NewRequest(method, server.URL+"/v2/screen", strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	request.Header.Set("Content-Type", "text/html; charset=utf-8")
	if etag != "" {
		request.Header.Set("If-Match", etag)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != want {
		t.Fatalf("HTTPS %d %q %v", response.StatusCode, body, err)
	}
	return response.Header.Get("ETag")
}
