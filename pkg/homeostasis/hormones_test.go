package homeostasis

import (
	"testing"
)

func TestHormoneSingleUpdate(t *testing.T) {
	state := NewState()
	state.Energy = 0.5

	initialErr := state.DriveError()
	if initialErr != 1.0 {
		t.Fatalf("Expected initial DriveError=1.0, got %.4f", initialErr)
	}

	prevErr := state.DriveError()
	state.Energy = 1.0
	newErr := state.DriveError()

	state.UpdateHormones(prevErr, newErr)

	if state.Dopamine <= 1.0 {
		t.Fatalf("Expected Dopamine > 1.0 on drive improvement, got %.4f", state.Dopamine)
	}

	if state.Cortisol != 0.0 {
		t.Fatalf("Expected Cortisol 0.0 on drive improvement, got %.4f", state.Cortisol)
	}

	t.Logf("Hormone Single Update PASS: DriveError=%.4f, Dopamine=%.4f, Cortisol=%.4f",
		newErr, state.Dopamine, state.Cortisol)
}

func TestHomeostasisDriveConvergenceTable(t *testing.T) {
	state := NewState()

	state.Energy = 0.2
	state.Curiosity = 0.8
	state.Stress = 0.5
	state.Dopamine = 0.5
	state.Cortisol = 1.5

	t.Logf("=== HOMEOSTASIS DRIVE CONVERGENCE SIMULATION (15 STEPS) ===")
	t.Logf("%-5s | %-6s | %-9s | %-6s | %-10s | %-8s | %-8s | %-9s",
		"Step", "Energy", "Curiosity", "Stress", "DriveError", "Dopamine", "Cortisol", "Serotonin")
	t.Logf("-----------------------------------------------------------------------------------------")

	initialError := state.DriveError()
	errors := []float64{initialError}

	t.Logf("%-5d | %-6.2f | %-9.2f | %-6.2f | %-10.4f | %-8.4f | %-8.4f | %-9.4f",
		0, state.Energy, state.Curiosity, state.Stress, initialError, state.Dopamine, state.Cortisol, state.Serotonin)

	for step := 1; step <= 15; step++ {
		prevErr := state.DriveError()

		state.Energy += (1.0 - state.Energy) * 0.20
		state.Curiosity -= state.Curiosity * 0.20
		state.Stress -= state.Stress * 0.20

		newErr := state.DriveError()
		state.UpdateHormones(prevErr, newErr)

		errors = append(errors, newErr)

		t.Logf("%-5d | %-6.2f | %-9.2f | %-6.2f | %-10.4f | %-8.4f | %-8.4f | %-9.4f",
			step, state.Energy, state.Curiosity, state.Stress, newErr, state.Dopamine, state.Cortisol, state.Serotonin)
	}

	finalError := errors[len(errors)-1]

	if finalError >= initialError {
		t.Fatalf("FAIL: Homeostatic DriveError failed to reduce! Initial=%.4f, Final=%.4f", initialError, finalError)
	}
}

func TestHomeostasisMidSimulationPerturbation(t *testing.T) {
	state := NewState()

	state.Energy = 0.3
	state.Curiosity = 0.7
	state.Stress = 0.4

	t.Logf("=== HOMEOSTASIS PERTURBATION TEST (SHOCK AT STEP 8) ===")
	t.Logf("%-5s | %-6s | %-9s | %-6s | %-10s | %-8s | %-8s | %-9s | %-15s",
		"Step", "Energy", "Curiosity", "Stress", "DriveError", "Dopamine", "Cortisol", "Serotonin", "Event")
	t.Logf("---------------------------------------------------------------------------------------------------------")

	t.Logf("%-5d | %-6.2f | %-9.2f | %-6.2f | %-10.4f | %-8.4f | %-8.4f | %-9.4f | Initial",
		0, state.Energy, state.Curiosity, state.Stress, state.DriveError(), state.Dopamine, state.Cortisol, state.Serotonin)

	for step := 1; step <= 7; step++ {
		prevErr := state.DriveError()
		state.Energy += (1.0 - state.Energy) * 0.25
		state.Curiosity -= state.Curiosity * 0.25
		state.Stress -= state.Stress * 0.25
		newErr := state.DriveError()
		state.UpdateHormones(prevErr, newErr)

		t.Logf("%-5d | %-6.2f | %-9.2f | %-6.2f | %-10.4f | %-8.4f | %-8.4f | %-9.4f | Normal step",
			step, state.Energy, state.Curiosity, state.Stress, newErr, state.Dopamine, state.Cortisol, state.Serotonin)
	}

	// STEP 8 - SUDDEN ENVIRONMENTAL PERTURBATION / SHOCK!
	prevErr := state.DriveError()
	state.Energy = 0.05 // Severe energy collapse
	state.Stress = 0.95 // Massive stress spike
	newErr := state.DriveError()
	state.UpdateHormones(prevErr, newErr)

	t.Logf("%-5d | %-6.2f | %-9.2f | %-6.2f | %-10.4f | %-8.4f | %-8.4f | %-9.4f | SHOCK APPLIED!",
		8, state.Energy, state.Curiosity, state.Stress, newErr, state.Dopamine, state.Cortisol, state.Serotonin)

	if state.Cortisol <= 0.5 {
		t.Fatalf("FAIL: Cortisol failed to reactivate on environmental shock! Got Cortisol=%.4f", state.Cortisol)
	}

	// Steps 9-18 Recovery after shock
	for step := 9; step <= 18; step++ {
		pErr := state.DriveError()
		state.Energy += (1.0 - state.Energy) * 0.25
		state.Curiosity -= state.Curiosity * 0.25
		state.Stress -= state.Stress * 0.25
		nErr := state.DriveError()
		state.UpdateHormones(pErr, nErr)

		t.Logf("%-5d | %-6.2f | %-9.2f | %-6.2f | %-10.4f | %-8.4f | %-8.4f | %-9.4f | Post-shock recovery",
			step, state.Energy, state.Curiosity, state.Stress, nErr, state.Dopamine, state.Cortisol, state.Serotonin)
	}

	if state.Cortisol > 1e-6 {
		t.Fatalf("FAIL: Cortisol failed to return to 0.0 after post-shock recovery! Got Cortisol=%.4f", state.Cortisol)
	}

	t.Logf("Homeostasis Mid-Run Perturbation PASS: System reacted to shock at Step 8 (Cortisol spike 0.0 -> 2.0) and recovered Cortisol back to 0.0 by Step 18!")
}
