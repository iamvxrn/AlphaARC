// Package conflict implements CONCEPT.md Section 16's conflict-resolution
// model: when specialists disagree, the winner is picked by explicit
// criteria (source reliability first, since that's the signal already
// available in this system: Agent.TrustScore), never by naive majority
// vote -- and critically, every candidate is kept in the resulting Record,
// not just the winner, since "the system must be able to keep a conflict as
// a memory object, rather than forcibly erasing one of the positions."
package conflict

// Candidate is one competing position in a conflict -- typically a
// specialist agent's prediction for the current cycle.
type Candidate struct {
	Source     string // e.g. the specialist agent's ID
	ClusterID  int
	Value      []float64
	TrustScore float64
	Confidence float64
}

// Record is a resolved conflict. It keeps every candidate that competed,
// not just the winner -- CONCEPT.md Section 16's memory-object requirement.
type Record struct {
	Step       int
	Candidates []Candidate
	WinnerIdx  int
	Reason     string
}

// Winner returns the candidate that won this conflict.
func (r *Record) Winner() Candidate {
	return r.Candidates[r.WinnerIdx]
}

// Resolve picks a winner among candidates by source reliability
// (TrustScore) first, Confidence as a tie-break, and preserves every
// candidate in the returned Record. Returns nil for an empty candidate
// list. A single candidate is still recorded (a trivial, uncontested
// "conflict") so the history stays consistent regardless of how many
// specialists were actually competing that cycle.
func Resolve(step int, candidates []Candidate) *Record {
	if len(candidates) == 0 {
		return nil
	}

	winnerIdx := 0
	best := candidates[0]
	for i := 1; i < len(candidates); i++ {
		c := candidates[i]
		if c.TrustScore > best.TrustScore ||
			(c.TrustScore == best.TrustScore && c.Confidence > best.Confidence) {
			best = c
			winnerIdx = i
		}
	}

	reason := "highest trust score (confidence as tie-break)"
	if len(candidates) == 1 {
		reason = "sole candidate, no competition"
	}

	return &Record{
		Step:       step,
		Candidates: candidates,
		WinnerIdx:  winnerIdx,
		Reason:     reason,
	}
}
