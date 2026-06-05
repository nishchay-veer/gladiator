package game

import "testing"

func TestMatchEndsAtScoreLimit(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Match.ScoreLimit = 2
	state.Players[0].Score = state.Match.ScoreLimit - 1
	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[0].Facing = Right
	state.Players[1].Position = Point{X: 6, Y: 5}
	state.Players[1].Health = 1

	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, Buttons: ButtonFire})

	if !state.Match.Over {
		t.Fatal("match should end when score limit is reached")
	}
	if !state.Match.HasWinner || state.Match.Winner != PlayerOne {
		t.Fatalf("winner = %v hasWinner=%v, want PlayerOne", state.Match.Winner, state.Match.HasWinner)
	}
}

func TestLocalMatchKeepsRunningAfterFiveScores(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[1].Score = 4
	state.Players[1].Position = Point{X: 5, Y: 5}
	state.Players[1].Facing = Right
	state.Players[0].Position = Point{X: 6, Y: 5}
	state.Players[0].Health = 1

	stepWithCommand(&state, InputCommand{PlayerID: PlayerTwo, Buttons: ButtonFire})

	if state.Players[1].Score != 5 {
		t.Fatalf("player two score = %d, want 5", state.Players[1].Score)
	}
	if state.Match.Over {
		t.Fatal("local match should keep running after five scores")
	}

	for i := 0; i < respawnDelayTicks-1; i++ {
		state.Step(NewInputFrame(state.Tick))
	}

	if !state.Players[0].Alive() {
		t.Fatal("player one should respawn after the local score passes five")
	}
}

func TestResetMatchRestoresInitialState(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Tick = 99
	state.Players[0].Score = 3
	state.Players[0].Health = 1
	state.Bullets = append(state.Bullets, Bullet{
		Position:  Point{X: 2, Y: 2},
		Direction: Right,
		Owner:     PlayerOne,
		TTL:       10,
	})

	if err := state.ResetMatch(); err != nil {
		t.Fatalf("ResetMatch() error = %v", err)
	}
	if state.Tick != 0 {
		t.Fatalf("tick = %d, want 0", state.Tick)
	}
	if state.Players[0].Score != 0 || state.Players[1].Score != 0 {
		t.Fatalf("scores = %d/%d, want 0/0", state.Players[0].Score, state.Players[1].Score)
	}
	if state.Players[0].Health != state.Players[0].MaxHealth {
		t.Fatalf("p1 health = %d, want max %d", state.Players[0].Health, state.Players[0].MaxHealth)
	}
	if len(state.Bullets) != 0 {
		t.Fatalf("bullets = %d, want 0", len(state.Bullets))
	}
}

func TestMatchEndsAtTimeLimit(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Match.TimeLimitTicks = 2
	state.Players[0].Score = 1
	state.Step(NewInputFrame(state.Tick))
	state.Step(NewInputFrame(state.Tick))

	if !state.Match.Over {
		t.Fatal("match should end when time limit is reached")
	}
	if !state.Match.HasWinner || state.Match.Winner != PlayerOne {
		t.Fatalf("winner = %v hasWinner=%v, want PlayerOne", state.Match.Winner, state.Match.HasWinner)
	}
}
