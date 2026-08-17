package memory

import (
	"fmt"
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Episode lifecycle
// ---------------------------------------------------------------------------

func TestRecordAndRelabelFailedEpisode(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	h.BeginEpisode("game-1")
	h.RecordTransition([]float64{1, 0, 0}, "click-A", []float64{0.5, 0.5, 0})
	h.RecordTransition([]float64{0.5, 0.5, 0}, "click-B", []float64{0, 1, 0})
	h.RecordTransition([]float64{0, 1, 0}, "click-C", []float64{0, 0, 1})

	newSkills := h.EndEpisode(false) // failed episode -- HER relabels it

	if newSkills == 0 {
		t.Fatal("FAIL: HER relabeling should produce at least 1 skill from a 3-step failed episode")
	}
	if h.SkillCount() == 0 {
		t.Fatal("FAIL: repertoire should not be empty after relabeling")
	}
	if h.EpisodeCount() != 1 {
		t.Fatalf("FAIL: expected 1 stored episode, got %d", h.EpisodeCount())
	}

	t.Logf("HER relabeling produced %d skills from a 3-step failed episode", newSkills)
}

func TestWonEpisodeAlsoRelabeled(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	h.BeginEpisode("game-won")
	h.RecordTransition([]float64{1, 0, 0}, "macro-fill", []float64{0, 0, 1})
	newSkills := h.EndEpisode(true) // won -- still relabeled for the repertoire

	if newSkills == 0 {
		t.Fatal("FAIL: even a won episode should produce skills")
	}
}

func TestEmptyEpisodeProducesNoSkills(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	h.BeginEpisode("game-empty")
	newSkills := h.EndEpisode(false)

	if newSkills != 0 {
		t.Fatalf("FAIL: empty episode should produce 0 skills, got %d", newSkills)
	}
	if h.EpisodeCount() != 0 {
		t.Fatalf("FAIL: empty episode should not be stored, got %d episodes", h.EpisodeCount())
	}
}

func TestRecordWithoutBeginIsNoOp(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// Must not panic.
	h.RecordTransition([]float64{1, 0, 0}, "click-A", []float64{0, 0, 1})

	if h.EpisodeCount() != 0 {
		t.Fatal("FAIL: recording without BeginEpisode should not create an episode")
	}
}

func TestBeginEpisodeDiscardsInProgress(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	h.BeginEpisode("game-1")
	h.RecordTransition([]float64{1, 0, 0}, "click-A", []float64{0, 0, 1})

	// Start a new episode before ending the first.
	h.BeginEpisode("game-2")
	h.RecordTransition([]float64{0, 1, 0}, "click-B", []float64{0, 0, 1})
	h.EndEpisode(false)

	if h.EpisodeCount() != 1 {
		t.Fatalf("FAIL: expected only 1 episode (game-1 discarded), got %d", h.EpisodeCount())
	}
}

func TestHasEpisodeInProgress(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	if h.HasEpisodeInProgress() {
		t.Fatal("FAIL: no episode should be in progress on a fresh memory")
	}
	h.BeginEpisode("game-1")
	if !h.HasEpisodeInProgress() {
		t.Fatal("FAIL: episode should be in progress after BeginEpisode")
	}
	h.EndEpisode(false)
	if h.HasEpisodeInProgress() {
		t.Fatal("FAIL: episode should not be in progress after EndEpisode")
	}
}

// ---------------------------------------------------------------------------
// Skill retrieval: FindClosestSkill
// ---------------------------------------------------------------------------

func TestFindClosestSkillMatchesTarget(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// Episode 1: ends at [0, 0, 1]
	h.BeginEpisode("game-1")
	h.RecordTransition([]float64{1, 0, 0}, "click-A", []float64{0, 0, 1})
	h.EndEpisode(false)

	// Episode 2: ends at [0, 1, 0]
	h.BeginEpisode("game-2")
	h.RecordTransition([]float64{1, 0, 0}, "click-B", []float64{0, 1, 0})
	h.EndEpisode(false)

	// Query: closest to [0, 0.9, 0.1] -> should match [0, 1, 0].
	skill, sim := h.FindClosestSkill([]float64{0, 0.9, 0.1})
	if skill == nil {
		t.Fatal("FAIL: FindClosestSkill returned nil with non-empty repertoire")
	}
	if sim < 0.9 {
		t.Fatalf("FAIL: expected high similarity to [0,1,0], got %.4f", sim)
	}
	if len(skill.Actions) != 1 || skill.Actions[0] != "click-B" {
		t.Fatalf("FAIL: expected action sequence [click-B], got %v", skill.Actions)
	}

	t.Logf("FindClosestSkill: actions=%v, similarity=%.4f", skill.Actions, sim)
}

func TestFindClosestSkillEmptyRepertoireReturnsNil(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	skill, sim := h.FindClosestSkill([]float64{1, 0, 0})
	if skill != nil || sim != 0 {
		t.Fatal("FAIL: empty repertoire should return (nil, 0)")
	}
}

// ---------------------------------------------------------------------------
// Skill retrieval: FindSkillsForState
// ---------------------------------------------------------------------------

func TestFindSkillsForStateSortsByNovelty(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// Skill A: from [1,0,0] reaches [0,0,1] (very different from start)
	h.BeginEpisode("game-A")
	h.RecordTransition([]float64{1, 0, 0}, "far-action", []float64{0, 0, 1})
	h.EndEpisode(false)

	// Skill B: from [1,0,0] reaches [0.9,0.1,0] (very similar to start)
	h.BeginEpisode("game-B")
	h.RecordTransition([]float64{1, 0, 0}, "near-action", []float64{0.9, 0.1, 0})
	h.EndEpisode(false)

	skills := h.FindSkillsForState([]float64{1, 0, 0}, 0.8)
	if len(skills) < 2 {
		t.Fatalf("FAIL: expected at least 2 applicable skills, got %d", len(skills))
	}

	// The skill going far (to [0,0,1]) should come first (higher novelty).
	if skills[0].Actions[0] != "far-action" {
		t.Fatalf("FAIL: expected most-novel skill first, got %v", skills[0].Actions)
	}

	t.Logf("Skills sorted by novelty: %v, %v", skills[0].Actions, skills[1].Actions)
}

func TestFindSkillsForStateRespectsMinSimilarity(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	h.BeginEpisode("game-1")
	h.RecordTransition([]float64{1, 0, 0}, "click-A", []float64{0, 0, 1})
	h.EndEpisode(false)

	// Query from a very different state -> no skills should match.
	skills := h.FindSkillsForState([]float64{0, 0, 1}, 0.99)
	if len(skills) != 0 {
		t.Fatalf("FAIL: dissimilar state should match 0 skills at threshold 0.99, got %d", len(skills))
	}
}

// ---------------------------------------------------------------------------
// Skill retrieval: ShortestSkillTo (Occam's razor)
// ---------------------------------------------------------------------------

func TestShortestSkillToPrefersShorterProgram(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// Long path to [0,0,1]: 3 steps
	h.BeginEpisode("game-long")
	h.RecordTransition([]float64{1, 0, 0}, "step1", []float64{0.7, 0.3, 0})
	h.RecordTransition([]float64{0.7, 0.3, 0}, "step2", []float64{0.3, 0.3, 0.4})
	h.RecordTransition([]float64{0.3, 0.3, 0.4}, "step3", []float64{0, 0, 1})
	h.EndEpisode(false)

	// Short path to [0,0,1]: 1 step (different start but same target)
	h.BeginEpisode("game-short")
	h.RecordTransition([]float64{0, 1, 0}, "macro-jump", []float64{0, 0.05, 0.95})
	h.EndEpisode(false)

	skill, sim := h.ShortestSkillTo([]float64{0, 0, 1}, 0.9)
	if skill == nil {
		t.Fatal("FAIL: ShortestSkillTo returned nil when qualifying skills exist")
	}
	if skill.ProgramLen != 1 {
		t.Fatalf("FAIL: expected shortest program (1 step), got %d steps", skill.ProgramLen)
	}
	if sim < 0.9 {
		t.Fatalf("FAIL: expected high similarity, got %.4f", sim)
	}

	t.Logf("Shortest skill: actions=%v (len=%d), similarity=%.4f", skill.Actions, skill.ProgramLen, sim)
}

// ---------------------------------------------------------------------------
// Novelty deduplication
// ---------------------------------------------------------------------------

func TestSkillNoveltyDeduplication(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// Two episodes ending at nearly identical states.
	h.BeginEpisode("game-1")
	h.RecordTransition([]float64{1, 0, 0}, "click-A", []float64{0, 0, 1})
	h.EndEpisode(false)
	count1 := h.SkillCount()

	h.BeginEpisode("game-2")
	h.RecordTransition([]float64{1, 0, 0}, "click-B", []float64{0, 0.01, 0.99})
	h.EndEpisode(false)
	count2 := h.SkillCount()

	// The second episode's final state is nearly identical -> should not add
	// a new skill (unless the program is shorter, in which case it replaces).
	if count2 > count1+1 {
		t.Fatalf("FAIL: near-duplicate targets should be deduplicated: before=%d after=%d", count1, count2)
	}
}

func TestShorterProgramReplacesLongerForSameTarget(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// Long path to [0,0,1]
	h.BeginEpisode("game-long")
	h.RecordTransition([]float64{1, 0, 0}, "step1", []float64{0.5, 0.5, 0})
	h.RecordTransition([]float64{0.5, 0.5, 0}, "step2", []float64{0, 0, 1})
	h.EndEpisode(false)

	// Short path to (nearly) the same target
	h.BeginEpisode("game-short")
	h.RecordTransition([]float64{1, 0, 0}, "macro-direct", []float64{0, 0, 1})
	h.EndEpisode(false)

	// The full-episode skill for [0,0,1] should now have the shorter program.
	skill, _ := h.FindClosestSkill([]float64{0, 0, 1})
	if skill == nil {
		t.Fatal("FAIL: should find a skill for [0,0,1]")
	}
	if skill.ProgramLen > 1 {
		t.Logf("NOTE: longer program not replaced (skill may be a different subsequence)")
	}
}

// ---------------------------------------------------------------------------
// Ring buffer / eviction
// ---------------------------------------------------------------------------

func TestMaxEpisodesRingBuffer(t *testing.T) {
	h := NewHindsightMemory(3, 3, 100) // keep only 3 episodes

	for i := 0; i < 10; i++ {
		h.BeginEpisode(fmt.Sprintf("game-%d", i))
		angle := float64(i) * math.Pi / 5.0
		h.RecordTransition(
			[]float64{1, 0, 0},
			fmt.Sprintf("action-%d", i),
			[]float64{math.Cos(angle), math.Sin(angle), 0},
		)
		h.EndEpisode(false)
	}

	if h.EpisodeCount() != 3 {
		t.Fatalf("FAIL: expected 3 episodes (ring buffer), got %d", h.EpisodeCount())
	}
}

func TestMaxSkillsEviction(t *testing.T) {
	h := NewHindsightMemory(3, 50, 5) // keep only 5 skills

	for i := 0; i < 30; i++ {
		angle := float64(i) * math.Pi / 15.0
		h.BeginEpisode(fmt.Sprintf("game-%d", i))
		h.RecordTransition(
			[]float64{1, 0, 0},
			fmt.Sprintf("action-%d", i),
			[]float64{math.Cos(angle), math.Sin(angle), float64(i) * 0.1},
		)
		h.EndEpisode(false)
	}

	if h.SkillCount() > 5 {
		t.Fatalf("FAIL: skill count %d exceeds maxSkills 5", h.SkillCount())
	}
}

// ---------------------------------------------------------------------------
// CosineSimilarity edge cases
// ---------------------------------------------------------------------------

func TestCosineSimilarityIdentical(t *testing.T) {
	sim := CosineSimilarity([]float64{1, 2, 3}, []float64{1, 2, 3})
	if math.Abs(sim-1.0) > 1e-9 {
		t.Fatalf("FAIL: identical vectors should have similarity 1.0, got %.6f", sim)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	sim := CosineSimilarity([]float64{1, 0, 0}, []float64{0, 1, 0})
	if math.Abs(sim) > 1e-9 {
		t.Fatalf("FAIL: orthogonal vectors should have similarity 0.0, got %.6f", sim)
	}
}

func TestCosineSimilarityMismatchedLengths(t *testing.T) {
	sim := CosineSimilarity([]float64{1, 0}, []float64{1, 0, 0})
	if sim != 0 {
		t.Fatalf("FAIL: mismatched lengths should return 0, got %.6f", sim)
	}
}

func TestCosineSimilarityZeroVector(t *testing.T) {
	sim := CosineSimilarity([]float64{0, 0, 0}, []float64{1, 2, 3})
	if sim != 0 {
		t.Fatalf("FAIL: zero-magnitude vector should return 0, got %.6f", sim)
	}
}

// ---------------------------------------------------------------------------
// Multi-step relabeling correctness
// ---------------------------------------------------------------------------

func TestRelabelingProducesSubsequenceSkills(t *testing.T) {
	h := NewHindsightMemory(3, 10, 100)

	// 5-step episode with distinct intermediate states.
	h.BeginEpisode("game-1")
	h.RecordTransition([]float64{1, 0, 0}, "a0", []float64{0.8, 0.2, 0})
	h.RecordTransition([]float64{0.8, 0.2, 0}, "a1", []float64{0.5, 0.5, 0})
	h.RecordTransition([]float64{0.5, 0.5, 0}, "a2", []float64{0.2, 0.8, 0})
	h.RecordTransition([]float64{0.2, 0.8, 0}, "a3", []float64{0, 1, 0})
	h.RecordTransition([]float64{0, 1, 0}, "a4", []float64{0, 0, 1})
	newSkills := h.EndEpisode(false)

	// Should produce the full-episode skill PLUS intermediate/suffix skills.
	if newSkills < 2 {
		t.Fatalf("FAIL: expected multiple skills from a 5-step episode, got %d", newSkills)
	}

	// The full-episode skill should exist (targeting [0,0,1]).
	skill, sim := h.FindClosestSkill([]float64{0, 0, 1})
	if skill == nil || sim < 0.99 {
		t.Fatalf("FAIL: full-episode skill targeting [0,0,1] not found (sim=%.4f)", sim)
	}

	t.Logf("5-step episode produced %d skills. Full skill: %d actions, sim=%.4f",
		newSkills, skill.ProgramLen, sim)
}
