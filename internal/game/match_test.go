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
