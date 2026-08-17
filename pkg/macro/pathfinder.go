package macro

import (
	"container/heap"
	"fmt"
	"math"

	"alphaarc/pkg/environment"
)

// Vector2D represents a movement in the grid.
type Vector2D struct {
	DX, DY int
}

// MotorKinematics auto-calibrates which buttons cause which directional movements
// and tracks the avatar/cursor's identity on the grid.
type MotorKinematics struct {
	ButtonMap    map[environment.ActionID]Vector2D
	ActionBtn    environment.ActionID // The button that paints/interacts (dx=0, dy=0)
	AvatarColor  int
	IsCalibrated bool
	BgColor      int
}

func NewMotorKinematics(bgColor int) *MotorKinematics {
	return &MotorKinematics{
		ButtonMap:   make(map[environment.ActionID]Vector2D),
		AvatarColor: -1,
		ActionBtn:   0, // 0 is invalid/unset
		BgColor:     bgColor,
	}
}

// Observe records a frame transition to deduce avatar movement kinematics.
func (m *MotorKinematics) Observe(action environment.ActionID, before, after [][]int) {
	if m.IsCalibrated {
		return
	}

	// 1. Find all pixel changes
	type pt struct{ x, y int }
	var disappeared []pt
	var appeared []pt

	for y := 0; y < len(before) && y < len(after); y++ {
		for x := 0; x < len(before[y]) && x < len(after[y]); x++ {
			c1 := before[y][x]
			c2 := after[y][x]
			if c1 != c2 {
				if c1 != m.BgColor {
					disappeared = append(disappeared, pt{x, y})
				}
				if c2 != m.BgColor {
					appeared = append(appeared, pt{x, y})
				}
			}
		}
	}

	// If no movement happened, but pixels changed color, this is likely the interaction/paint button.
	if len(disappeared) == 0 && len(appeared) > 0 {
		m.ActionBtn = action
		return
	}

	// Simple heuristic: we assume the avatar translates cleanly.
	// We check if all appeared pixels are a strict translation of disappeared pixels.
	if len(appeared) > 0 && len(appeared) == len(disappeared) {
		// Calculate potential dx, dy from the first pixel
		dx := appeared[0].x - disappeared[0].x
		dy := appeared[0].y - disappeared[0].y

		valid := true
		for i := 1; i < len(appeared); i++ {
			if appeared[i].x-disappeared[i].x != dx || appeared[i].y-disappeared[i].y != dy {
				valid = false
				break
			}
		}

		if valid {
			m.ButtonMap[action] = Vector2D{DX: dx, DY: dy}
			m.AvatarColor = after[appeared[0].y][appeared[0].x]
		}
	}
}

// CheckCalibrated verifies if we have at least 1 directional button and the action button.
func (m *MotorKinematics) CheckCalibrated() bool {
	if len(m.ButtonMap) > 0 && m.AvatarColor != -1 {
		m.IsCalibrated = true
	}
	return m.IsCalibrated
}

// LocateAvatar finds the center/anchor point of the avatar in the current grid.
func (m *MotorKinematics) LocateAvatar(grid [][]int) (int, int, error) {
	if m.AvatarColor == -1 {
		return -1, -1, fmt.Errorf("avatar color unknown")
	}
	for y := 0; y < len(grid); y++ {
		for x := 0; x < len(grid[y]); x++ {
			if grid[y][x] == m.AvatarColor {
				// Return first found pixel (sufficient for 1x1 cursors).
				// For larger avatars, a centroid calculation can be added later.
				return x, y, nil
			}
		}
	}
	return -1, -1, fmt.Errorf("avatar not found on grid")
}

// --- A* Pathfinder implementation ---

type pathNode struct {
	x, y      int
	cost      int
	heuristic int
	index     int
	parent    *pathNode
}

type priorityQueue []*pathNode

func (pq priorityQueue) Len() int { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].cost+pq[i].heuristic < pq[j].cost+pq[j].heuristic
}
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*pathNode)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// AStar calculates a path from start to target.
// Treats m.BgColor and target color as walkable, everything else as obstacles.
func (m *MotorKinematics) AStar(grid [][]int, startX, startY, targetX, targetY int) []Vector2D {
	rows := len(grid)
	if rows == 0 {
		return nil
	}
	cols := len(grid[0])

	pq := make(priorityQueue, 0)
	heap.Init(&pq)

	startNode := &pathNode{x: startX, y: startY, cost: 0, heuristic: heuristic(startX, startY, targetX, targetY)}
	heap.Push(&pq, startNode)

	visited := make(map[string]bool)

	for pq.Len() > 0 {
		current := heap.Pop(&pq).(*pathNode)
		
		if current.x == targetX && current.y == targetY {
			return reconstructPath(current)
		}

		stateKey := fmt.Sprintf("%d,%d", current.x, current.y)
		if visited[stateKey] {
			continue
		}
		visited[stateKey] = true

		// Try available buttons (we only move using known kinematics)
		for _, vec := range m.ButtonMap {
			nx, ny := current.x+vec.DX, current.y+vec.DY
			
			// Bounds check
			if nx < 0 || ny < 0 || nx >= cols || ny >= rows {
				continue
			}

			// Obstacle check: we can walk on background, or our own avatar color, or the target cell
			cellColor := grid[ny][nx]
			if cellColor != m.BgColor && cellColor != m.AvatarColor && !(nx == targetX && ny == targetY) {
				continue
			}

			neighborState := fmt.Sprintf("%d,%d", nx, ny)
			if visited[neighborState] {
				continue
			}

			neighbor := &pathNode{
				x: nx, y: ny,
				cost:      current.cost + 1,
				heuristic: heuristic(nx, ny, targetX, targetY),
				parent:    current,
			}
			heap.Push(&pq, neighbor)
		}
	}
	return nil // No path found
}

func heuristic(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

func reconstructPath(endNode *pathNode) []Vector2D {
	var path []Vector2D
	curr := endNode
	for curr.parent != nil {
		dx := curr.x - curr.parent.x
		dy := curr.y - curr.parent.y
		path = append([]Vector2D{{DX: dx, DY: dy}}, path...)
		curr = curr.parent
	}
	return path
}

// CompilePath translates a sequence of grid vectors into actual button ActionIDs.
func (m *MotorKinematics) CompilePath(path []Vector2D) []environment.Action {
	var actions []environment.Action
	
	// Reverse lookup map for vectors to buttons
	vecToBtn := make(map[Vector2D]environment.ActionID)
	for btn, vec := range m.ButtonMap {
		vecToBtn[vec] = btn
	}

	for _, vec := range path {
		if btn, ok := vecToBtn[vec]; ok {
			actions = append(actions, environment.Action{ID: btn})
		}
	}
	
	// If we have an action button (paint), we press it at the destination
	if m.ActionBtn != 0 {
		actions = append(actions, environment.Action{ID: m.ActionBtn})
	}
	
	return actions
}
