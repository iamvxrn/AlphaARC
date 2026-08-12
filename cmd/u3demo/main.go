package main

import (
	"protaxon/pkg/core"
	"protaxon/pkg/graph"
	"protaxon/pkg/homeostasis"
	"protaxon/pkg/memory"
	"protaxon/pkg/offline"
	"fmt"
	"math/rand"
)

func main() {
	fmt.Println("============================================================")
	fmt.Println("         U3 PHASE 2: NEUROMORPHIC HOMEOSTATIC ENGINE        ")
	fmt.Println("============================================================")

	runFSMBenchmark()
	runDeadAgentReversalBenchmark()
	runHopfieldCapacityBenchmark()
	runHomeostasisPlasticityBenchmark()
	runSubconsciousSleepBenchmark()
}

func runDeadAgentReversalBenchmark() {
	fmt.Println("\n--- BENCHMARK: Dead-Agent Protection & Reversal Recovery ---")
	wA, wB := 0.6, 0.55
	penalty := 0.15
	boost := 0.05

	inA := []float64{0.90, 0.85, 0.80, 0.10, 0.10, 0.10, 0.10}
	inB := []float64{0.40, 0.35, 0.30, 1.00, 1.00, 1.00, 1.00}

	fmt.Printf("Initial weights: wA=%.3f wB=%.3f (MinWeightFloor=%.3f)\n", wA, wB, graph.MinWeightFloor)

	for r := 0; r < len(inA); r++ {
		actA := wA * inA[r]
		actB := wB * inB[r]
		var winner string
		if actA >= actB {
			winner = "A"
		} else {
			winner = "B"
		}
		graph.UpdateCandidateWeights(&wA, &wB, inA[r], inB[r], boost, penalty)
		fmt.Printf("  Round %d (inA=%.2f inB=%.2f): winner=%s | wA=%.4f wB=%.4f\n",
			r, inA[r], inB[r], winner, wA, wB)
	}

	if wB > graph.MinWeightFloor && wB > wA {
		fmt.Println("  RESULT: SUCCESS — Candidate B recovered from suppression post-reversal!")
	} else {
		fmt.Println("  RESULT: FAILED — Dead-agent lock occurred!")
	}
}

func runFSMBenchmark() {
	fmt.Println("\n--- BENCHMARK 1: FSM Mode Enforcement ---")
	sys := core.NewSystem()

	fmt.Printf("Initial mode: %s. Attempting ONLINE operation...\n", sys.Mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  EXPECTED PANIC: %v\n", r)
			}
		}()
		sys.OfflineGuard("LouvainCommunityDetection")
	}()

	sys.Mode = core.Offline
	fmt.Printf("Switching mode to %s. Attempting OFFLINE operation...\n", sys.Mode)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  UNEXPECTED PANIC: %v\n", r)
			}
		}()
		sys.OfflineGuard("LouvainCommunityDetection")
		fmt.Println("  ALLOWED: OFFLINE operation executed successfully.")
	}()
}

func runHopfieldCapacityBenchmark() {
	fmt.Println("\n--- BENCHMARK 2: Modern vs Classical Hopfield Memory Capacity ---")
	sys := core.NewSystem()
	rng := rand.New(rand.NewSource(42))

	N := 20         // Vector dimension
	numPatterns := 8 // Exceeds classical capacity ~0.15*20 = 3

	classical := memory.NewClassicalHopfield(N)
	modern := memory.NewModernHopfield(N, 5.0)

	patterns := make([][]float64, numPatterns)
	for p := 0; p < numPatterns; p++ {
		pat := make([]float64, N)
		for i := 0; i < N; i++ {
			if rng.Float64() < 0.5 {
				pat[i] = 1.0
			} else {
				pat[i] = -1.0
			}
		}
		patterns[p] = pat
		classical.StorePattern(sys, pat)
		modern.StorePattern(sys, pat)
	}

	fmt.Printf("Stored %d bipolar patterns (Dim N=%d). Testing noisy recall (10%% bit flips)...\n", numPatterns, N)

	classicalSuccess := 0
	modernSuccess := 0

	for p := 0; p < numPatterns; p++ {
		noisy := append([]float64{}, patterns[p]...)
		// Flip 2 bits (~10% noise)
		for _, idx := range rng.Perm(N)[:2] {
			noisy[idx] = -noisy[idx]
		}

		recClass := classical.Recall(sys, noisy, 20)
		recMod := modern.Recall(sys, noisy)

		if isBipolarEqual(recClass, patterns[p]) {
			classicalSuccess++
		}
		if isBipolarEqualThreshold(recMod, patterns[p]) {
			modernSuccess++
		}
	}

	fmt.Printf("  Classical Hopfield (1982) Recoveries: %d / %d\n", classicalSuccess, numPatterns)
	fmt.Printf("  Modern Continuous Hopfield (2020) Recoveries: %d / %d\n", modernSuccess, numPatterns)
}

func runHomeostasisPlasticityBenchmark() {
	fmt.Println("\n--- BENCHMARK 3: Homeostasis & Dopamine-Modulated Plasticity ---")
	sys := core.NewSystem()
	homeo := homeostasis.NewState()
	g := graph.NewGraph()

	// Create 4 nodes
	for i := 0; i < 4; i++ {
		g.AddNode(graph.NewNode(i, 0.1, 0))
	}
	g.AddEdge(0, 1, 0.2, false)
	g.AddEdge(1, 2, 0.2, false)

	fmt.Printf("Initial state: Energy=%.2f Dopamine=%.2f DriveError=%.2f\n",
		homeo.Energy, homeo.Dopamine, homeo.DriveError())

	// Simulate 5 steps of environmental interaction
	for step := 1; step <= 5; step++ {
		prevError := homeo.DriveError()

		// System consumes energy, but finds reward
		homeo.Energy -= 0.1
		if step%2 == 0 {
			homeo.Energy += 0.25 // reward!
		}
		homeo.Curiosity -= 0.05

		newError := homeo.DriveError()
		homeo.UpdateHormones(prevError, newError)

		// Propagate & Hebbian update
		inputs := map[int]float64{0: 1.0, 1: 0.8}
		g.Propagate(sys, inputs)
		g.HebbianUpdate(sys, 0.2, homeo.Dopamine, 5.0)

		fmt.Printf("  Step %d: DriveError=%.2f | Dopamine=%.2f Cortisol=%.2f | Edge(0->1) weight=%.4f\n",
			step, newError, homeo.Dopamine, homeo.Cortisol, g.Nodes[0].Edges[1].Weight)
	}
}

func runSubconsciousSleepBenchmark() {
	fmt.Println("\n--- BENCHMARK 4: Subconscious Sleep & Louvain Graph Reorganization ---")
	sys := core.NewSystem()
	g := graph.NewGraph()

	// Create 10 nodes forming 2 dense clusters + weak random noise edges
	for i := 0; i < 10; i++ {
		g.AddNode(graph.NewNode(i, 0.1, 0))
	}

	// Cluster 1 (0..4)
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i != j {
				g.AddEdge(i, j, 0.8, false)
			}
		}
	}

	// Cluster 2 (5..9)
	for i := 5; i < 10; i++ {
		for j := 5; j < 10; j++ {
			if i != j {
				g.AddEdge(i, j, 0.7, false)
			}
		}
	}

	// Weak noise edges (<0.05) between clusters
	g.AddEdge(2, 7, 0.02, true)
	g.AddEdge(3, 8, 0.01, true)

	fmt.Println("Idle timeout detected. Transitioning ONLINE -> OFFLINE...")
	sys.Mode = core.Offline

	stats := offline.SubconsciousSleep(sys, g, 0.05)

	fmt.Printf("Sleep Cycle Complete. Re-assigned %d nodes into %d communities.\n",
		len(g.Nodes), stats.ClustersFound)

	sys.Mode = core.Online
	fmt.Printf("Transitioning back OFFLINE -> ONLINE. Current mode: %s\n", sys.Mode)
}

func isBipolarEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isBipolarEqualThreshold(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		signA := 1.0
		if a[i] < 0 {
			signA = -1.0
		}
		if signA != b[i] {
			return false
		}
	}
	return true
}
