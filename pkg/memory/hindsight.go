// Package memory already holds Hopfield networks (classical + modern).
// This file adds Hindsight Experience Replay (HER) -- the "Nexus-8
// implanted memory" architecture from the aSoA design session:
//
// The agent almost never receives external reward (levels_completed is
// vanishingly rare in ARC-AGI-3). HER turns every FAILED episode into a
// synthetic success by relabeling: "I didn't solve the puzzle, but I DID
// reach state S_f -- so now I know HOW to reach S_f." Over many episodes
// the agent builds a repertoire of Skills: (startState, actions) -> targetState.
//
// This solves half of the Schollé Wall: goal-conditioned competence from
// sparse reward. The other half (which goal to pursue) is handled by the
// Goal Proposer (compression / MDL / macro system). Together they close
// the loop: "name any target state -> I can get there."
//
// Reference: Andrychowicz et al., "Hindsight Experience Replay" (NeurIPS 2017).
package memory

import (
	"math"
	"sort"
	"sync"
)

// Transition records one step of agent-environment interaction:
// the agent observed State, took Action, and the world moved to NextState.
type Transition struct {
	State     []float64
	Action    string
	NextState []float64
}

// Episode is a complete trajectory from one game/level: the ordered
// sequence of transitions the agent traversed before the game ended
// (win, loss, or action-budget exhaustion).
type Episode struct {
	GameID      string
	Transitions []Transition
	Won         bool
}

// FinalState returns the last state reached, or nil if empty.
func (ep *Episode) FinalState() []float64 {
	if len(ep.Transitions) == 0 {
		return nil
	}
	return ep.Transitions[len(ep.Transitions)-1].NextState
}

// ActionSequence returns the ordered list of actions taken.
func (ep *Episode) ActionSequence() []string {
	out := make([]string, len(ep.Transitions))
	for i, t := range ep.Transitions {
		out[i] = t.Action
	}
	return out
}

// Skill is a learned "I can reach TargetState from StartState by executing
// Actions" entry, induced by HER relabeling. The agent failed the original
// goal but DID arrive at TargetState -- a synthetic success memory.
//
// Shorter action sequences are preferred by transformation-MDL: a 1-step
// skill (one macro) is a shorter program than a 30-step skill, so the
// planner favors composing a few short skills over replaying one long
// episode verbatim.
type Skill struct {
	TargetState []float64 // the state this skill reaches
	Actions     []string  // the action sequence that gets there
	StartState  []float64 // the state you need to start from
	Uses        int       // how many times this skill was matched/retrieved
	ProgramLen  int       // len(Actions) -- cached for sorting by MDL
}

// HindsightMemory implements Hindsight Experience Replay for Protaxon.
//
// Lifecycle (managed by the caller, typically bridge or cmd):
//
//	h.BeginEpisode("game-id")
//	for each tick:
//	    h.RecordTransition(prevState, action, curState)
//	newSkills := h.EndEpisode(won)
//
// The Engine's RunPredictiveCycle records transitions automatically when
// HER is non-nil. BeginEpisode/EndEpisode are called by the game loop.
type HindsightMemory struct {
	mu          sync.Mutex
	current     *Episode   // episode being recorded right now
	episodes    []*Episode // completed episodes (bounded ring buffer)
	skills      []*Skill   // learned repertoire: "states I can reach"
	maxEpisodes int
	maxSkills   int
	dim         int // state vector dimensionality
}

// NewHindsightMemory creates a fresh HER memory. dim must match
// pipeline.ObservationVectorDim so state vectors are comparable.
func NewHindsightMemory(dim, maxEpisodes, maxSkills int) *HindsightMemory {
	return &HindsightMemory{
		dim:         dim,
		maxEpisodes: maxEpisodes,
		maxSkills:   maxSkills,
	}
}

// BeginEpisode starts recording a new episode. Any in-progress episode
// is silently discarded (e.g. on a mid-episode game reset).
func (h *HindsightMemory) BeginEpisode(gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = &Episode{GameID: gameID}
}

// RecordTransition appends one (state, action, nextState) step to the
// current episode. All vectors are defensively copied. No-op if no
// episode is in progress or action is empty.
func (h *HindsightMemory) RecordTransition(state []float64, action string, nextState []float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current == nil || action == "" {
		return
	}
	h.current.Transitions = append(h.current.Transitions, Transition{
		State:     append([]float64(nil), state...),
		Action:    action,
		NextState: append([]float64(nil), nextState...),
	})
}

// EndEpisode completes the current episode, stores it, and runs hindsight
// relabeling to extract skills. Returns the number of new skills induced.
// No-op (returns 0) if no episode is in progress or it has no transitions.
func (h *HindsightMemory) EndEpisode(won bool) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current == nil || len(h.current.Transitions) == 0 {
		h.current = nil
		return 0
	}
	h.current.Won = won

	// Store the completed episode (ring buffer).
	h.episodes = append(h.episodes, h.current)
	if len(h.episodes) > h.maxEpisodes {
		h.episodes = h.episodes[len(h.episodes)-h.maxEpisodes:]
	}

	// Run HER relabeling.
	newSkills := h.relabel(h.current)
	h.current = nil
	return newSkills
}

// relabel implements HER's combined "final" + "future" strategy.
//
// "Final": the full episode teaches how to reach the final state.
// "Future" (sampled): subsequences teach shorter programs to reach
// intermediate and final states. Shorter programs are preferred by
// transformation-MDL, so these are higher-value skills.
//
// Bounded to O(maxRelabelSamples) new candidates per episode to avoid
// quadratic explosion on long episodes (~60 steps in ARC-AGI).
func (h *HindsightMemory) relabel(ep *Episode) int {
	n := len(ep.Transitions)
	if n == 0 {
		return 0
	}
	added := 0
	finalState := ep.FinalState()

	// Strategy 1 ("final"): full episode -> final state.
	if finalState != nil {
		skill := &Skill{
			TargetState: append([]float64(nil), finalState...),
			Actions:     ep.ActionSequence(),
			StartState:  append([]float64(nil), ep.Transitions[0].State...),
			ProgramLen:  n,
		}
		if h.addSkillIfNovel(skill) {
			added++
		}
	}

	// Strategy 2 ("future", sampled): pick evenly-spaced intermediate
	// points and generate suffix-to-final and prefix-to-intermediate skills.
	step := n / maxRelabelSamples
	if step < 1 {
		step = 1
	}
	for i := step; i < n; i += step {
		// Suffix skill: from state[i], actions[i..N-1] reach finalState.
		suffixActions := make([]string, 0, n-i)
		for j := i; j < n; j++ {
			suffixActions = append(suffixActions, ep.Transitions[j].Action)
		}
		suffSkill := &Skill{
			TargetState: append([]float64(nil), finalState...),
			Actions:     suffixActions,
			StartState:  append([]float64(nil), ep.Transitions[i].State...),
			ProgramLen:  len(suffixActions),
		}
		if h.addSkillIfNovel(suffSkill) {
			added++
		}

		// Prefix skill: from state[0], actions[0..i-1] reach state[i].
		// These intermediate targets diversify the repertoire.
		if i > 1 {
			prefixActions := make([]string, i)
			for j := 0; j < i; j++ {
				prefixActions[j] = ep.Transitions[j].Action
			}
			prefSkill := &Skill{
				TargetState: append([]float64(nil), ep.Transitions[i].State...),
				Actions:     prefixActions,
				StartState:  append([]float64(nil), ep.Transitions[0].State...),
				ProgramLen:  len(prefixActions),
			}
			if h.addSkillIfNovel(prefSkill) {
				added++
			}
		}
	}

	return added
}

// maxRelabelSamples bounds how many intermediate points are sampled per
// episode for the "future" relabeling strategy. 10 covers a 60-step
// episode at 6-step intervals, generating ~20 candidate skills per
// episode -- enough diversity without flooding the repertoire.
const maxRelabelSamples = 10

// noveltyThreshold: a new skill is "novel" only if its target state has
// cosine distance > this from every existing skill's target. Prevents
// the repertoire from filling up with near-duplicates.
const noveltyThreshold = 0.05

// addSkillIfNovel adds a skill only if its target is sufficiently
// different from all existing skills. Enforces maxSkills by evicting
// the least-used skill (LRU-style, favoring skills the planner actually
// relies on). Returns true if the skill was added.
func (h *HindsightMemory) addSkillIfNovel(skill *Skill) bool {
	for _, existing := range h.skills {
		if CosineSimilarity(existing.TargetState, skill.TargetState) > 1.0-noveltyThreshold {
			// Near-duplicate target. But if the new skill has a SHORTER
			// program (lower transformation-MDL), replace it: same
			// destination, better route.
			if skill.ProgramLen < existing.ProgramLen {
				existing.Actions = skill.Actions
				existing.StartState = skill.StartState
				existing.ProgramLen = skill.ProgramLen
			}
			return false
		}
	}

	h.skills = append(h.skills, skill)

	// Evict the least-used skill if we're over budget.
	if len(h.skills) > h.maxSkills {
		minIdx, minUses := 0, h.skills[0].Uses
		for i, s := range h.skills {
			if s.Uses < minUses {
				minIdx, minUses = i, s.Uses
			}
		}
		h.skills = append(h.skills[:minIdx], h.skills[minIdx+1:]...)
	}
	return true
}

// FindClosestSkill searches the repertoire for the skill whose target
// state is most similar to the desired target (by cosine similarity).
// Returns (nil, 0) if the repertoire is empty.
//
// This is goal-conditioned planning via HER: "I want to reach target;
// which of my learned skills gets closest?" The planner can then replay
// that skill's action sequence (or a prefix of it) to move toward the
// target.
func (h *HindsightMemory) FindClosestSkill(target []float64) (*Skill, float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.skills) == 0 || len(target) == 0 {
		return nil, 0
	}

	var best *Skill
	bestSim := math.Inf(-1)
	for _, s := range h.skills {
		sim := CosineSimilarity(s.TargetState, target)
		if sim > bestSim {
			bestSim = sim
			best = s
		}
	}
	if best != nil {
		best.Uses++
	}
	return best, bestSim
}

// FindSkillsForState returns all skills whose StartState is similar to
// currentState (cosine similarity >= minSimilarity), sorted by target
// novelty (most different from currentState first -- the agent should
// prefer skills that take it somewhere NEW, not ones that leave it in
// place).
//
// This is the "what CAN I do from here?" query -- the repertoire of
// reachable states given my current position. Combined with a Goal
// Proposer that scores which targets are worth reaching, this gives
// goal-conditioned action selection.
func (h *HindsightMemory) FindSkillsForState(currentState []float64, minSimilarity float64) []*Skill {
	h.mu.Lock()
	defer h.mu.Unlock()

	type scored struct {
		skill   *Skill
		novelty float64
	}
	var matches []scored
	for _, s := range h.skills {
		if CosineSimilarity(s.StartState, currentState) >= minSimilarity {
			novelty := 1.0 - CosineSimilarity(s.TargetState, currentState)
			matches = append(matches, scored{skill: s, novelty: novelty})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].novelty > matches[j].novelty
	})
	result := make([]*Skill, len(matches))
	for i, m := range matches {
		result[i] = m.skill
	}
	return result
}

// ShortestSkillTo finds the skill that reaches closest to target AND has
// the shortest action sequence (lowest transformation-MDL). This is the
// Occam's-razor planner: among all skills that can reach the neighborhood
// of the target, prefer the one with the shortest program.
//
// minTargetSimilarity gates how close "close enough" is (cosine similarity
// to the target state). Returns (nil, 0) if nothing qualifies.
func (h *HindsightMemory) ShortestSkillTo(target []float64, minTargetSimilarity float64) (*Skill, float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var best *Skill
	bestSim := 0.0

	for _, s := range h.skills {
		sim := CosineSimilarity(s.TargetState, target)
		if sim < minTargetSimilarity {
			continue
		}
		if best == nil || s.ProgramLen < best.ProgramLen || (s.ProgramLen == best.ProgramLen && sim > bestSim) {
			best = s
			bestSim = sim
		}
	}
	if best != nil {
		best.Uses++
	}
	return best, bestSim
}

// SkillCount returns how many skills are in the repertoire.
func (h *HindsightMemory) SkillCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.skills)
}

// EpisodeCount returns how many completed episodes are stored.
func (h *HindsightMemory) EpisodeCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.episodes)
}

// Episodes returns a copy of the completed episodes slice, allowing external
// analysis (e.g., Goal Proposers) to read the transition history without
// breaking encapsulation.
func (h *HindsightMemory) Episodes() []*Episode {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*Episode, len(h.episodes))
	copy(out, h.episodes)
	return out
}

// HasEpisodeInProgress reports whether BeginEpisode was called without
// a matching EndEpisode yet.
func (h *HindsightMemory) HasEpisodeInProgress() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.current != nil
}

// CurrentEpisodeLen returns the number of transitions recorded in the
// current in-progress episode (0 if none).
func (h *HindsightMemory) CurrentEpisodeLen() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.current == nil {
		return 0
	}
	return len(h.current.Transitions)
}

// CosineSimilarity returns the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude or mismatched lengths.
// Exported because LearnedPreference and DangerProximity in the Engine
// compute the same thing inline -- this centralizes it.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	dot, magA, magB := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}
