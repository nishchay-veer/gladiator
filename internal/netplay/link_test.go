package netplay

import (
	"testing"
	"time"
)

func TestLinkSimulationDelayPatternIsDeterministic(t *testing.T) {
	simulation := LinkSimulation{
		BaseDelay: 10 * time.Millisecond,
		Jitter:    8 * time.Millisecond,
	}

	delays := []time.Duration{
		simulation.delayFor(1),
		simulation.delayFor(2),
		simulation.delayFor(3),
		simulation.delayFor(4),
	}
	want := []time.Duration{
		10 * time.Millisecond,
		14 * time.Millisecond,
		18 * time.Millisecond,
		12 * time.Millisecond,
	}

	for i := range delays {
		if delays[i] != want[i] {
			t.Fatalf("delay %d = %s, want %s", i+1, delays[i], want[i])
		}
	}
}

func TestLinkSimulationNormalizesNegativeValues(t *testing.T) {
	simulation := LinkSimulation{
		DropEvery: -1,
		BaseDelay: -time.Millisecond,
		Jitter:    -time.Millisecond,
	}.normalized()

	if simulation.DropEvery != 0 {
		t.Fatalf("drop every = %d, want 0", simulation.DropEvery)
	}
	if simulation.BaseDelay != 0 {
		t.Fatalf("base delay = %s, want 0", simulation.BaseDelay)
	}
	if simulation.Jitter != 0 {
		t.Fatalf("jitter = %s, want 0", simulation.Jitter)
	}
}
