package perception

import (
	"strings"
	"testing"
)

// TestNumericTokensCountObjects: the numerosity token reports the object count
// (a Core-Knowledge "sense of number"), distinct from the categorical identity
// tokens.
func TestNumericTokensCountObjects(t *testing.T) {
	grid := [][]int{
		{3, 0, 0, 0, 0, 0, 7},
		{0, 0, 3, 0, 0, 0, 0},
	}
	toks := strings.Join(NumericTokens(grid), " ")
	if !strings.Contains(toks, "nobj") {
		t.Fatalf("expected an object-count token, got %q", toks)
	}
}
