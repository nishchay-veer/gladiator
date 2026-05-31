package game

import "testing"

func TestPredictionHistoryPrunesOldEntries(t *testing.T) {
	history := NewPredictionHistory(2)

	for tick := uint64(0); tick < 3; tick++ {
		snapshot := Snapshot{Tick: tick + 1}
		history.Record(NewInputCommand(tick, PlayerOne), snapshot)
	}

	if got := history.Len(); got != 2 {
		t.Fatalf("history length = %d, want 2", got)
	}
	if _, ok := history.SnapshotAt(1); ok {
		t.Fatal("old snapshot was retained after pruning")
	}
	if _, ok := history.SnapshotAt(3); !ok {
		t.Fatal("latest snapshot was not retained")
	}
}

func TestPredictionReconcileReplaysLocalCommands(t *testing.T) {
	base, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	history := NewPredictionHistory(8)
	predicted := base.Clone()
	var authoritative Snapshot
	for i := 0; i < 3; i++ {
		command := NewInputCommand(predicted.Tick, PlayerOne)
		command.MoveX = 1
		command.Aim = Right
		command.HasAim = true

		frame := NewInputFrame(predicted.Tick)
		frame.Set(command)
		predicted.Step(frame)
		history.Record(command, predicted.Snapshot())

		if i == 0 {
			authoritative = predicted.Snapshot()
		}
	}

	result := history.Reconcile(base, authoritative, PlayerOne)
	if result.TargetTick != predicted.Tick {
		t.Fatalf("target tick = %d, want %d", result.TargetTick, predicted.Tick)
	}
	if result.State.Tick != predicted.Tick {
		t.Fatalf("reconciled tick = %d, want %d", result.State.Tick, predicted.Tick)
	}
	if result.ReplayedCommands != 2 {
		t.Fatalf("replayed commands = %d, want 2", result.ReplayedCommands)
	}
	if result.State.Players[0].Position != predicted.Players[0].Position {
		t.Fatalf("player position = %+v, want %+v", result.State.Players[0].Position, predicted.Players[0].Position)
	}
	if result.NeedsCorrection {
		t.Fatal("needs correction = true, want false")
	}
}

func TestPredictionReconcileDetectsLocalPlayerCorrection(t *testing.T) {
	base, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	history := NewPredictionHistory(4)
	predicted := base.Clone()
	command := NewInputCommand(predicted.Tick, PlayerOne)
	command.MoveX = 1
	command.Aim = Right
	command.HasAim = true
	frame := NewInputFrame(predicted.Tick)
	frame.Set(command)
	predicted.Step(frame)
	history.Record(command, predicted.Snapshot())

	authoritative := base.Snapshot()
	authoritative.Tick = predicted.Tick

	result := history.Reconcile(base, authoritative, PlayerOne)
	if !result.HadPrediction {
		t.Fatal("had prediction = false, want true")
	}
	if !result.NeedsCorrection {
		t.Fatal("needs correction = false, want true")
	}
	if result.PredictedPlayer.Position == result.AuthoritativePlayer.Position {
		t.Fatal("predicted and authoritative positions unexpectedly matched")
	}
}

func TestPredictionReconcileIgnoresRemoteOnlyMismatch(t *testing.T) {
	base, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	history := NewPredictionHistory(4)
	predicted := base.Clone()
	predicted.Players[1].Position.X--
	history.Record(NewInputCommand(0, PlayerOne), predicted.Snapshot())

	authoritative := predicted.Snapshot()
	authoritative.Players[1].Position.X--

	result := history.Reconcile(base, authoritative, PlayerOne)
	if result.NeedsCorrection {
		t.Fatal("needs correction = true for remote-only mismatch, want false")
	}
}
