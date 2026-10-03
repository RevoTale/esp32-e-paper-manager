package provision

// Code is a stable, source-free USB result. State describes the authoritative
// journal read after the operation, including when Code reports a failed write.
type Code uint8

const (
	CodeOK Code = iota
	CodeConfiguration
	CodeState
	CodeStorage
	CodeBusy
	CodeRecovery // Boot lifetime unavailable; physical erase then reboot required.
)

func (codec Codec) validResponse(r Response) bool {
	if !validOperation(r.Operation) || r.Code > CodeRecovery || !validSuccess(r) {
		return false
	}
	switch r.State {
	case StateProvisioned:
		return codec.validMetadata(r)
	case StateBlank, StateCorrupt:
		return emptyMetadata(r)
	case StateUnknown:
		return r.Code == CodeStorage && emptyMetadata(r)
	default:
		return false
	}
}

func validSuccess(r Response) bool {
	if r.Code != CodeOK {
		return true
	}
	switch r.Operation {
	case OperationProvision, OperationRotate:
		return r.State == StateProvisioned
	case OperationErase:
		return r.State == StateBlank
	default:
		return true
	}
}

func (codec Codec) validMetadata(r Response) bool {
	return r.Generation > 0 && !allZero(r.DeviceID[:]) && r.Auth == codec.AuthMode() &&
		boundedText(r.SSID, 1, MaxSSID) && codec.validManager(r.Manager) && validTimezone(r.Timezone)
}

func emptyMetadata(r Response) bool {
	return r.Generation == 0 && r.Auth == 0 && allZero(r.DeviceID[:]) &&
		r.SSID == "" && r.Manager == "" && r.Timezone == ""
}

func requestPadding(wire []byte, request Request) bool {
	if !needsConfig(request.Operation) {
		return allZero(wire[6:508])
	}
	c := request.Config
	return allZero(wire[6:16]) && allZero(wire[22:24]) && allZero(wire[486:508]) &&
		allZero(wire[72+len(c.Manager):327]) && allZero(wire[327+len(c.SSID):359]) &&
		allZero(wire[359+len(c.Passphrase):422]) && allZero(wire[422+len(c.Timezone):486])
}

func responsePadding(wire []byte, r Response) bool {
	return allZero(wire[13:16]) && allZero(wire[391:508]) &&
		allZero(wire[40+len(r.SSID):72]) && allZero(wire[72+len(r.Manager):327]) &&
		allZero(wire[327+len(r.Timezone):391])
}
