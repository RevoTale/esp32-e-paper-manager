package screenwire

type Response struct {
	Status       Status
	Capabilities Capabilities // populated only for Hello
	Health       HealthStatus // populated only for Health
	Panel        PanelStatus  // populated only for PanelTrace
}

// ParseReply is mandatory after generic Decode on the client. It correlates
// the reply with the exact request, including operation-specific body length.
// The client must additionally bind boot/generation to its remembered lease.
func ParseReply(reply, request Record) (Response, error) {
	if !replyMatches(reply, request) {
		return Response{}, ErrRecord
	}
	s, err := DecodeStatus(reply.Payload[:StatusSize])
	if err != nil || s.Operation != request.Kind {
		return Response{}, ErrRecord
	}
	r := Response{Status: s}
	switch request.Kind {
	case Hello:
		r.Capabilities, err = DecodeCapabilities(reply.Payload[StatusSize:])
	case Health:
		r.Health, err = DecodeHealth(reply.Payload[StatusSize:])
	case PanelTrace:
		r.Panel, err = DecodePanelStatus(reply.Payload[StatusSize:])
	}
	if err != nil {
		return Response{}, err
	}
	return r, nil
}

func replyMatches(reply, request Record) bool {
	if !request.Kind.request() || !request.validShape(len(request.Payload)) {
		return false
	}
	if reply.Kind != Reply || reply.Pass != 0 || reply.Offset != 0 {
		return false
	}
	if reply.Epoch != request.Epoch || reply.ID != request.ID {
		return false
	}
	return len(reply.Payload) == replySize(request.Kind)
}

func replySize(kind Kind) int {
	switch kind {
	case Hello:
		return StatusSize + CapabilitiesSize
	case Health:
		return StatusSize + HealthSize
	case PanelTrace:
		return StatusSize + PanelStatusSize
	default:
		return StatusSize
	}
}
