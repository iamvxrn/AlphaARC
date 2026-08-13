package pipeline

import "testing"

func TestObservationVectorLengthIsAlwaysFixed(t *testing.T) {
	cases := []string{"", "color3-cell0-0", "color3-cell0-0 north color5-cell1-1 aligned"}
	for _, obs := range cases {
		if got := len(ObservationVector(obs)); got != ObservationVectorDim {
			t.Fatalf("FAIL [%q]: expected length %d, got %d", obs, ObservationVectorDim, got)
		}
	}
}

func TestObservationVectorEmptyIsZeroVector(t *testing.T) {
	vec := ObservationVector("")
	for i, v := range vec {
		if v != 0 {
			t.Fatalf("FAIL: expected all-zero vector for empty observation, got nonzero at index %d: %v", i, vec)
		}
	}
}

// TestObservationVectorIsDeterministic is the core property a real forward
// model needs and the pseudo-random stateVector it's meant to replace
// (predictive_loop.go) does NOT have: the same content must always produce
// the same vector, call after call, regardless of e.StepCounter or any
// other engine state -- ObservationVector takes no such state at all.
func TestObservationVectorIsDeterministic(t *testing.T) {
	obs := "color3-cell0-0 north color5-cell1-1"
	first := ObservationVector(obs)
	for i := 0; i < 5; i++ {
		got := ObservationVector(obs)
		for j := range first {
			if got[j] != first[j] {
				t.Fatalf("FAIL: call %d differs from call 0 at index %d: %v vs %v", i, j, got, first)
			}
		}
	}
}

// TestObservationVectorDiffersForDifferentObservations confirms the whole
// point of this function: two observations with different content must not
// collapse to the same vector (the pseudo-random baseline this replaces
// would happily do that, or the reverse -- return different vectors for the
// literal same observation across calls, since it's keyed on stepCounter).
func TestObservationVectorDiffersForDifferentObservations(t *testing.T) {
	a := ObservationVector("color3-cell0-0 north")
	b := ObservationVector("color9-cell7-2 south aligned")
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("FAIL: expected different observations to produce different vectors, both got %v", a)
	}
}

// TestObservationVectorAccumulatesRepeatedTokens confirms a token repeated
// in the same observation adds up on its dimension rather than being
// deduplicated or overwritten -- e.g. two blobs sharing a category label
// should carry more weight than a single occurrence, matching how a real
// bag-of-tokens embedding is expected to behave.
func TestObservationVectorAccumulatesRepeatedTokens(t *testing.T) {
	single := ObservationVector("color3-cell0-0")
	doubled := ObservationVector("color3-cell0-0 color3-cell0-0")
	for i := range single {
		if doubled[i] != 2*single[i] {
			t.Fatalf("FAIL: expected repeating the same token to double its dimension's value at index %d: single=%v doubled=%v", i, single, doubled)
		}
	}
}
