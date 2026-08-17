package goals

import (
	"math"
	"alphaarc/pkg/memory"
)

// ProposeIrreversibleTarget analyzes the HER transition memory and identifies
// states that are "irreversible" from the start state -- i.e., milestones that,
// once reached, the agent has never managed to return from.
// 
// This implements the "irreversibility" Goal Proposer, giving HER a target
// to aim for even when external reward (victory) is absent. Reversible changes
// (toggles) are noise; irreversible changes (one-way doors) are progress.
//
// How it works:
// 1. Groups similar states across all episodes into clusters (nodes).
// 2. Adds directed edges for every observed transition.
// 3. Finds clusters that are reachable from the start state, but have NO path
//    back to the start state.
// 4. Returns the state vector of the deepest such irreversible cluster.
// Returns nil if no irreversible state is found.
func ProposeIrreversibleTarget(her *memory.HindsightMemory, startState []float64, similarityThreshold float64) []float64 {
	episodes := her.Episodes()
	if len(episodes) == 0 {
		return nil
	}

	var centroids [][]float64
	
	// Helper to find or create a cluster ID for a state vector.
	getClusterID := func(vec []float64) int {
		bestID := -1
		bestSim := math.Inf(-1)
		for i, c := range centroids {
			sim := memory.CosineSimilarity(c, vec)
			if sim > bestSim {
				bestSim = sim
				bestID = i
			}
		}
		if bestSim >= similarityThreshold {
			return bestID
		}
		centroids = append(centroids, vec)
		return len(centroids) - 1
	}

	startID := getClusterID(startState)

	// Build the transition graph.
	// adj[u] contains all clusters reachable in 1 step from u.
	// rev[u] contains all clusters that can reach u in 1 step.
	adj := make(map[int]map[int]bool)
	rev := make(map[int]map[int]bool)

	addEdge := func(u, v int) {
		if adj[u] == nil {
			adj[u] = make(map[int]bool)
		}
		adj[u][v] = true
		
		if rev[v] == nil {
			rev[v] = make(map[int]bool)
		}
		rev[v][u] = true
	}

	for _, ep := range episodes {
		if len(ep.Transitions) == 0 {
			continue
		}
		// First transition's starting state
		prevID := getClusterID(ep.Transitions[0].State)
		for _, tr := range ep.Transitions {
			currID := getClusterID(tr.NextState)
			if prevID != currID {
				addEdge(prevID, currID)
			}
			prevID = currID
		}
	}

	// 1. Find all clusters reachable from startID.
	reachableFromStart := make(map[int]bool)
	queue := []int{startID}
	reachableFromStart[startID] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for next := range adj[curr] {
			if !reachableFromStart[next] {
				reachableFromStart[next] = true
				queue = append(queue, next)
			}
		}
	}

	// 2. Find all clusters that can reach startID (backward search).
	canReachStart := make(map[int]bool)
	queue = []int{startID}
	canReachStart[startID] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for prev := range rev[curr] {
			if !canReachStart[prev] {
				canReachStart[prev] = true
				queue = append(queue, prev)
			}
		}
	}

	// 3. Irreversible clusters are those reachable from start, but cannot reach start.
	// We want to find the "deepest" one. We can define deepest as the maximum shortest
	// path distance from the start node.
	distances := make(map[int]int)
	queue = []int{startID}
	distances[startID] = 0

	deepestID := -1
	maxDist := -1

	for len(queue) > 0 {
		curr := queue[0]
		dist := distances[curr]
		queue = queue[1:]

		// Is it irreversible?
		if !canReachStart[curr] {
			if dist > maxDist {
				maxDist = dist
				deepestID = curr
			}
		}

		for next := range adj[curr] {
			if _, visited := distances[next]; !visited {
				distances[next] = dist + 1
				queue = append(queue, next)
			}
		}
	}

	if deepestID != -1 {
		return centroids[deepestID]
	}

	return nil
}
