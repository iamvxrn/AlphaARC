package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
)

// ============================================================
// Step 0: SystemMode — finite state machine, hard ONLINE/OFFLINE lock
// ============================================================

type SystemMode int

const (
	Online SystemMode = iota
	Offline
)

func (m SystemMode) String() string {
	if m == Online {
		return "ONLINE"
	}
	return "OFFLINE"
}

type System struct {
	Mode SystemMode
	Tick int
}

// onlineGuard panics (does not silently skip) if called outside ONLINE mode.
func (s *System) onlineGuard(fn string) {
	if s.Mode != Online {
		panic(fmt.Sprintf("MODE VIOLATION: %s requires ONLINE, current mode=%s", fn, s.Mode))
	}
}

// offlineGuard panics if called outside OFFLINE mode.
func (s *System) offlineGuard(fn string) {
	if s.Mode != Offline {
		panic(fmt.Sprintf("MODE VIOLATION: %s requires OFFLINE, current mode=%s", fn, s.Mode))
	}
}

// ---- ONLINE-only operation (cheap, every tick) ----
func (s *System) OnlineNoop(fn string) {
	s.onlineGuard(fn)
}

// ---- OFFLINE-only operations (expensive, rare, stubbed for MVP) ----
func (s *System) CommunityDetection() {
	s.offlineGuard("CommunityDetection")
	fmt.Println("  [OFFLINE] would run ashby_reorg here (community detection, no-op in MVP)")
}

func (s *System) GraphRestructure() {
	s.offlineGuard("GraphRestructure")
	fmt.Println("  [OFFLINE] would run graph restructure here (add/remove nodes, no-op in MVP)")
}

func step0() {
	fmt.Println("=== Step 0: SystemMode lock ===")
	sys := &System{Mode: Online}

	fmt.Printf("Tick %d: mode=%s -> attempting OFFLINE op CommunityDetection() while ONLINE\n", sys.Tick, sys.Mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  BLOCKED (panic recovered): %v\n", r)
			}
		}()
		sys.CommunityDetection()
	}()

	sys.Tick++
	fmt.Printf("Tick %d: mode=%s -> attempting ONLINE op OnlineNoop() while ONLINE\n", sys.Tick, sys.Mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  BLOCKED (panic recovered): %v\n", r)
			}
		}()
		sys.OnlineNoop("HebbianUpdate-test")
		fmt.Println("  ALLOWED: OnlineNoop executed successfully")
	}()

	sys.Tick++
	fmt.Printf("Tick %d: switching mode ONLINE -> OFFLINE\n", sys.Tick)
	sys.Mode = Offline

	fmt.Printf("Tick %d: mode=%s -> attempting OFFLINE op CommunityDetection() while OFFLINE\n", sys.Tick, sys.Mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  BLOCKED (panic recovered): %v\n", r)
			}
		}()
		sys.CommunityDetection()
		fmt.Println("  ALLOWED: CommunityDetection executed successfully")
	}()

	fmt.Printf("Tick %d: mode=%s -> attempting ONLINE op OnlineNoop() while OFFLINE\n", sys.Tick, sys.Mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  BLOCKED (panic recovered): %v\n", r)
			}
		}()
		sys.OnlineNoop("HebbianUpdate-test")
		fmt.Println("  ALLOWED: OnlineNoop executed successfully")
	}()

	sys.Mode = Online
	fmt.Printf("Tick %d: switching mode OFFLINE -> ONLINE\n", sys.Tick)
	fmt.Println()
}

// ============================================================
// Helpers
// ============================================================

func softmax(x []float64) []float64 {
	out := make([]float64, len(x))
	maxV := math.Inf(-1)
	for _, v := range x {
		if v > maxV {
			maxV = v
		}
	}
	sum := 0.0
	for i, v := range x {
		out[i] = math.Exp(v - maxV)
		sum += out[i]
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

func sumAbs(row []float64) float64 {
	s := 0.0
	for _, v := range row {
		s += math.Abs(v)
	}
	return s
}

func normalizeRowL1(row []float64, cap float64) {
	s := sumAbs(row)
	if s > cap {
		scale := cap / s
		for i := range row {
			row[i] *= scale
		}
	}
}

// ============================================================
// Step 1: Predictor (ONLINE) — grid world 5x5, Hebbian prediction
// ============================================================

const gridN = 5
const nStates = gridN * gridN

func idx(x, y int) int { return y*gridN + x }

// stochasticNextStates returns the (up to) two possible next positions for a
// given position under a fixed random policy: each state has a 50/50 chance
// of moving "right" (wrap) or "down" (wrap). This is intentionally
// non-deterministic so Hebbian learning has something nontrivial to predict.
func stochasticNextStates(pos int) (a, b int) {
	x := pos % gridN
	y := pos / gridN
	right := idx((x+1)%gridN, y)
	down := idx(x, (y+1)%gridN)
	return right, down
}

type Predictor struct {
	W  [][]float64 // W[pos][nextPos]
	LR float64
}

func NewPredictor(n int, lr float64) *Predictor {
	w := make([][]float64, n)
	for i := range w {
		w[i] = make([]float64, n)
	}
	return &Predictor{W: w, LR: lr}
}

// ForwardPass: ONLINE, produces predicted-next-state distribution.
func (p *Predictor) ForwardPass(sys *System, pos int) []float64 {
	sys.onlineGuard("Predictor.ForwardPass")
	return softmax(p.W[pos])
}

// HebbianUpdate: ONLINE, w[i][j] += lr*pre[i]*post[j], O(1) per active connection.
func (p *Predictor) HebbianUpdate(sys *System, pos, nextPos int) {
	sys.onlineGuard("Predictor.HebbianUpdate")
	// pre is one-hot at pos, post is one-hot at nextPos => only w[pos][nextPos] moves.
	p.W[pos][nextPos] += p.LR * 1.0 * 1.0
	normalizeRowL1(p.W[pos], 10.0)
}

func step1() {
	fmt.Println("=== Step 1: Predictor training (MODE=ONLINE) ===")
	sys := &System{Mode: Online}
	rng := rand.New(rand.NewSource(1))
	pred := NewPredictor(nStates, 0.3)

	pos := 0
	iterations := 1000
	errAtCheckpoint := []float64{}
	checkpoints := []int{}

	// track error via moving average over last 50 samples for a smoother, honest signal
	window := []float64{}
	windowSize := 50

	for it := 0; it < iterations; it++ {
		a, b := stochasticNextStates(pos)
		var next int
		if rng.Float64() < 0.5 {
			next = a
		} else {
			next = b
		}

		probs := pred.ForwardPass(sys, pos)
		predErr := 1.0 - probs[next]

		window = append(window, predErr)
		if len(window) > windowSize {
			window = window[1:]
		}

		pred.HebbianUpdate(sys, pos, next)

		if it%100 == 0 || it == iterations-1 {
			avg := 0.0
			for _, v := range window {
				avg += v
			}
			avg /= float64(len(window))
			fmt.Printf("Iteration %4d: prediction_error(instant)=%.4f  prediction_error(avg-last-%d)=%.4f\n", it, predErr, len(window), avg)
			errAtCheckpoint = append(errAtCheckpoint, avg)
			checkpoints = append(checkpoints, it)
		}

		pos = next
	}

	monotonic := true
	for i := 1; i < len(errAtCheckpoint); i++ {
		if errAtCheckpoint[i] > errAtCheckpoint[i-1]+1e-9 {
			monotonic = false
			break
		}
	}
	firstErr := errAtCheckpoint[0]
	lastErr := errAtCheckpoint[len(errAtCheckpoint)-1]
	fmt.Printf("RESULT: error decreased monotonically across checkpoints? %v\n", boolYesNo(monotonic))
	fmt.Printf("RESULT: first checkpoint avg-error=%.4f, last checkpoint avg-error=%.4f, delta=%.4f\n", firstErr, lastErr, firstErr-lastErr)
	fmt.Printf("NOTE: transitions are stochastic (50/50 per state), so theoretical best-case error floor is ~0.5, not 0.0\n")
	fmt.Println()
}

func boolYesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

// ============================================================
// Step 2: Actor with homeostatic drive (ONLINE)
// ============================================================

type Actor struct {
	W  [][]float64 // W[pos][action] preference weights, 4 actions: up,down,left,right
	E  [][]float64 // eligibility trace, same shape
	LR float64
	Decay float64
}

const (
	actUp = iota
	actDown
	actLeft
	actRight
	numActions
)

func NewActor(n int, lr, decay float64) *Actor {
	w := make([][]float64, n)
	e := make([][]float64, n)
	for i := range w {
		w[i] = make([]float64, numActions)
		e[i] = make([]float64, numActions)
	}
	return &Actor{W: w, E: e, LR: lr, Decay: decay}
}

func (ac *Actor) ForwardPass(sys *System, pos int) []float64 {
	sys.onlineGuard("Actor.ForwardPass")
	return softmax(ac.W[pos])
}

// EligibilityUpdate: ONLINE, decays all traces then bumps the taken (pos,action).
func (ac *Actor) EligibilityUpdate(sys *System, pos, action int) {
	sys.onlineGuard("Actor.EligibilityUpdate")
	for i := range ac.E {
		for j := range ac.E[i] {
			ac.E[i][j] *= ac.Decay
		}
	}
	ac.E[pos][action] += 1.0
}

// HebbianDriveUpdate: ONLINE. drive is the homeostatic error (distance to goal,
// setpoint=0). reward = reduction in drive. w[i][j] += lr * reward * e[i][j].
func (ac *Actor) HebbianDriveUpdate(sys *System, reward float64) {
	sys.onlineGuard("Actor.HebbianDriveUpdate")
	for i := range ac.W {
		for j := range ac.W[i] {
			if ac.E[i][j] != 0 {
				ac.W[i][j] += ac.LR * reward * ac.E[i][j]
			}
		}
		normalizeRowL1(ac.W[i], 10.0)
	}
}

func applyAction(x, y, action int) (int, int) {
	switch action {
	case actUp:
		if y > 0 {
			y--
		}
	case actDown:
		if y < gridN-1 {
			y++
		}
	case actLeft:
		if x > 0 {
			x--
		}
	case actRight:
		if x < gridN-1 {
			x++
		}
	}
	return x, y
}

func manhattan(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func step2() {
	fmt.Println("=== Step 2: Actor homeostatic drive (MODE=ONLINE) ===")
	sys := &System{Mode: Online}
	rng := rand.New(rand.NewSource(7))
	actor := NewActor(nStates, 0.5, 0.7)

	goalX, goalY := 4, 4
	episodesReachedZero := 0
	numEpisodes := 3
	stepsPerEpisode := 20

	for ep := 0; ep < numEpisodes; ep++ {
		x, y := rng.Intn(gridN), rng.Intn(gridN)
		drive := float64(manhattan(x, y, goalX, goalY))
		fmt.Printf("--- Episode %d (start=(%d,%d) goal=(%d,%d)) ---\n", ep, x, y, goalX, goalY)
		reachedZero := false
		for st := 0; st < stepsPerEpisode; st++ {
			pos := idx(x, y)
			probs := actor.ForwardPass(sys, pos)
			action := sampleFromProbs(rng, probs)
			actor.EligibilityUpdate(sys, pos, action)

			nx, ny := applyAction(x, y, action)
			newDrive := float64(manhattan(nx, ny, goalX, goalY))
			reward := drive - newDrive // positive if closer to setpoint 0
			actor.HebbianDriveUpdate(sys, reward)

			x, y = nx, ny
			drive = newDrive
			fmt.Printf("Episode %d Step %2d: pos=(%d,%d) drive=%.1f reward=%.1f\n", ep, st, x, y, drive, reward)
			if drive == 0 {
				reachedZero = true
			}
		}
		if reachedZero {
			episodesReachedZero++
		}
	}
	fmt.Printf("RESULT: episodes where drive reached 0: %d / %d\n", episodesReachedZero, numEpisodes)
	fmt.Println()
}

func sampleFromProbs(rng *rand.Rand, probs []float64) int {
	r := rng.Float64()
	cum := 0.0
	for i, p := range probs {
		cum += p
		if r <= cum {
			return i
		}
	}
	return len(probs) - 1
}

// ============================================================
// Step 3: Weight normalization + threshold gate (ONLINE)
// ============================================================

type Edge struct {
	From, To       int
	Weight         float64
	IsCrossCluster bool
}

const (
	LocalThreshold        = 0.2
	CrossClusterThreshold = 0.5
)

func propagate(edge Edge, activation float64) float64 {
	threshold := LocalThreshold
	decay := 1.0
	if edge.IsCrossCluster {
		threshold = CrossClusterThreshold
		decay = 0.5
	}
	if activation < threshold {
		return 0.0
	}
	return activation * edge.Weight * decay
}

func step3() {
	fmt.Println("=== Step 3: Weight normalization + threshold gate (MODE=ONLINE) ===")
	sys := &System{Mode: Online}
	_ = sys

	// -- Part A: normalization growth control --
	fmt.Println("-- Part A: sum(|weights|) before/after normalization at checkpoints --")
	row := make([]float64, 10)
	rng := rand.New(rand.NewSource(3))
	checkpoints := []int{10, 50, 100}
	nextCheckpoint := 0
	for step := 1; step <= 100; step++ {
		i := rng.Intn(len(row))
		row[i] += 0.3 // raw additive Hebbian growth, no normalization yet
		if nextCheckpoint < len(checkpoints) && step == checkpoints[nextCheckpoint] {
			before := sumAbs(row)
			normalizeRowL1(row, 10.0)
			after := sumAbs(row)
			fmt.Printf("updates=%3d: sum|w| before-normalize=%.4f  after-normalize(cap=10.0)=%.4f\n", step, before, after)
			nextCheckpoint++
		}
	}

	// -- Part B: threshold gate --
	fmt.Println("-- Part B: threshold gate on propagate() --")
	localEdge := Edge{From: 0, To: 1, Weight: 0.8, IsCrossCluster: false}
	crossEdge := Edge{From: 0, To: 5, Weight: 0.8, IsCrossCluster: true}

	testActivations := []float64{0.05, 0.15, 0.2, 0.3, 0.5, 0.6, 0.9}
	for _, act := range testActivations {
		outLocal := propagate(localEdge, act)
		outCross := propagate(crossEdge, act)
		fmt.Printf("activation=%.2f | local-edge(threshold=%.2f): before=%.2f after=%.4f gated=%v | cross-edge(threshold=%.2f): before=%.2f after=%.4f gated=%v\n",
			act, LocalThreshold, act, outLocal, outLocal == 0.0,
			CrossClusterThreshold, act, outCross, outCross == 0.0)
	}
	fmt.Println()
}

// ============================================================
// Step 4: Associator, pattern completion (ONLINE) — Hopfield-style
// ============================================================

type Associator struct {
	W [][]float64
	N int
}

func NewAssociator(n int) *Associator {
	w := make([][]float64, n)
	for i := range w {
		w[i] = make([]float64, n)
	}
	return &Associator{W: w, N: n}
}

// StorePattern: ONLINE Hebbian outer-product learning, normalized by N.
func (as *Associator) StorePattern(sys *System, pattern []float64) {
	sys.onlineGuard("Associator.StorePattern")
	for i := 0; i < as.N; i++ {
		for j := 0; j < as.N; j++ {
			if i == j {
				continue
			}
			as.W[i][j] += (pattern[i] * pattern[j]) / float64(as.N)
		}
	}
}

// Recall: ONLINE synchronous update, iterate until stable or maxIter.
func (as *Associator) Recall(sys *System, input []float64, maxIter int) []float64 {
	sys.onlineGuard("Associator.Recall")
	state := append([]float64{}, input...)
	for it := 0; it < maxIter; it++ {
		next := make([]float64, as.N)
		changed := false
		for i := 0; i < as.N; i++ {
			sum := 0.0
			for j := 0; j < as.N; j++ {
				sum += as.W[i][j] * state[j]
			}
			if sum >= 0 {
				next[i] = 1.0
			} else {
				next[i] = -1.0
			}
			if next[i] != state[i] {
				changed = true
			}
		}
		state = next
		if !changed {
			break
		}
	}
	return state
}

func hamming(a, b []float64) int {
	d := 0
	for i := range a {
		if a[i] != b[i] {
			d++
		}
	}
	return d
}

func randomPattern(rng *rand.Rand, n int) []float64 {
	p := make([]float64, n)
	for i := range p {
		if rng.Float64() < 0.5 {
			p[i] = 1.0
		} else {
			p[i] = -1.0
		}
	}
	return p
}

func corrupt(rng *rand.Rand, pattern []float64, flips int) []float64 {
	out := append([]float64{}, pattern...)
	idxs := rng.Perm(len(out))[:flips]
	for _, i := range idxs {
		out[i] = -out[i]
	}
	return out
}

func step4() {
	fmt.Println("=== Step 4: Associator pattern completion (MODE=ONLINE) ===")
	sys := &System{Mode: Online}
	rng := rand.New(rand.NewSource(11))

	n := nStates // 25
	as := NewAssociator(n)

	numStored := 3
	patterns := make([][]float64, numStored)
	for i := range patterns {
		patterns[i] = randomPattern(rng, n)
		as.StorePattern(sys, patterns[i])
	}
	fmt.Printf("Stored %d random bipolar patterns of length %d (Hopfield capacity ~0.15*N=%.1f)\n", numStored, n, 0.15*float64(n))

	target := patterns[0]
	corruptionLevels := []int{2, 4, 6, 8, 10} // out of 25 bits flipped
	correct := 0
	for i, flips := range corruptionLevels {
		noisy := corrupt(rng, target, flips)
		recovered := as.Recall(sys, noisy, 20)
		dist := hamming(recovered, target)
		ok := dist == 0
		if ok {
			correct++
		}
		fmt.Printf("Test %d: bits_flipped=%2d  hamming(noisy,target)=%2d  hamming(recovered,target)=%2d  exact_recovery=%v\n",
			i+1, flips, hamming(noisy, target), dist, ok)
	}
	fmt.Printf("RESULT: exact recoveries: %d / %d test cases\n", correct, len(corruptionLevels))
	fmt.Println()
}

// ============================================================
// Step 5: Lateral inhibition (ONLINE)
// ============================================================

func step5() {
	fmt.Println("=== Step 5: Lateral inhibition (MODE=ONLINE) ===")
	sys := &System{Mode: Online}
	_ = sys

	// Two conflicting candidate agents competing over the same output slot.
	wA, wB := 0.6, 0.55
	inputs := []float64{0.9, 0.7, 0.85, 0.6, 0.95}
	penalty := 0.15
	boost := 0.05

	fmt.Printf("Initial weights: candidate A=%.3f  candidate B=%.3f\n", wA, wB)
	for round, in := range inputs {
		actA := wA * in
		actB := wB * in
		beforeA, beforeB := wA, wB
		var winner string
		if actA >= actB {
			wA += boost
			wB -= penalty
			if wB < 0 {
				wB = 0
			}
			winner = "A"
		} else {
			wB += boost
			wA -= penalty
			if wA < 0 {
				wA = 0
			}
			winner = "B"
		}
		fmt.Printf("Round %d: input=%.2f  actA=%.4f actB=%.4f  winner=%s | wA: %.4f->%.4f  wB: %.4f->%.4f\n",
			round, in, actA, actB, winner, beforeA, wA, beforeB, wB)
	}
	fmt.Println()
}

// ============================================================
// Step 6: OFFLINE stub — trigger + real call across the boundary
// ============================================================

func step6() {
	fmt.Println("=== Step 6: ONLINE -> OFFLINE transition trigger (stub) ===")
	sys := &System{Mode: Online}

	stimulusAtTick := map[int]bool{0: true, 1: true, 2: false, 3: false, 4: false, 5: false, 6: true, 7: false}
	ticksSinceStimulus := 0
	const offlineTriggerN = 3

	for tick := 0; tick <= 7; tick++ {
		sys.Tick = tick
		hasStimulus := stimulusAtTick[tick]
		if hasStimulus {
			ticksSinceStimulus = 0
		} else {
			ticksSinceStimulus++
		}
		fmt.Printf("Tick %d: mode=%s stimulus=%v ticks_since_stimulus=%d\n", tick, sys.Mode, hasStimulus, ticksSinceStimulus)

		if sys.Mode == Online && ticksSinceStimulus >= offlineTriggerN {
			fmt.Printf("  TRIGGER: %d ticks without stimulus >= %d -> switching ONLINE -> OFFLINE\n", ticksSinceStimulus, offlineTriggerN)
			sys.Mode = Offline
			sys.CommunityDetection()
			sys.GraphRestructure()
			fmt.Println("  switching OFFLINE -> ONLINE")
			sys.Mode = Online
			ticksSinceStimulus = 0
		}
	}
	fmt.Println()
}

// ============================================================
// main
// ============================================================

func main() {
	if len(os.Args) > 1 && os.Args[1] == "refine" {
		refine1()
		refine2()
		refine3()
		return
	}
	step0()
	step1()
	step2()
	step3()
	step4()
	step5()
	step6()
}

// ============================================================
// Refinement 1: Predictor on Actor-controlled (goal-directed) trajectories
// vs random walk. Question: does the error floor drop below 0.5 when the
// policy is directed rather than a coin-flip random walk?
// ============================================================

func argmax(x []float64) int {
	best := 0
	for i := 1; i < len(x); i++ {
		if x[i] > x[best] {
			best = i
		}
	}
	return best
}

// trainActor runs many drive-minimizing episodes so the Actor policy becomes
// goal-directed, then returns it. Reuses the exact Actor/reward from Step 2.
func trainActor(sys *System, rng *rand.Rand, goalX, goalY, episodes, stepsPer int) *Actor {
	actor := NewActor(nStates, 0.5, 0.7)
	for ep := 0; ep < episodes; ep++ {
		x, y := rng.Intn(gridN), rng.Intn(gridN)
		drive := float64(manhattan(x, y, goalX, goalY))
		for st := 0; st < stepsPer; st++ {
			pos := idx(x, y)
			probs := actor.ForwardPass(sys, pos)
			action := sampleFromProbs(rng, probs)
			actor.EligibilityUpdate(sys, pos, action)
			nx, ny := applyAction(x, y, action)
			newDrive := float64(manhattan(nx, ny, goalX, goalY))
			reward := drive - newDrive
			actor.HebbianDriveUpdate(sys, reward)
			x, y, drive = nx, ny, newDrive
		}
	}
	return actor
}

// predictorErrorTable trains a fresh Predictor on a stream of (pos,next)
// transitions produced by nextFn, and prints a checkpoint table of the
// moving-average prediction error. Returns the final avg error.
func predictorErrorTable(sys *System, label string, iterations int,
	startPos int, nextFn func(pos int) int) float64 {
	pred := NewPredictor(nStates, 0.3)
	pos := startPos
	window := []float64{}
	windowSize := 50
	var lastAvg float64
	for it := 0; it < iterations; it++ {
		next := nextFn(pos)
		probs := pred.ForwardPass(sys, pos)
		predErr := 1.0 - probs[next]
		window = append(window, predErr)
		if len(window) > windowSize {
			window = window[1:]
		}
		pred.HebbianUpdate(sys, pos, next)
		if it%100 == 0 || it == iterations-1 {
			avg := 0.0
			for _, v := range window {
				avg += v
			}
			avg /= float64(len(window))
			lastAvg = avg
			fmt.Printf("[%s] Iteration %4d: prediction_error(avg-last-%d)=%.4f\n", label, it, len(window), avg)
		}
		pos = next
	}
	return lastAvg
}

func refine1() {
	fmt.Println("=== Refinement 1: Predictor on random-walk vs Actor-controlled data ===")
	sys := &System{Mode: Online}
	goalX, goalY := 4, 4
	iterations := 1000

	// (a) Baseline: random-walk transitions (same as Step 1: 50/50 right/down).
	rngRW := rand.New(rand.NewSource(1))
	fmt.Println("-- (a) RANDOM WALK (coin-flip transitions) --")
	rwFloor := predictorErrorTable(sys, "randwalk", iterations, 0, func(pos int) int {
		a, b := stochasticNextStates(pos)
		if rngRW.Float64() < 0.5 {
			return a
		}
		return b
	})

	// (b) Actor-controlled: train the Actor first, then generate GREEDY
	// (argmax) goal-directed transitions and feed those to a fresh Predictor.
	rngTrain := rand.New(rand.NewSource(7))
	actor := trainActor(sys, rngTrain, goalX, goalY, 400, 40)
	fmt.Println("-- (b) ACTOR-CONTROLLED (greedy argmax policy, goal-directed) --")
	// The controlled trajectory resets to a random start when it reaches the goal
	// and dwells (otherwise it would sit on the goal forever producing trivial
	// self-transitions). Reset uses its own RNG so the stream is reproducible.
	rngReset := rand.New(rand.NewSource(42))
	curX, curY := 0, 0
	ctrlFloor := predictorErrorTable(sys, "actorctrl", iterations, idx(curX, curY), func(pos int) int {
		x, y := pos%gridN, pos/gridN
		if x == goalX && y == goalY {
			// reached goal: teleport to a fresh random start (a real transition
			// the predictor must also learn)
			x, y = rngReset.Intn(gridN), rngReset.Intn(gridN)
			return idx(x, y)
		}
		probs := actor.ForwardPass(sys, pos)
		action := argmax(probs)
		nx, ny := applyAction(x, y, action)
		return idx(nx, ny)
	})

	fmt.Printf("RESULT: random-walk final avg-error   = %.4f\n", rwFloor)
	fmt.Printf("RESULT: actor-controlled final avg-error = %.4f\n", ctrlFloor)
	fmt.Printf("RESULT: did controlled floor drop below random-walk floor? %v\n", boolYesNo(ctrlFloor < rwFloor))
	fmt.Printf("RESULT: did controlled floor drop below 0.5? %v\n", boolYesNo(ctrlFloor < 0.5))
	fmt.Println()
}

// ============================================================
// Refinement 2: Lateral inhibition with a mid-run reversal. Can candidate B
// recover after its weight was pushed near zero, or is the penalty irreversible?
// ============================================================

func refine2() {
	fmt.Println("=== Refinement 2: Lateral inhibition with victory reversal ===")

	wA, wB := 0.6, 0.55
	penalty := 0.15
	boost := 0.05

	// Each candidate has its OWN input channel per round. Rounds 0-2 favour A;
	// from round 3 the input reverses hard to favour B. We watch whether B,
	// after being suppressed, can climb back and start winning.
	inA := []float64{0.90, 0.85, 0.80, 0.10, 0.10, 0.10, 0.10}
	inB := []float64{0.40, 0.35, 0.30, 1.00, 1.00, 1.00, 1.00}

	fmt.Printf("Initial weights: wA=%.3f wB=%.3f | penalty=%.2f boost=%.2f (loser clamped at 0)\n", wA, wB, penalty, boost)
	bWonAfterReversal := false
	for r := 0; r < len(inA); r++ {
		actA := wA * inA[r]
		actB := wB * inB[r]
		beforeA, beforeB := wA, wB
		var winner string
		if actA >= actB {
			wA += boost
			wB -= penalty
			if wB < 0 {
				wB = 0
			}
			winner = "A"
		} else {
			wB += boost
			wA -= penalty
			if wA < 0 {
				wA = 0
			}
			winner = "B"
			if r >= 3 {
				bWonAfterReversal = true
			}
		}
		fmt.Printf("Round %d: inA=%.2f inB=%.2f | actA=%.4f actB=%.4f winner=%s | wA:%.4f->%.4f wB:%.4f->%.4f\n",
			r, inA[r], inB[r], actA, actB, winner, beforeA, wA, beforeB, wB)
	}
	fmt.Printf("RESULT: minimum wB reached during suppression, then final wB=%.4f\n", wB)
	fmt.Printf("RESULT: did B win any round AFTER the reversal (round>=3)? %v\n", boolYesNo(bWonAfterReversal))
	fmt.Printf("NOTE: if wB is clamped exactly to 0, actB=wB*inB=0 forever -> recovery impossible (irreversible)\n")
	fmt.Println()
}

// ============================================================
// Refinement 3: Associator at LOW noise (1-2 flips) across all 3 stored
// patterns, 15 tests. Misses here => implementation bug, not capacity limit.
// ============================================================

func refine3() {
	fmt.Println("=== Refinement 3: Associator low-noise recovery (1-2 flips, all 3 patterns) ===")
	sys := &System{Mode: Online}
	rng := rand.New(rand.NewSource(11))

	n := nStates
	as := NewAssociator(n)
	numStored := 3
	patterns := make([][]float64, numStored)
	for i := range patterns {
		patterns[i] = randomPattern(rng, n)
		as.StorePattern(sys, patterns[i])
	}
	fmt.Printf("Stored %d bipolar patterns of length %d (capacity ~%.1f)\n", numStored, n, 0.15*float64(n))

	correct := 0
	total := 0
	for p := 0; p < numStored; p++ {
		for trial := 0; trial < 5; trial++ {
			flips := 1
			if trial >= 2 {
				flips = 2
			}
			noisy := corrupt(rng, patterns[p], flips)
			recovered := as.Recall(sys, noisy, 20)
			dist := hamming(recovered, patterns[p])
			ok := dist == 0
			if ok {
				correct++
			}
			total++
			fmt.Printf("pattern=%d trial=%d flips=%d hamming(noisy)=%2d hamming(recovered)=%2d exact=%v\n",
				p, trial, flips, hamming(noisy, patterns[p]), dist, ok)
		}
	}
	fmt.Printf("RESULT: exact recoveries at low noise: %d / %d\n", correct, total)
	fmt.Printf("NOTE: any miss here at 1-2 flips indicates implementation issue, not capacity\n")
	fmt.Println()
}
