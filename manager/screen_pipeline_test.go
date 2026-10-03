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
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamwire"
)

// This is the real codec/receiver with a reusable in-memory sink, not a claim
// about physical SPI/BUSY or visible e-paper pixels.
type pipelineSink struct {
	planes  [2][]byte
	commits int
}

func (*pipelineSink) Begin() error { return nil }
func (s *pipelineSink) Write(pass uint8, _ uint32, b []byte) error {
	s.planes[pass] = append(s.planes[pass], b...)
	return nil
}
func (s *pipelineSink) Commit() error { s.commits++; return nil }
func (*pipelineSink) Abort() error    { return nil }

type pipelineLink struct {
	link  *streamwire.Link
	reply bytes.Buffer
}

func (l *pipelineLink) Write(b []byte) (int, error) {
	err := l.link.Push(b, 0, func(p []byte) error { _, err := l.reply.Write(p); return err })
	return len(b), err
}
func (l *pipelineLink) Read(b []byte) (int, error) { return l.reply.Read(b) }

func TestHTTPSSceneToRealEPS1Receiver(t *testing.T) {
	a, s := newScreenAPI(t)
	queueThroughTLS(t, a)
	runPipeline(t, s)
}

func queueThroughTLS(t *testing.T, a *ScreenAPI) {
	t.Helper()
	server := httptest.NewTLSServer(a)
	defer server.Close()
	get := authenticatedHTTP(t, server, "GET", "", "")
	etag := get.Header.Get("ETag")
	if err := get.Body.Close(); err != nil {
		t.Fatal(err)
	}
	put := authenticatedHTTP(t, server, "PUT", etag, "<p>one scene</p>")
	if put.StatusCode != 202 {
		t.Fatal(put.StatusCode)
	}
	if err := put.Body.Close(); err != nil {
		t.Fatal(err)
	}
}

func runPipeline(t *testing.T, s *Screen) {
	t.Helper()
	var sink pipelineSink
	link, err := streamwire.NewLink(streamrx.Config{Epoch: 1, Width: 8, Height: 1, Passes: 2, MaxChunk: 1, Idle: time.Second, Total: time.Second}, &sink, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, err := NewScreenPump(s, screenSender(func(_ context.Context, frame display.Frame) error {
		err := streamwire.Send(&pipelineLink{link: link}, 7, frame)
		cancel()
		return err
	}), time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if sink.commits != 1 || !bytes.Equal(sink.planes[0], []byte{0}) || !bytes.Equal(sink.planes[1], []byte{0}) || s.Status().Confirmed != 1 {
		t.Fatal(sink, s.Status())
	}
}

func authenticatedHTTP(t *testing.T, server *httptest.Server, method, etag, body string) *http.Response {
	t.Helper()
	r, err := http.NewRequest(method, server.URL+"/v2/screen", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	r.Header.Set("Content-Type", "text/html")
	if etag != "" {
		r.Header.Set("If-Match", etag)
	}
	response, err := server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	return response
}
