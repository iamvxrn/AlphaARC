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
	// eligibility is a decaying trace per label -- how recently/repeatedly it
	// was clicked -- so a sparse reward (a level completion) can be credited
	// BACK along the recent action sequence, most to the last click, less to
	// earlier ones. This is branch C: the only ground-truth reward is
	// levels_completed, which is rare and has never fired live, so when it
	// finally does the whole sequence that produced it must stick, not just
	// the single last click.
	eligibility map[string]float64
}

// eligibilityDecay is how fast a label's credit trace fades each click
// (0.7 -> a click ~3 steps back still carries ~1/3 the credit of the last).
const eligibilityDecay = 0.7

// NewOutcomeMemory returns an empty memory, ready to use.
func NewOutcomeMemory() *OutcomeMemory {
	return &OutcomeMemory{
		attempts:    make(map[string]int),
		successes:   make(map[string]int),
		eligibility: make(map[string]float64),
	}
}

// Record logs one outcome for label. A blank label (nothing was clicked
// yet, e.g. the very first call in a session) is silently ignored rather
// than polluting the memory with a meaningless empty-string entry.
func (m *OutcomeMemory) Record(label string, success bool) {
	if label == "" {
		return
	}
	// Age every eligibility trace, then mark this label as most-recent.
	for l := range m.eligibility {
		m.eligibility[l] *= eligibilityDecay
	}
	m.attempts[label]++
	if success {
		m.successes[label]++
	}
	m.eligibility[label] += 1.0
}

// ReinforceLevelCompletion credits a sparse reward (a completed level) back
// along the recent action sequence: each label gets a success-and-attempt
// boost proportional to its eligibility trace, so the clicks that led up to
// the win -- most of all the last one -- become strongly "proven" and get
// repeated. strength scales the boost (successes added to the last click ~=
// strength). Traces are consumed (reset) so one win is credited once.
func (m *OutcomeMemory) ReinforceLevelCompletion(strength float64) {
	for label, elig := range m.eligibility {
		credit := int(strength*elig + 0.5)
		if credit > 0 {
			m.successes[label] += credit
			m.attempts[label] += credit
		}
		m.eligibility[label] = 0
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

// PenalizeSequence is the mirror of ReinforceLevelCompletion for a LOSS
// (GAME_OVER): it credits FAILURE back along the recent click sequence --
// each label gets attempts (but no successes) proportional to its eligibility
// trace, dropping its success rate, most for the last click. So the path that
// led to death becomes less "proven" and gets avoided on the retry. Traces are
// consumed so one loss is credited once.
func (m *OutcomeMemory) PenalizeSequence(strength float64) {
	for label, elig := range m.eligibility {
		credit := int(strength*elig + 0.5)
		if credit > 0 {
			m.attempts[label] += credit // failures: attempts up, successes flat -> rate falls
		}
		m.eligibility[label] = 0
	}
}
