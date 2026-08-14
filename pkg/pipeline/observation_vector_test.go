package pipeline

import (
	"math"
	"testing"
)

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

// TestObservationVectorIsUnitNormalized confirms the scale-invariance fix:
// any non-empty observation embeds to a unit-length vector, regardless of how
// many tokens it has. This is what keeps the forward model's input magnitude
// bounded and forecast error comparable across observation sizes.
func TestObservationVectorIsUnitNormalized(t *testing.T) {
	for _, obs := range []string{"color3-cell0-0", "a b c d e f", "color3-cell0-0 north color5-cell1-1 aligned cmult3 cmult3 cmult3"} {
		v := ObservationVector(obs)
		norm := 0.0
		for _, x := range v {
			norm += x * x
		}
		if math.Abs(norm-1.0) > 1e-9 {
			t.Fatalf("FAIL [%q]: expected unit L2 norm, got squared-norm %.6f", obs, norm)
		}
	}
}

// TestObservationVectorRepetitionShiftsTowardToken confirms a repeated token
// still carries more weight -- post-normalization the exact "2x" is gone, but
// repeating a token must pull the (unit) observation vector MORE toward that
// token's own direction, i.e. accumulation still matters for the balance
// between distinct tokens.
func TestObservationVectorRepetitionShiftsTowardToken(t *testing.T) {
	dot := func(a, b []float64) float64 {
		s := 0.0
		for i := range a {
			s += a[i] * b[i]
		}
		return s
	}
	va := ObservationVector("aaa")               // unit direction of token "aaa" alone
	ab := ObservationVector("aaa bbb")           // aaa + bbb
	aab := ObservationVector("aaa aaa aaa bbb")  // aaa weighted 3x
	if !(dot(aab, va) > dot(ab, va)) {
		t.Fatalf("FAIL: weighting a token did not pull the vector toward it: dot(aab,va)=%.4f not > dot(ab,va)=%.4f (hash collision on the two tokens?)", dot(aab, va), dot(ab, va))
	}
}
