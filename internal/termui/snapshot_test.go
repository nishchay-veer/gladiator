package termui

import (
	"testing"

	"gladiator/internal/game"
)

func TestApplySnapshotPreservesArenaAndSpawns(t *testing.T) {
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	snapshot := state.Snapshot()
	snapshot.Tick = 12
	snapshot.Players[0].Position = game.Point{X: 3, Y: 1}
	snapshot.Players[0].Facing = game.Right
	snapshot.Players[0].Score = 2
	snapshot.Bullets = []game.BulletSnapshot{
		{
			Position:  game.Point{X: 4, Y: 1},
			Direction: game.Right,
			Owner:     game.PlayerOne,
			Age:       1,
			TTL:       10,
		},
	}

	got := applySnapshot(state, snapshot)
	if got.Arena.Width != state.Arena.Width || got.Arena.Height != state.Arena.Height {
		t.Fatal("arena dimensions changed after applying snapshot")
	}
	if got.Players[0].Spawn != state.Players[0].Spawn {
		t.Fatalf("player one spawn = %+v, want %+v", got.Players[0].Spawn, state.Players[0].Spawn)
	}
	if got.Tick != snapshot.Tick {
		t.Fatalf("tick = %d, want %d", got.Tick, snapshot.Tick)
	}
	if got.Players[0].Position != snapshot.Players[0].Position {
		t.Fatalf("player one position = %+v, want %+v", got.Players[0].Position, snapshot.Players[0].Position)
	}
	if len(got.Bullets) != 1 || got.Bullets[0].Position != snapshot.Bullets[0].Position {
		t.Fatalf("bullets = %#v, want %#v", got.Bullets, snapshot.Bullets)
	}
}
