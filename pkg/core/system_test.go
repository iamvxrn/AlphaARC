package core

import "testing"

func TestFSMGuards(t *testing.T) {
	sys := NewSystem()

	// Should pass
	sys.OnlineGuard("OnlineOp")

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when calling OfflineGuard while in ONLINE mode")
		}
	}()
	sys.OfflineGuard("OfflineOp")
}
