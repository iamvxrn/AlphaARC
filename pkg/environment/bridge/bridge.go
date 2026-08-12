// Package bridge connects Protaxon's graph/router cognition
// (pkg/pipeline.Engine) to an environment.Environment: Observe -> Decide ->
// Act, closing the loop that Stage 4 built the pieces for but never drove
// against a real interactive task.
package bridge

import (
	"context"
	"fmt"
	"protaxon/pkg/environment"
	"protaxon/pkg/graph"
	"protaxon/pkg/pipeline"
	"strings"
)

// DescribeFrame converts an agent/target position pair into a short
// textual observation Protaxon's graph-based cognition can act on: just the
// bare direction words needed, nothing else. (Deliberately no filler words
// like "target" -- EnsureConceptNodes would create a node for that word
// too, and it would compete with the real direction concepts for the
// router's attention on every single cycle, since it's a fixed word that
// always appears.) This is a deliberately simple perception stand-in -- a
// real vision pipeline over the raw 64x64 grid is future work; this exists
// to validate the Observe -> Decide -> Act loop end to end, not to solve
// perception.
func DescribeFrame(agentX, agentY, targetX, targetY int) string {
	words := make([]string, 0, 2)
	if targetY < agentY {
		words = append(words, "north")
	} else if targetY > agentY {
		words = append(words, "south")
	}
	if targetX < agentX {
		words = append(words, "west")
	} else if targetX > agentX {
		words = append(words, "east")
	}
	if len(words) == 0 {
		return "aligned"
	}
	return strings.Join(words, " ")
}

var directionActions = map[string]environment.ActionID{
	"north": environment.Action1,
	"south": environment.Action2,
	"west":  environment.Action3,
	"east":  environment.Action4,
}

// ChooseAction runs one predictive cycle against engine using observation,
// then reads back which direction concept actually won the Stage 2 router's
// competition (res.ActiveNodeIDs) and maps that label to a game action.
//
// This deliberately reuses Protaxon's existing graph/router machinery as
// the decision mechanism rather than adding a separate hand-written policy
// -- but it is NOT a claim that the choice is "intelligent." With no prior
// training on this specific task, which direction wins is governed by
// Winner-Takes-All tie-breaking among concepts (activation magnitude, and
// node-creation order on exact ties), not a learned preference. What this
// validates is that the full loop actually wires together end to end.
//
// actualSuccess is passed as true on every intermediate step -- there is no
// clear per-step reward signal from the environment yet (only a terminal
// WIN/GAME_OVER), so this is a known, documented simplification for this
// first version, not a claim that every step was actually "successful."
func ChooseAction(ctx context.Context, engine *pipeline.Engine, observation, goal string) (environment.Action, *pipeline.CycleResult, error) {
	res, err := engine.RunPredictiveCycle(ctx, observation, goal, true)
	if err != nil {
		return environment.Action{}, nil, fmt.Errorf("predictive cycle: %w", err)
	}

	for _, nodeID := range res.ActiveNodeIDs {
		word := labelForNode(engine.Graph, nodeID)
		if actionID, ok := directionActions[word]; ok {
			return environment.Action{ID: actionID}, res, nil
		}
	}

	// No recognized direction concept won this cycle -- shouldn't happen
	// once DescribeFrame emits at least one direction word, but stay safe
	// rather than returning a zero-value action that could hang the caller.
	return environment.Action{ID: environment.Action2}, res, nil
}

func labelForNode(g *graph.Graph, nodeID int) string {
	for keyword, ids := range g.Labels {
		for _, id := range ids {
			if id == nodeID {
				return keyword
			}
		}
	}
	return ""
}
