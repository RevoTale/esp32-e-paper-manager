package provision

import "errors"

// Report executes once, then reloads the journal even after an error. Save may
// fail after writing a valid slot; FactoryReset can erase only one of two slots
// before failure. Only the reload, never the requested config, is authoritative.
// Physical USB ownership/credential fencing belongs to the caller.
// Config contains secrets for the owner; only Response may be serialized.
func (s *Service) Report(request Request) (Response, Config) {
	performed, operationErr := s.Execute(request)
	response, config := s.Snapshot(request.Operation)
	if response.Code == CodeOK {
		response.Code = resultCode(operationErr)
		if operationErr == nil && !postcondition(request, performed, response, config) {
			response.Code = CodeStorage
		}
	}
	return response, config
}

func postcondition(request Request, performed, loaded Response, config Config) bool {
	switch request.Operation {
	case OperationProvision, OperationRotate:
		return loaded.State == StateProvisioned && config == request.Config && loaded.Generation == performed.Generation
	case OperationErase:
		return loaded.State == StateBlank
	default:
		return true
	}
}

// Snapshot is read-only. Unknown means an I/O failure, not an erased journal.
func (s *Service) Snapshot(operation Operation) (Response, Config) {
	response := Response{Operation: operation, State: StateUnknown, Code: CodeStorage}
	if s == nil || s.store == nil {
		return response, Config{}
	}
	config, generation, err := s.store.Load()
	switch {
	case err == nil:
		return publicResponse(operation, config, generation), config
	case errors.Is(err, ErrBlank):
		response.State, response.Code = StateBlank, CodeOK
	case errors.Is(err, ErrCorrupt):
		response.State, response.Code = StateCorrupt, CodeOK
	}
	return response, Config{}
}

func resultCode(err error) Code {
	switch {
	case err == nil:
		return CodeOK
	case errors.Is(err, ErrInvalidState):
		return CodeState
	case errors.Is(err, ErrInvalidConfig):
		return CodeConfiguration
	default:
		return CodeStorage
	}
}
