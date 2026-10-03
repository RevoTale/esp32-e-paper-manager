package screenlink

import (
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

func (c *Connection) respond(request screenwire.Record, result streamsession.Result, cause error, write func([]byte) error) error {
	if cause == nil {
		cause = result.Status.Failure
	}
	epoch := c.device.session.Epoch()
	status := screenwire.Status{Operation: request.Kind, Code: errorCode(cause), State: result.Status.State,
		Pass: result.Status.Pass, Offset: result.Status.Offset, CurrentImage: result.CurrentImage, Generation: epoch.Generation, Boot: epoch.Boot}
	remaining := c.device.session.Cooldown()
	status.CooldownMS = uint32((remaining + time.Millisecond - 1) / time.Millisecond)
	if cause != nil && c.device.diagnose != nil {
		status.Diagnostic = c.device.diagnose(cause)
	}
	if err := screenwire.EncodeStatus(c.body[:screenwire.StatusSize], status); err != nil {
		return err
	}
	size, err := c.metadata(request.Kind)
	if err != nil {
		return err
	}
	n, err := screenwire.Encode(c.reply[:], screenwire.Record{Kind: screenwire.Reply, Epoch: request.Epoch, ID: request.ID, Payload: c.body[:size]})
	if err != nil {
		return err
	}
	return write(c.reply[:n])
}

func (c *Connection) metadata(kind screenwire.Kind) (int, error) {
	size := screenwire.StatusSize
	switch kind {
	case screenwire.Hello:
		if err := screenwire.EncodeCapabilities(c.body[size:], c.device.caps); err != nil {
			return 0, err
		}
		size += screenwire.CapabilitiesSize
	case screenwire.Health:
		health := screenwire.HealthStatus{Version: screenwire.HealthVersion}
		if c.device.health != nil {
			health = c.device.health()
		}
		if err := screenwire.EncodeHealth(c.body[size:size+screenwire.HealthSize], health); err != nil {
			return 0, err
		}
		size += screenwire.HealthSize
	case screenwire.PanelTrace:
		trace := screenwire.PanelStatus{Version: 1}
		if c.device.panelTrace != nil {
			trace = c.device.panelTrace()
		}
		if err := screenwire.EncodePanelStatus(c.body[size:size+screenwire.PanelStatusSize], trace); err != nil {
			return 0, err
		}
		size += screenwire.PanelStatusSize
	}
	return size, nil
}

var codes = [...]struct {
	err  error
	code screenwire.Code
}{
	{screenwire.ErrRecord, screenwire.CodeRecord},
	{streamsession.ErrLease, screenwire.CodeLease},
	{streamsession.ErrBusy, screenwire.CodeBusy},
	{streamsession.ErrStale, screenwire.CodeStale},
	{streamsession.ErrConflict, screenwire.CodeConflict},
	{streamsession.ErrCooldown, screenwire.CodeCooldown},
	{streamrx.ErrState, screenwire.CodeState},
	{streamrx.ErrChunk, screenwire.CodeChunk},
	{streamrx.ErrDigest, screenwire.CodeDigest},
	{streamrx.ErrTimeout, screenwire.CodeTimeout},
	{streamrx.ErrConfig, screenwire.CodeConfig},
}

func errorCode(err error) screenwire.Code {
	if err == nil {
		return screenwire.CodeOK
	}
	for _, entry := range codes {
		if errors.Is(err, entry.err) {
			return entry.code
		}
	}
	return screenwire.CodeHardware
}
