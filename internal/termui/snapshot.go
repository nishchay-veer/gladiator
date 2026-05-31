package termui

import "gladiator/internal/game"

func applySnapshot(base game.State, snapshot game.Snapshot) game.State {
	return base.ApplySnapshot(snapshot)
}
