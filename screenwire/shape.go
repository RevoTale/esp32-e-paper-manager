package screenwire

type presence uint8

const (
	anyValue presence = iota
	zeroValue
	nonzeroValue
)

func (p presence) accepts(v uint64) bool {
	switch p {
	case zeroValue:
		return v == 0
	case nonzeroValue:
		return v != 0
	default:
		return true
	}
}

type shape struct {
	epoch, id presence
	length    int
}

// Negative lengths identify the only two variable-length record families.
var shapes = [...]shape{
	Hello:        {zeroValue, zeroValue, 0},
	Acquire:      {anyValue, zeroValue, 32},
	Bind:         {nonzeroValue, zeroValue, 32},
	Begin:        {nonzeroValue, nonzeroValue, 32},
	Data:         {nonzeroValue, nonzeroValue, -1},
	DataPacked:   {nonzeroValue, nonzeroValue, -1},
	Commit:       {nonzeroValue, nonzeroValue, 32},
	Query:        {nonzeroValue, nonzeroValue, 32},
	Abort:        {nonzeroValue, zeroValue, 0},
	Reply:        {anyValue, anyValue, -2},
	Health:       {zeroValue, zeroValue, 0},
	PanelTrace:   {zeroValue, zeroValue, 0},
	BeginRefresh: {nonzeroValue, nonzeroValue, 44},
	BeginRegion:  {nonzeroValue, nonzeroValue, RegionBeginSize},
}

func (r Record) validCoordinates() bool {
	if r.Kind == Data || r.Kind == DataPacked {
		return r.Pass < 2
	}
	return r.Pass == 0 && r.Offset == 0
}

func (r Record) validLength(want, n int) bool {
	if r.Kind == DataPacked {
		return n >= 4 // u16 decoded length plus at least one complete run.
	}
	if r.Kind == Data {
		return n > 0
	}
	if r.Kind == Reply {
		return n == StatusSize || n == StatusSize+CapabilitiesSize || n == StatusSize+HealthSize || n == StatusSize+PanelStatusSize
	}
	return n == want
}

// Request kinds are explicit: Reply sits between old requests and Health.
func (k Kind) request() bool {
	switch k {
	case Hello, Acquire, Bind, Begin, Data, Commit, Query, Abort, Health, DataPacked, PanelTrace, BeginRefresh, BeginRegion:
		return true
	default:
		return false
	}
}
