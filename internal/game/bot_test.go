package game

import "testing"

func TestBotAimsAtVisibleTargetBeforeFiring(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[1].Position = Point{X: 8, Y: 5}
	stepWithCommand(&state, state.BotCommand(PlayerTwo, PlayerOne, state.Tick))

	if state.Players[1].Facing != Left {
		t.Fatalf("bot facing = %v, want Left", state.Players[1].Facing)
	}
	if len(state.Bullets) != 0 {
		t.Fatalf("len(Bullets) = %d, want 0 during windup", len(state.Bullets))
	}
}

func TestBotFiresAtVisibleTargetOnCadence(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Tick = botFireWindupTicks
	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[1].Position = Point{X: 8, Y: 5}
	stepWithCommand(&state, state.BotCommand(PlayerTwo, PlayerOne, state.Tick))

	if state.Players[1].Facing != Left {
		t.Fatalf("bot facing = %v, want Left", state.Players[1].Facing)
	}
	if len(state.Bullets) != 1 {
		t.Fatalf("len(Bullets) = %d, want 1", len(state.Bullets))
	}
	if state.Bullets[0].Owner != PlayerTwo {
		t.Fatalf("bullet owner = %v, want PlayerTwo", state.Bullets[0].Owner)
	}
}

func TestBotCanKillPlayerOne(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[0].Health = 1
	state.Players[1].Position = Point{X: 6, Y: 5}
	state.Tick = botFireWindupTicks

	stepWithCommand(&state, state.BotCommand(PlayerTwo, PlayerOne, state.Tick))

	if state.Players[0].Alive() {
		t.Fatal("player one should be down after adjacent bot shot")
	}
	if state.Players[1].Score != 1 {
		t.Fatalf("player two score = %d, want 1", state.Players[1].Score)
	}
}

func TestBotChasesWhenTargetIsNotVisible(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[1].Position = Point{X: 10, Y: 10}
	stepWithCommand(&state, state.BotCommand(PlayerTwo, PlayerOne, state.Tick))

	if state.Players[1].Position != (Point{X: 9, Y: 10}) {
		t.Fatalf("bot position = %+v, want {X:9 Y:10}", state.Players[1].Position)
	}
}
