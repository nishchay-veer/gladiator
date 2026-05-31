package game

import "testing"

type replayInput struct {
	tick    int
	player  PlayerID
	moveX   int
	moveY   int
	aim     Direction
	hasAim  bool
	buttons Buttons
}

var duelReplay = []replayInput{
	{tick: 0, player: PlayerOne, moveX: 1, aim: Right, hasAim: true},
	{tick: 1, player: PlayerOne, moveX: 1, aim: Right, hasAim: true},
	{tick: 2, player: PlayerOne, buttons: ButtonFire},
	{tick: 6, player: PlayerTwo, moveX: -1, aim: Left, hasAim: true},
	{tick: 8, player: PlayerOne, moveY: 1, aim: Down, hasAim: true},
	{tick: 10, player: PlayerTwo, buttons: ButtonFire},
}

func TestDuelReplayGoldenHash(t *testing.T) {
	snapshot := runReplayFixture(t, duelReplay, 18)
	const wantHash uint64 = 16150732565052198470

	if snapshot.Hash() != wantHash {
		t.Fatalf("snapshot hash = %d, want %d\nsnapshot: %#v", snapshot.Hash(), wantHash, snapshot)
	}
}

func runReplayFixture(t *testing.T, replay []replayInput, ticks int) Snapshot {
	t.Helper()

	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	for tick := 0; tick < ticks; tick++ {
		frame := NewInputFrame(state.Tick)
		for _, entry := range replay {
			if entry.tick != tick {
				continue
			}
			frame.Set(InputCommand{
				Tick:     state.Tick,
				PlayerID: entry.player,
				MoveX:    entry.moveX,
				MoveY:    entry.moveY,
				Aim:      entry.aim,
				HasAim:   entry.hasAim,
				Buttons:  entry.buttons,
			})
		}
		state.Step(frame)
	}

	return state.Snapshot()
}
