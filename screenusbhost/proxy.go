package screenusbhost

import (
	"errors"
	"io"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"go.bug.st/serial"
)

type serialPort interface {
	io.ReadWriteCloser
	SetDTR(bool) error
	SetReadTimeout(time.Duration) error
}

var openPort = func(name string) (serialPort, error) {
	return serial.Open(name, &serial.Mode{BaudRate: 115200})
}

// SerialProxy is the supervised child's only operation: one open CDC port,
// complete bounded EPS2 records, no rendering, fallback, retry or stdout text.
// The parent must supervise it: serial.Close alone does not bound kernel writes.
func SerialProxy(name string, input io.Reader, output io.Writer) (result error) {
	if esp32Endpoint(name) {
		return esp32Proxy(name, input, output)
	}
	p, err := openPort(name)
	if err != nil {
		return ErrWorker
	}
	defer func() { result = errors.Join(result, p.Close()) }()
	if err = p.SetReadTimeout(180 * time.Second); err != nil {
		return ErrWorker
	}
	if err = p.SetDTR(true); err != nil {
		return ErrWorker
	}
	defer func() { result = errors.Join(result, p.SetDTR(false)) }()
	return serveProxy(input, output, p)
}

func serveProxy(input io.Reader, output io.Writer, p io.ReadWriter) error {
	var data [screenwire.MaxRecord]byte
	for {
		n, err := readRecord(input, data[:])
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err = proxyExchange(p, output, data[:], n); err != nil {
			return err
		}
	}
}

func proxyExchange(p io.ReadWriter, output io.Writer, data []byte, n int) error {
	request, err := screenwire.Decode(data[:n])
	if err != nil {
		return err
	}
	if request.Kind == screenwire.Reply {
		return screenwire.ErrRecord
	}
	if err = writeAll(p, data[:n]); err != nil {
		return err
	}
	// Preserve the original payload length for request-shape validation.
	// ParseReply uses header fields and length, never the aliased payload bytes.
	n, err = readRecord(p, data)
	if err != nil {
		return err
	}
	reply, err := screenwire.Decode(data[:n])
	if err != nil {
		return err
	}
	if _, err = screenwire.ParseReply(reply, request); err != nil {
		return err
	}
	return writeAll(output, data[:n])
}

type progressReader struct{ io.Reader }

func (r progressReader) Read(data []byte) (int, error) {
	n, err := r.Reader.Read(data)
	if n == 0 && err == nil && len(data) > 0 {
		return 0, io.ErrNoProgress
	}
	return n, err
}

func readRecord(input io.Reader, data []byte) (int, error) {
	if _, err := io.ReadFull(progressReader{input}, data[:screenwire.HeaderSize]); err != nil {
		return 0, err
	}
	n, err := screenwire.Size(data[:screenwire.HeaderSize])
	if err != nil {
		return 0, err
	}
	if _, err = io.ReadFull(progressReader{input}, data[screenwire.HeaderSize:n]); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	return n, nil
}

func writeAll(output io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := output.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
