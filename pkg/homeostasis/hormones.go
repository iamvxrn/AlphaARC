package homeostasis

import "math"

// State represents the internal homeostatic metrics of the agent.
type State struct {
	Energy    float64 // Goal setpoint = 1.0
	Curiosity float64 // Goal setpoint = 0.0
	Stress    float64 // Goal setpoint = 0.0

	Dopamine  float64 // Global plasticity multiplier (Reward / Prediction Error)
	Cortisol  float64 // Stress / Alarm signal
	Serotonin float64 // Stability / Satisfaction signal
}

func NewState() *State {
	return &State{
		Energy:    1.0,
		Curiosity: 0.5,
		Stress:    0.0,
		Dopamine:  1.0,
		Cortisol:  0.0,
		Serotonin: 0.5,
	}
}

// DriveError calculates the L1 distance to homeostatic setpoints.
func (s *State) DriveError() float64 {
	errEnergy := math.Abs(1.0 - s.Energy)
	errCuriosity := math.Abs(0.0 - s.Curiosity)
	errStress := math.Abs(0.0 - s.Stress)
	return errEnergy + errCuriosity + errStress
}

// UpdateHormones adjusts hormone levels based on drive error delta.
func (s *State) UpdateHormones(prevDriveError, newDriveError float64) {
	delta := prevDriveError - newDriveError // positive = improvement

	if delta > 0 {
		// Drive error reduced: Dopamine & Serotonin boost, Cortisol drops
		s.Dopamine = math.Min(2.0, 1.0+delta*2.0)
		s.Serotonin = math.Min(1.0, s.Serotonin+0.1)
		s.Cortisol = math.Max(0.0, s.Cortisol-0.2)
	} else {
		// Drive error increased: Cortisol rises, Dopamine drops
		s.Dopamine = math.Max(0.1, 1.0+delta)
		s.Cortisol = math.Min(2.0, s.Cortisol+math.Abs(delta)*2.0)
		s.Serotonin = math.Max(0.0, s.Serotonin-0.1)
	}
}
