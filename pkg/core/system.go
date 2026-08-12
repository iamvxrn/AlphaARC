package core

import "fmt"

type SystemMode int

const (
	Online SystemMode = iota
	Offline
)

func (m SystemMode) String() string {
	if m == Online {
		return "ONLINE"
	}
	return "OFFLINE"
}

type System struct {
	Mode SystemMode
	Tick int
}

func NewSystem() *System {
	return &System{
		Mode: Online,
		Tick: 0,
	}
}

// OnlineGuard panics if called outside ONLINE mode.
func (s *System) OnlineGuard(fn string) {
	if s.Mode != Online {
		panic(fmt.Sprintf("MODE VIOLATION: %s requires ONLINE, current mode=%s", fn, s.Mode))
	}
}

// OfflineGuard panics if called outside OFFLINE mode.
func (s *System) OfflineGuard(fn string) {
	if s.Mode != Offline {
		panic(fmt.Sprintf("MODE VIOLATION: %s requires OFFLINE, current mode=%s", fn, s.Mode))
	}
}
