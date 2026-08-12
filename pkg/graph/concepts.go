package graph

import "strings"

// stopWords are filtered out of observation text so common function words
// never become graph nodes -- only meaningful concept vocabulary does.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true,
	"be": true, "to": true, "of": true, "in": true, "on": true, "for": true,
	"with": true, "and": true, "or": true, "at": true, "by": true, "from": true,
	"as": true, "this": true, "that": true, "it": true, "via": true,
}

// EnsureConceptNodes implements "Observation -> Node Lookup-or-Create": every
// meaningful token (length >= 3, not a stopword) in observation that has no
// registered label yet becomes a brand new graph node, auto-labeled with
// that token. This is what lets the graph grow organically from real
// experience instead of staying pinned to whatever nodes were hand-seeded at
// construction time -- LookupSeeds alone (Stage 2) can only recognize
// vocabulary that was already registered; it never adds to the graph.
//
// Returns the full seed activation set for the observation (pre-existing
// and newly created concepts alike), exactly like LookupSeeds but after
// growing the graph first.
func (g *Graph) EnsureConceptNodes(observation string, threshold float64) map[int]float64 {
	nextID := 0
	for id := range g.Nodes {
		if id >= nextID {
			nextID = id + 1
		}
	}

	for _, tok := range strings.Fields(strings.ToLower(observation)) {
		tok = strings.Trim(tok, ".,:;!?()[]")
		if len(tok) < 3 || stopWords[tok] {
			continue
		}
		if _, exists := g.Labels[tok]; exists {
			continue
		}
		node := NewNode(nextID, threshold, 0)
		g.AddNode(node)
		g.AddLabel(tok, nextID)
		nextID++
	}

	return g.LookupSeeds(observation)
}
