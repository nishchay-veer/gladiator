package termui

import (
	"testing"

	"github.com/nishchay-veer/gladiator/internal/game"
)

func TestCorrectionAnimationInterpolatesAndCompletes(t *testing.T) {
	correction := newCorrectionAnimation(game.PlayerTwo, game.Point{X: 35, Y: 14}, game.Point{X: 31, Y: 14})

	wantPositions := []game.Point{
		{X: 35, Y: 14},
		{X: 34, Y: 14},
		{X: 33, Y: 14},
		{X: 32, Y: 14},
	}
	for _, want := range wantPositions {
		if got := correction.Position(); got != want {
			t.Fatalf("position = %+v, want %+v", got, want)
		}
		correction.Advance()
	}

	if correction.Active {
		t.Fatal("correction active = true, want false after final frame")
	}
	if got := correction.Position(); got != (game.Point{X: 31, Y: 14}) {
		t.Fatalf("final position = %+v, want target", got)
	}
}

func TestCorrectionAnimationPositionForPlayer(t *testing.T) {
	correction := newCorrectionAnimation(game.PlayerTwo, game.Point{X: 35, Y: 14}, game.Point{X: 31, Y: 14})
	player := game.Player{
		ID:           game.PlayerTwo,
		Position:     game.Point{X: 31, Y: 14},
		Health:       1,
		RespawnTicks: 0,
	}

	if got := correction.PositionFor(player); got != (game.Point{X: 35, Y: 14}) {
		t.Fatalf("position for corrected player = %+v, want correction start", got)
	}

	player.ID = game.PlayerOne
	if got := correction.PositionFor(player); got != player.Position {
		t.Fatalf("position for other player = %+v, want actual %+v", got, player.Position)
	}
}

func TestCorrectionAnimationRetargetKeepsCurrentPosition(t *testing.T) {
	correction := newCorrectionAnimation(game.PlayerTwo, game.Point{X: 35, Y: 14}, game.Point{X: 31, Y: 14})
	correction.Advance()

	correction.Retarget(game.Point{X: 30, Y: 14})
	if got := correction.Position(); got != (game.Point{X: 34, Y: 14}) {
		t.Fatalf("retargeted start = %+v, want current visual position", got)
	}
	if correction.To != (game.Point{X: 30, Y: 14}) {
		t.Fatalf("retargeted target = %+v, want {X:30 Y:14}", correction.To)
	}
}
