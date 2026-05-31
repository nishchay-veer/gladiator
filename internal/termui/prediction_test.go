package termui

import (
	"testing"

	"gladiator/internal/config"
	"gladiator/internal/game"
)

func TestApplyPredictedCommandRecordsHistory(t *testing.T) {
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	app := localApp{
		cfg:        config.Default(),
		state:      state,
		player:     game.PlayerTwo,
		prediction: game.NewPredictionHistory(4),
	}
	command := game.NewInputCommand(state.Tick, game.PlayerTwo)
	command.MoveX = -1
	command.Aim = game.Left
	command.HasAim = true

	app.applyPredictedCommand(command)

	if app.state.Tick != 1 {
		t.Fatalf("state tick = %d, want 1", app.state.Tick)
	}
	if app.state.Players[1].Position.X != state.Players[1].Position.X-1 {
		t.Fatalf("player two x = %d, want %d", app.state.Players[1].Position.X, state.Players[1].Position.X-1)
	}
	if app.prediction.Len() != 1 {
		t.Fatalf("prediction history len = %d, want 1", app.prediction.Len())
	}
}

func TestApplyAuthoritativeSnapshotReplaysPredictedCommands(t *testing.T) {
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	app := localApp{
		cfg:        config.Default(),
		state:      state,
		player:     game.PlayerTwo,
		prediction: game.NewPredictionHistory(8),
	}

	first := game.NewInputCommand(app.state.Tick, game.PlayerTwo)
	first.MoveX = -1
	first.Aim = game.Left
	first.HasAim = true
	app.applyPredictedCommand(first)
	authoritative := app.state.Snapshot()

	second := game.NewInputCommand(app.state.Tick, game.PlayerTwo)
	second.MoveX = -1
	second.Aim = game.Left
	second.HasAim = true
	app.applyPredictedCommand(second)
	predictedPosition := app.state.Players[1].Position

	app.applyAuthoritativeSnapshot(authoritative)

	if app.state.Tick != 2 {
		t.Fatalf("state tick = %d, want 2", app.state.Tick)
	}
	if app.state.Players[1].Position != predictedPosition {
		t.Fatalf("player two position = %+v, want replayed %+v", app.state.Players[1].Position, predictedPosition)
	}
}

func TestApplyAuthoritativeSnapshotStartsCorrectionAnimation(t *testing.T) {
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	app := localApp{
		cfg:        config.Default(),
		state:      state,
		player:     game.PlayerTwo,
		prediction: game.NewPredictionHistory(8),
	}

	command := game.NewInputCommand(app.state.Tick, game.PlayerTwo)
	command.MoveX = -1
	command.Aim = game.Left
	command.HasAim = true
	app.applyPredictedCommand(command)
	predictedPosition := app.state.Players[1].Position

	authoritative := state.Snapshot()
	authoritative.Tick = app.state.Tick
	app.applyAuthoritativeSnapshot(authoritative)

	if !app.correction.Active {
		t.Fatal("correction active = false, want true")
	}
	if app.correction.From != predictedPosition {
		t.Fatalf("correction from = %+v, want predicted %+v", app.correction.From, predictedPosition)
	}
	if app.correction.To != app.state.Players[1].Position {
		t.Fatalf("correction to = %+v, want authoritative %+v", app.correction.To, app.state.Players[1].Position)
	}
}
