package termui

import (
	"math"

	"github.com/nishchay-veer/gladiator/internal/game"
)

const correctionFrames = 4

type correctionAnimation struct {
	Active bool
	Player game.PlayerID
	From   game.Point
	To     game.Point
	Frame  int
	Frames int
}

func newCorrectionAnimation(player game.PlayerID, from, to game.Point) correctionAnimation {
	if from == to {
		return correctionAnimation{}
	}
	return correctionAnimation{
		Active: true,
		Player: player,
		From:   from,
		To:     to,
		Frames: correctionFrames,
	}
}

func (c correctionAnimation) Position() game.Point {
	if !c.Active || c.Frames <= 0 || c.Frame >= c.Frames {
		return c.To
	}

	progress := float64(c.Frame) / float64(c.Frames)
	return game.Point{
		X: interpolateAxis(c.From.X, c.To.X, progress),
		Y: interpolateAxis(c.From.Y, c.To.Y, progress),
	}
}

func (c correctionAnimation) PositionFor(player game.Player) game.Point {
	if !c.Active || c.Player != player.ID || !player.Alive() {
		return player.Position
	}
	return c.Position()
}

func (c *correctionAnimation) Advance() {
	if c == nil || !c.Active {
		return
	}

	c.Frame++
	if c.Frame >= c.Frames {
		c.Active = false
	}
}

func (c *correctionAnimation) Retarget(to game.Point) {
	if c == nil || !c.Active || c.To == to {
		return
	}

	*c = newCorrectionAnimation(c.Player, c.Position(), to)
}

func interpolateAxis(from, to int, progress float64) int {
	return int(math.Round(float64(from) + float64(to-from)*progress))
}
