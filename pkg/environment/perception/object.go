package perception

// moveDirection names the dominant axis of a move (rows are y; up = smaller y).
// Kept as the shared helper the ObjectTracker uses to label object motion; the
// former shape-hash object primitives (ObjectSignature/ObjectTokens/
// ObjectMotions) were removed in the consolidation pass -- they were superseded
// by the continuity-based ObjectTracker (the brittle exact-pixel-shape identity
// churned on any deformation), so nothing called them anymore.
func moveDirection(dx, dy int) string {
	if abs(dy) >= abs(dx) {
		if dy < 0 {
			return "up"
		}
		return "down"
	}
	if dx < 0 {
		return "left"
	}
	return "right"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
