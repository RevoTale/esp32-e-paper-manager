package devicelink

import (
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/update"
)

type Runtime interface {
	ApplyNetwork(update.Request) update.Result
	RecordNetwork(uint64, update.Result)
	NetworkReport() (update.Result, uint64, bool)
}

// Exchange runs one authenticated manager poll. The retained report is sent
// first, so a terminal physical-refresh result is not lost between polls.
func Exchange(connection io.ReadWriter, config DeviceConfig, random io.Reader, runtime Runtime) error {
	if runtime == nil {
		return ErrConfiguration
	}
	session, err := ConnectDevice(connection, config, random)
	if err != nil {
		return err
	}
	if err = reportRetained(session, runtime); err != nil {
		return err
	}
	request, available, err := session.Receive()
	if err != nil || !available {
		return err
	}
	result := runtime.ApplyNetwork(request)
	runtime.RecordNetwork(session.Generation(), result)
	return session.SendResult(result)
}

func reportRetained(session *DeviceSession, runtime Runtime) error {
	result, generation, available := runtime.NetworkReport()
	if !available {
		return session.Report(nil, 0)
	}
	return session.Report(&result, generation)
}
