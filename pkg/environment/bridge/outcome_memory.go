package bridge

// OutcomeMemory tracks, per blob-category label, how often clicking it
// actually caused a change -- explicit, persistent memory independent of
// the Hebbian/Louvain graph dynamics driving winningBlobLabel. That
// independence matters: a real live run showed the SAME category's
// standing scrambled across sleep cycles (Louvain re-clustering, edge
// pruning) even when nothing about the world had actually changed --
// structural graph state is not the same thing as accumulated evidence.
//
// This is a deliberately modest first step toward "does this thing
// actually work," not a causal/relational world model: it tracks raw
// frequency per label, not state-dependent outcomes (the same label's true
// effect could differ depending on what else is going on in the game;
// this doesn't distinguish that yet -- see ChooseClickAction's doc comment
// for where this sits in the bigger picture).
type OutcomeMemory struct {
	attempts  map[string]int
	successes map[string]int
}

// NewOutcomeMemory returns an empty memory, ready to use.
func NewOutcomeMemory() *OutcomeMemory {
	return &OutcomeMemory{
		attempts:  make(map[string]int),
		successes: make(map[string]int),
	}
}

// Record logs one outcome for label. A blank label (nothing was clicked
// yet, e.g. the very first call in a session) is silently ignored rather
// than polluting the memory with a meaningless empty-string entry.
func (m *OutcomeMemory) Record(label string, success bool) {
	if label == "" {
		return
	}
	m.attempts[label]++
	if success {
		m.successes[label]++
	}
}

// SuccessRate returns label's observed success rate and how many attempts
// it's based on. (0, 0) for a label never recorded -- callers must check
// attempts before trusting rate, since rate alone can't distinguish "never
// tried" from "tried and always failed" (both report rate 0).
func (m *OutcomeMemory) SuccessRate(label string) (rate float64, attempts int) {
	a := m.attempts[label]
	if a == 0 {
		return 0, 0
	}
	return float64(m.successes[label]) / float64(a), a
}
