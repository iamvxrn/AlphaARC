package conflict

import "testing"

func TestResolvePicksHighestTrustScore(t *testing.T) {
	candidates := []Candidate{
		{Source: "pred-cluster-1", ClusterID: 1, TrustScore: 0.9, Confidence: 0.5},
		{Source: "pred-cluster-2", ClusterID: 2, TrustScore: 1.4, Confidence: 0.5},
		{Source: "pred-cluster-3", ClusterID: 3, TrustScore: 1.1, Confidence: 0.5},
	}

	r := Resolve(7, candidates)
	if r == nil {
		t.Fatalf("FAIL: expected a non-nil Record")
	}
	if r.WinnerIdx != 1 {
		t.Fatalf("FAIL: expected candidate index 1 (trust 1.4) to win, got index %d", r.WinnerIdx)
	}
	if r.Winner().Source != "pred-cluster-2" {
		t.Fatalf("FAIL: expected winner Source 'pred-cluster-2', got %q", r.Winner().Source)
	}
	if len(r.Candidates) != 3 {
		t.Fatalf("FAIL: expected all 3 candidates preserved in the record, got %d", len(r.Candidates))
	}

	t.Logf("Resolve PASS: winner=%s (trust=%.2f) among %d candidates, reason=%q", r.Winner().Source, r.Winner().TrustScore, len(r.Candidates), r.Reason)
}

func TestResolveTiedTrustScoreBreaksOnConfidence(t *testing.T) {
	candidates := []Candidate{
		{Source: "a", TrustScore: 1.0, Confidence: 0.4},
		{Source: "b", TrustScore: 1.0, Confidence: 0.9},
	}

	r := Resolve(1, candidates)
	if r.Winner().Source != "b" {
		t.Fatalf("FAIL: expected 'b' to win on higher confidence at tied trust score, got %q", r.Winner().Source)
	}
}

func TestResolveNeverDiscardsLosingCandidates(t *testing.T) {
	candidates := []Candidate{
		{Source: "winner", TrustScore: 2.0},
		{Source: "loser-1", TrustScore: 0.5},
		{Source: "loser-2", TrustScore: 0.1},
	}

	r := Resolve(1, candidates)
	sources := make(map[string]bool)
	for _, c := range r.Candidates {
		sources[c.Source] = true
	}
	for _, want := range []string{"winner", "loser-1", "loser-2"} {
		if !sources[want] {
			t.Fatalf("FAIL: expected losing candidate %q to still be present in the Record (CONCEPT.md Section 16: don't erase losing positions), got %v", want, sources)
		}
	}
}

func TestResolveEmptyCandidatesReturnsNil(t *testing.T) {
	if r := Resolve(1, nil); r != nil {
		t.Fatalf("FAIL: expected nil Record for empty candidates, got %v", r)
	}
}

func TestResolveSingleCandidateIsUncontested(t *testing.T) {
	r := Resolve(1, []Candidate{{Source: "only", TrustScore: 1.0}})
	if r.WinnerIdx != 0 || r.Winner().Source != "only" {
		t.Fatalf("FAIL: expected the sole candidate to win trivially")
	}
	if r.Reason != "sole candidate, no competition" {
		t.Fatalf("FAIL: expected reason to reflect no real competition, got %q", r.Reason)
	}
}
