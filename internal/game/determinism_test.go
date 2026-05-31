package game

import "testing"

func TestStepGoldenSnapshot(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[0].Facing = Right
	state.Players[1].Position = Point{X: 8, Y: 5}

	fire := NewInputCommand(state.Tick, PlayerOne)
	fire.Buttons = ButtonFire
	frame := NewInputFrame(state.Tick)
	frame.Set(fire)
	state.Step(frame)

	for i := 0; i < 3; i++ {
		state.Step(NewInputFrame(state.Tick))
	}

	got := state.Snapshot()
	want := Snapshot{
		Tick: 4,
		Match: MatchSnapshot{
			MapID:          "local-arena-01",
			Mode:           "duel",
			TimeLimitTicks: openEndedTimeLimit,
			ScoreLimit:     openEndedScoreLimit,
			ElapsedTicks:   4,
		},
		Players: [2]PlayerSnapshot{
			{
				ID:           PlayerOne,
				Position:     Point{X: 5, Y: 5},
				Facing:       Right,
				Health:       maxHealth,
				MaxHealth:    maxHealth,
				FireCooldown: fireCooldownTicks - 4,
				Alive:        true,
			},
			{
				ID:        PlayerTwo,
				Position:  Point{X: 8, Y: 5},
				Facing:    Left,
				Health:    maxHealth - bulletDamage,
				MaxHealth: maxHealth,
				Alive:     true,
			},
		},
		Bullets: []BulletSnapshot{},
	}

	if !got.Equal(want) {
		t.Fatalf("snapshot mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestSameInputLogProducesSameSnapshot(t *testing.T) {
	first := runInputLog(t)
	second := runInputLog(t)

	if !first.Equal(second) {
		t.Fatalf("same input log produced different snapshots\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if first.Hash() != second.Hash() {
		t.Fatalf("same input log produced different hashes: %d != %d", first.Hash(), second.Hash())
	}
}

func TestNextDoesNotMutateOriginalState(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	command := NewInputCommand(state.Tick, PlayerOne)
	command.MoveX = 1
	frame := NewInputFrame(state.Tick)
	frame.Set(command)

	next := state.Next(frame)
	if state.Tick != 0 {
		t.Fatalf("original tick = %d, want 0", state.Tick)
	}
	if state.Players[0].Position != state.Players[0].Spawn {
		t.Fatalf("original player position = %+v, want spawn %+v", state.Players[0].Position, state.Players[0].Spawn)
	}
	if next.Tick != 1 {
		t.Fatalf("next tick = %d, want 1", next.Tick)
	}
	if next.Players[0].Position == state.Players[0].Position {
		t.Fatal("next state did not apply movement command")
	}
}

func TestSnapshotDoesNotAliasState(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	command := NewInputCommand(state.Tick, PlayerOne)
	command.Buttons = ButtonFire
	frame := NewInputFrame(state.Tick)
	frame.Set(command)
	state.Step(frame)

	snapshot := state.Snapshot()
	if len(snapshot.Bullets) != 1 {
		t.Fatalf("len(snapshot.Bullets) = %d, want 1", len(snapshot.Bullets))
	}

	state.Bullets[0].Position = Point{X: 99, Y: 99}
	if snapshot.Bullets[0].Position == state.Bullets[0].Position {
		t.Fatal("snapshot bullet position changed after state mutation")
	}
}

func runInputLog(t *testing.T) Snapshot {
	t.Helper()

	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[0].Facing = Right
	state.Players[1].Position = Point{X: 9, Y: 5}

	for tick := 0; tick < 8; tick++ {
		frame := NewInputFrame(state.Tick)
		if tick == 0 {
			command := NewInputCommand(state.Tick, PlayerOne)
			command.Buttons = ButtonFire
			frame.Set(command)
		}
		if tick == 5 {
			command := NewInputCommand(state.Tick, PlayerOne)
			command.MoveY = 1
			command.Aim = Down
			command.HasAim = true
			frame.Set(command)
		}
		state.Step(frame)
	}

	return state.Snapshot()
}
