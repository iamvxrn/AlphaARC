// Package goals implements a hierarchical goal stack (CONCEPT.md Section 8:
// external / internal / meta goals) whose active entries can be tagged with
// the graph ClusterIDs that were relevant while pursuing them -- the
// "multi-layered subgraphs" side of Stage 4 piece 2.
package goals

// Type distinguishes the three goal classes from CONCEPT.md Section 8.
type Type string

const (
	// TypeExternal goals come from the user or environment.
	TypeExternal Type = "external"
	// TypeInternal goals serve the system's own stability (memory quality,
	// reducing uncertainty, avoiding repeated errors, conserving resources).
	TypeInternal Type = "internal"
	// TypeMeta goals target the system's own architecture or strategy.
	TypeMeta Type = "meta"
)

// Goal is one entry in the hierarchy: pushing a goal makes it a child of
// whatever was on top of the stack at the time.
type Goal struct {
	ID            int
	Description   string
	Type          Type
	Priority      float64
	ParentID      int // 0 = root goal, no parent
	CreatedAtStep int
	Satisfied     bool
	// ScopeClusters records which graph ClusterIDs were active while this
	// goal was the current top of the stack, built incrementally via
	// Stack.RecordScope -- this goal's real footprint in the graph, not
	// just a description string.
	ScopeClusters map[int]bool
}

// Stack is a LIFO hierarchical goal stack. The current top is always the
// most specific, deepest goal being actively pursued; popping it returns
// control to its parent.
type Stack struct {
	goals  []*Goal
	nextID int
}

// NewStack returns an empty goal stack.
func NewStack() *Stack {
	return &Stack{nextID: 1}
}

// Push adds a new goal as a child of the current top (or as a root goal if
// the stack is empty) and makes it the new top.
func (s *Stack) Push(description string, goalType Type, priority float64, atStep int) *Goal {
	parentID := 0
	if top := s.Top(); top != nil {
		parentID = top.ID
	}
	g := &Goal{
		ID:            s.nextID,
		Description:   description,
		Type:          goalType,
		Priority:      priority,
		ParentID:      parentID,
		CreatedAtStep: atStep,
		ScopeClusters: make(map[int]bool),
	}
	s.nextID++
	s.goals = append(s.goals, g)
	return g
}

// Pop marks the current top goal Satisfied and removes it, returning
// control to its parent (the new top). Returns nil on an empty stack.
func (s *Stack) Pop() *Goal {
	if len(s.goals) == 0 {
		return nil
	}
	g := s.goals[len(s.goals)-1]
	g.Satisfied = true
	s.goals = s.goals[:len(s.goals)-1]
	return g
}

// Top returns the current, most specific active goal, or nil if the stack
// is empty.
func (s *Stack) Top() *Goal {
	if len(s.goals) == 0 {
		return nil
	}
	return s.goals[len(s.goals)-1]
}

// Depth returns how many goals are currently active (the hierarchy's depth).
func (s *Stack) Depth() int {
	return len(s.goals)
}

// Ancestry returns the full active chain from the root goal down to the
// current top, e.g. [FixSystem, IsolateFaultyNode, AddIndexEdge].
func (s *Stack) Ancestry() []*Goal {
	out := make([]*Goal, len(s.goals))
	copy(out, s.goals)
	return out
}

// RecordScope tags the current top goal with a graph ClusterID that was
// active while pursuing it. A no-op on an empty stack. This is what makes
// the hierarchy a real "multi-layered subgraph" structure instead of a
// purely symbolic stack of description strings -- each goal accumulates the
// set of graph regions it actually touched.
func (s *Stack) RecordScope(clusterID int) {
	if top := s.Top(); top != nil {
		top.ScopeClusters[clusterID] = true
	}
}
