package screenusbhost

import (
	"context"

	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type Snapshot struct {
	Capabilities screenwire.Capabilities
	Health       screenwire.HealthStatus
	Status       screenwire.Status
}

// Inspect performs read-only Hello and Health without acquiring or binding a
// lease. Opening USB/DTR still invokes normal USB priority and may abort an
// incomplete Wi-Fi upload. It never renders, uploads, refreshes or retries.
func (s *Sender) Inspect(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	p, err := s.process()
	if err != nil {
		return Snapshot{}, err
	}
	var result Snapshot
	err = operation(ctx, p, func() error { var err error; result, err = inspect(p); return err })
	if err != nil {
		return result, translate(err)
	}
	return result, nil
}

func inspect(p worker) (Snapshot, error) {
	var data [screenwire.MaxRecord]byte
	hello := screenwire.Record{Kind: screenwire.Hello}
	n, err := screenwire.Encode(data[:], hello)
	if err != nil {
		return Snapshot{}, err
	}
	if err = writeAll(p, data[:n]); err != nil {
		return Snapshot{}, err
	}
	n, err = readRecord(p, data[:])
	if err != nil {
		return Snapshot{}, err
	}
	r, err := screenwire.Decode(data[:n])
	if err != nil {
		return Snapshot{}, err
	}
	reply, err := screenwire.ParseReply(r, hello)
	if err != nil {
		return Snapshot{}, err
	}
	if reply.Status.Code != screenwire.CodeOK {
		return Snapshot{}, screenclient.RemoteError{Status: reply.Status}
	}
	health, err := screenclient.ReadHealth(p)
	if err != nil {
		return Snapshot{}, err
	}
	if health.Status.Boot != reply.Status.Boot || health.Status.Generation != reply.Status.Generation {
		return Snapshot{}, screenclient.ErrResync
	}
	return Snapshot{Capabilities: reply.Capabilities, Health: health.Health, Status: health.Status}, nil
}
