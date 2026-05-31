package termui

import (
	"testing"
	"time"

	"github.com/nishchay-veer/gladiator/internal/config"
)

func TestRenderRateUsesConfiguredRate(t *testing.T) {
	cfg := config.Default()
	cfg.RenderRate = 25 * time.Millisecond

	if got := renderRate(cfg); got != 25*time.Millisecond {
		t.Fatalf("renderRate() = %s, want 25ms", got)
	}
}

func TestRenderRateFallsBackToSimulationRate(t *testing.T) {
	cfg := config.Default()
	cfg.RenderRate = 0
	cfg.SimulationRate = 17 * time.Millisecond

	if got := renderRate(cfg); got != 17*time.Millisecond {
		t.Fatalf("renderRate() = %s, want 17ms", got)
	}
}
