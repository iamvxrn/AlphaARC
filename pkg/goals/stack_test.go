package goals

import "testing"

func TestPushCreatesRootGoalWhenStackEmpty(t *testing.T) {
	s := NewStack()
	g := s.Push("Fix system", TypeExternal, 1.0, 1)

	if g.ParentID != 0 {
		t.Fatalf("FAIL: expected root goal to have ParentID=0, got %d", g.ParentID)
	}
	if s.Depth() != 1 {
		t.Fatalf("FAIL: expected depth 1 after first push, got %d", s.Depth())
	}
	if s.Top() != g {
		t.Fatalf("FAIL: expected Top() to return the just-pushed goal")
	}
}

func TestPushCreatesChildOfCurrentTop(t *testing.T) {
	s := NewStack()
	root := s.Push("Fix system", TypeExternal, 1.0, 1)
	child := s.Push("Isolate faulty node", TypeInternal, 0.8, 2)
	grandchild := s.Push("Add index edge", TypeInternal, 0.5, 3)

	if child.ParentID != root.ID {
		t.Fatalf("FAIL: expected child's ParentID=%d (root), got %d", root.ID, child.ParentID)
	}
	if grandchild.ParentID != child.ID {
		t.Fatalf("FAIL: expected grandchild's ParentID=%d (child), got %d", child.ID, grandchild.ParentID)
	}
	if s.Depth() != 3 {
		t.Fatalf("FAIL: expected depth 3 after 3 pushes, got %d", s.Depth())
	}
	if s.Top() != grandchild {
		t.Fatalf("FAIL: expected Top() to be the most recently pushed goal (grandchild)")
	}

	t.Logf("Hierarchy PASS: root=%q(id=%d) -> child=%q(parent=%d) -> grandchild=%q(parent=%d)",
		root.Description, root.ID, child.Description, child.ParentID, grandchild.Description, grandchild.ParentID)
}

func TestPopReturnsToParentAndMarksSatisfied(t *testing.T) {
	s := NewStack()
	root := s.Push("Fix system", TypeExternal, 1.0, 1)
	child := s.Push("Isolate faulty node", TypeInternal, 0.8, 2)

	popped := s.Pop()
	if popped != child {
		t.Fatalf("FAIL: expected Pop() to return the child goal")
	}
	if !popped.Satisfied {
		t.Fatalf("FAIL: expected popped goal to be marked Satisfied")
	}
	if s.Depth() != 1 {
		t.Fatalf("FAIL: expected depth 1 after popping the child, got %d", s.Depth())
	}
	if s.Top() != root {
		t.Fatalf("FAIL: expected control to return to the root goal after popping its child")
	}
	if root.Satisfied {
		t.Fatalf("FAIL: popping the child must not mark the parent Satisfied")
	}
}

func TestPopOnEmptyStackReturnsNilWithoutPanic(t *testing.T) {
	s := NewStack()
	if got := s.Pop(); got != nil {
		t.Fatalf("FAIL: expected Pop() on an empty stack to return nil, got %v", got)
	}
	if got := s.Top(); got != nil {
		t.Fatalf("FAIL: expected Top() on an empty stack to return nil, got %v", got)
	}
}

func TestAncestryReflectsFullActiveChain(t *testing.T) {
	s := NewStack()
	s.Push("Fix system", TypeExternal, 1.0, 1)
	s.Push("Isolate faulty node", TypeInternal, 0.8, 2)
	s.Push("Add index edge", TypeInternal, 0.5, 3)

	chain := s.Ancestry()
	if len(chain) != 3 {
		t.Fatalf("FAIL: expected ancestry length 3, got %d", len(chain))
	}
	wantDescriptions := []string{"Fix system", "Isolate faulty node", "Add index edge"}
	for i, want := range wantDescriptions {
		if chain[i].Description != want {
			t.Fatalf("FAIL: expected ancestry[%d]=%q, got %q", i, want, chain[i].Description)
		}
	}

	s.Pop()
	chainAfterPop := s.Ancestry()
	if len(chainAfterPop) != 2 {
		t.Fatalf("FAIL: expected ancestry length 2 after popping the deepest goal, got %d", len(chainAfterPop))
	}
}

func TestRecordScopeTagsOnlyCurrentTop(t *testing.T) {
	s := NewStack()
	root := s.Push("Fix system", TypeExternal, 1.0, 1)
	s.RecordScope(0)

	child := s.Push("Isolate faulty node", TypeInternal, 0.8, 2)
	s.RecordScope(5)
	s.RecordScope(7)

	if len(root.ScopeClusters) != 1 || !root.ScopeClusters[0] {
		t.Fatalf("FAIL: expected root's scope to be exactly {0}, got %v", root.ScopeClusters)
	}
	if len(child.ScopeClusters) != 2 || !child.ScopeClusters[5] || !child.ScopeClusters[7] {
		t.Fatalf("FAIL: expected child's scope to be exactly {5,7}, got %v", child.ScopeClusters)
	}

	// RecordScope on an empty stack must not panic.
	empty := NewStack()
	empty.RecordScope(99)

	t.Logf("Scope PASS: root touched clusters %v, child touched clusters %v", root.ScopeClusters, child.ScopeClusters)
}
