package graph

import (
	"protaxon/pkg/core"
	"sort"
	"strings"
)

// AddLabel registers a keyword (case-insensitive) that, when found as a token
// inside a raw Observation string, seeds the given node during LookupSeeds.
func (g *Graph) AddLabel(keyword string, nodeID int) {
	if _, exists := g.Nodes[nodeID]; !exists {
		return
	}
	key := strings.ToLower(keyword)
	for _, id := range g.Labels[key] {
		if id == nodeID {
			return
		}
	}
	g.Labels[key] = append(g.Labels[key], nodeID)
}

// LookupSeeds implements the "Observation -> Node Lookup" step: it tokenizes
// a raw observation string and returns full-strength (1.0) seed activations
// for every node whose registered label appears as a token in the text.
func (g *Graph) LookupSeeds(observation string) map[int]float64 {
	seeds := make(map[int]float64)
	for _, tok := range strings.Fields(strings.ToLower(observation)) {
		tok = strings.Trim(tok, ".,:;!?()[]")
		for _, id := range g.Labels[tok] {
			seeds[id] = 1.0
		}
	}
	return seeds
}

// SpreadingActivation implements the "Spreading Activation" step: starting
// from seed node activations, it repeatedly runs Propagate for up to maxHops
// hops, applying hopDecay to the carried-forward activation each hop so that
// influence attenuates with graph distance and cannot grow unbounded on
// cyclic graphs. Seed nodes are re-injected at full strength every hop so the
// originating concepts remain part of their own active subgraph.
//
// Returns the final hop's activation map (node ID -> activation), suitable
// for ExtractActiveSubgraph.
func (g *Graph) SpreadingActivation(sys *core.System, seeds map[int]float64, maxHops int, hopDecay float64) map[int]float64 {
	sys.OnlineGuard("Graph.SpreadingActivation")

	if maxHops < 1 {
		maxHops = 1
	}
	if len(seeds) == 0 {
		return map[int]float64{}
	}

	inputs := make(map[int]float64, len(seeds))
	for id, act := range seeds {
		inputs[id] = act
	}

	var activations map[int]float64
	for hop := 0; hop < maxHops; hop++ {
		activations = g.Propagate(sys, inputs)

		next := make(map[int]float64, len(activations))
		for id, act := range activations {
			next[id] = act * hopDecay
		}
		for id, act := range seeds {
			next[id] = act
		}
		inputs = next
	}

	return activations
}

// ExtractActiveSubgraph implements the "Sub-graph Extraction" step: it
// filters an activation map down to the node IDs exceeding threshold,
// returned in sorted order for deterministic downstream Agent Context.
func ExtractActiveSubgraph(activations map[int]float64, threshold float64) []int {
	ids := make([]int, 0, len(activations))
	for id, act := range activations {
		if act > threshold {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}
