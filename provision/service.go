package provision

import "errors"

var ErrInvalidState = errors.New("provision: invalid state transition")

type Service struct {
	store *Store
}

func NewService(store *Store) (*Service, error) {
	if store == nil {
		return nil, ErrStorage
	}
	return &Service{store: store}, nil
}

func (s *Service) Execute(request Request) (Response, error) {
	if s == nil || s.store == nil || !validOperation(request.Operation) {
		return Response{}, ErrInvalidConfig
	}
	switch request.Operation {
	case OperationInspect, OperationDiagnose:
		return s.inspect(request.Operation)
	case OperationProvision:
		return s.provision(request)
	case OperationRotate:
		return s.rotate(request)
	case OperationErase:
		if err := s.store.FactoryReset(); err != nil {
			return Response{}, err
		}
		return Response{Operation: request.Operation, State: StateBlank}, nil
	default:
		return Response{}, ErrInvalidConfig
	}
}

func (s *Service) provision(request Request) (Response, error) {
	_, _, err := s.store.Load()
	if err == nil || !errors.Is(err, ErrBlank) && !errors.Is(err, ErrCorrupt) {
		return Response{}, ErrInvalidState
	}
	generation, err := s.store.Save(request.Config)
	if err != nil {
		return Response{}, err
	}
	return publicResponse(request.Operation, request.Config, generation), nil
}

func (s *Service) rotate(request Request) (Response, error) {
	current, _, err := s.store.Load()
	if err != nil || current.DeviceID != request.Config.DeviceID {
		return Response{}, ErrInvalidState
	}
	generation, err := s.store.Save(request.Config)
	if err != nil {
		return Response{}, err
	}
	return publicResponse(request.Operation, request.Config, generation), nil
}

func (s *Service) inspect(operation Operation) (Response, error) {
	config, generation, err := s.store.Load()
	if errors.Is(err, ErrBlank) {
		return Response{Operation: operation, State: StateBlank}, nil
	}
	if errors.Is(err, ErrCorrupt) {
		return Response{Operation: operation, State: StateCorrupt}, nil
	}
	if err != nil {
		return Response{}, err
	}
	return publicResponse(operation, config, generation), nil
}

func publicResponse(operation Operation, config Config, generation uint64) Response {
	return Response{Operation: operation, State: StateProvisioned, Generation: generation,
		Auth: config.Auth, DeviceID: config.DeviceID, SSID: config.SSID,
		Manager: config.Manager, Timezone: config.Timezone}
}
