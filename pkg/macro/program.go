package macro

import (
	"fmt"
)

// Program defines the interface for all macro-actions.
type Program interface {
	Apply(input [][]int) [][]int
	Complexity() float64
	Name() string
	ChangesShape() bool
}

// IdentityProgram returns the grid unchanged.
type IdentityProgram struct{}

func (p *IdentityProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		copy(out[r], input[r])
	}
	return out
}
func (p *IdentityProgram) Complexity() float64 { return 0.5 }
func (p *IdentityProgram) Name() string        { return "Identity()" }
func (p *IdentityProgram) ChangesShape() bool  { return false }

// RecolorProgram changes all pixels to TargetColor.
type RecolorProgram struct {
	TargetColor int
}

func (p *RecolorProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		for c := range input[r] {
			out[r][c] = p.TargetColor
		}
	}
	return out
}
func (p *RecolorProgram) Complexity() float64 { return 1.0 }
func (p *RecolorProgram) Name() string        { return fmt.Sprintf("Recolor(All, %d)", p.TargetColor) }
func (p *RecolorProgram) ChangesShape() bool  { return false }

// ColorMapProgram applies a global color translation mapping.
type ColorMapProgram struct {
	Mapping map[int]int
}

func (p *ColorMapProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		for c := range input[r] {
			if target, ok := p.Mapping[input[r][c]]; ok {
				out[r][c] = target
			} else {
				out[r][c] = input[r][c]
			}
		}
	}
	return out
}
func (p *ColorMapProgram) Complexity() float64 { return float64(len(p.Mapping)) }
func (p *ColorMapProgram) Name() string        { return fmt.Sprintf("ColorMap(%d rules)", len(p.Mapping)) }
func (p *ColorMapProgram) ChangesShape() bool  { return false }

// SequenceProgram applies multiple programs in order.
type SequenceProgram struct {
	Steps []Program
}

func (p *SequenceProgram) Apply(input [][]int) [][]int {
	curr := input
	for _, step := range p.Steps {
		curr = step.Apply(curr)
	}
	return curr
}
func (p *SequenceProgram) Complexity() float64 {
	c := 0.0
	for _, step := range p.Steps {
		c += step.Complexity()
	}
	return c + 0.5
}
func (p *SequenceProgram) Name() string {
	names := ""
	for i, step := range p.Steps {
		if i > 0 {
			names += " -> "
		}
		names += step.Name()
	}
	return fmt.Sprintf("Seq(%s)", names)
}
func (p *SequenceProgram) ChangesShape() bool {
	for _, step := range p.Steps {
		if step.ChangesShape() {
			return true
		}
	}
	return false
}
