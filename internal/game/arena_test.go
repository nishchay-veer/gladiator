package game

import "testing"

func TestNewLocalState(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	if state.Arena.Width != defaultArenaWidth {
		t.Fatalf("arena width = %d, want %d", state.Arena.Width, defaultArenaWidth)
	}
	if state.Arena.Height != defaultArenaHeight {
		t.Fatalf("arena height = %d, want %d", state.Arena.Height, defaultArenaHeight)
	}
	if !state.Players[0].Alive() {
		t.Fatal("player one should start alive")
	}
	if !state.Players[1].Alive() {
		t.Fatal("player two should start alive")
	}
	if state.Arena.IsBlocked(state.Players[0].Position) {
		t.Fatalf("player one spawn is blocked: %+v", state.Players[0].Position)
	}
	if state.Arena.IsBlocked(state.Players[1].Position) {
		t.Fatalf("player two spawn is blocked: %+v", state.Players[1].Position)
	}
	if state.Players[0].Position == state.Players[1].Position {
		t.Fatalf("players overlap at %+v", state.Players[0].Position)
	}
}

func TestStepMovesPlayer(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, MoveX: 1})
	if state.Players[0].Position != (Point{X: 2, Y: 1}) {
		t.Fatalf("player one = %+v, want {X:2 Y:1}", state.Players[0].Position)
	}

	state.Players[0].Position = Point{X: 1, Y: 1}
	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, MoveX: -1})
	if state.Players[0].Position != (Point{X: 1, Y: 1}) {
		t.Fatalf("blocked move changed player to %+v", state.Players[0].Position)
	}
}

func TestStepCannotMoveIntoOtherPlayer(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[1].Position = Point{X: 6, Y: 5}

	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, MoveX: 1})
	if state.Players[0].Position != (Point{X: 5, Y: 5}) {
		t.Fatalf("blocked overlap changed player to %+v", state.Players[0].Position)
	}
}

func TestFireCreatesBulletAndCooldown(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, Buttons: ButtonFire})
	if len(state.Bullets) != 1 {
		t.Fatalf("len(Bullets) = %d, want 1", len(state.Bullets))
	}
	if state.Players[0].FireCooldown != fireCooldownTicks-1 {
		t.Fatalf("cooldown = %d, want %d", state.Players[0].FireCooldown, fireCooldownTicks-1)
	}
	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, Buttons: ButtonFire})
	if len(state.Bullets) != 1 {
		t.Fatalf("len(Bullets) = %d, want 1 while fire is on cooldown", len(state.Bullets))
	}
}

func TestBulletDamagesPlayer(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[0].Facing = Right
	state.Players[1].Position = Point{X: 8, Y: 5}

	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, Buttons: ButtonFire})
	for i := 0; i < bulletMoveEvery+1; i++ {
		state.Step(NewInputFrame(state.Tick))
	}

	if got, want := state.Players[1].Health, maxHealth-bulletDamage; got != want {
		t.Fatalf("player two health = %d, want %d", got, want)
	}
	if len(state.Bullets) != 0 {
		t.Fatalf("len(Bullets) = %d, want 0 after hit", len(state.Bullets))
	}
}

func TestKillScoresAndRespawns(t *testing.T) {
	state, err := NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	state.Players[0].Position = Point{X: 5, Y: 5}
	state.Players[0].Facing = Right
	state.Players[1].Position = Point{X: 6, Y: 5}
	state.Players[1].Health = 1

	stepWithCommand(&state, InputCommand{PlayerID: PlayerOne, Buttons: ButtonFire})
	if state.Players[0].Score != 1 {
		t.Fatalf("player one score = %d, want 1", state.Players[0].Score)
	}
	if state.Players[1].Alive() {
		t.Fatal("player two should be down after lethal damage")
	}
	if state.Players[1].RespawnTicks != respawnDelayTicks-1 {
		t.Fatalf("respawn ticks = %d, want %d", state.Players[1].RespawnTicks, respawnDelayTicks-1)
	}

	for i := 0; i < respawnDelayTicks-1; i++ {
		state.Step(NewInputFrame(state.Tick))
	}

	if !state.Players[1].Alive() {
		t.Fatal("player two should respawn after respawn delay")
	}
	if state.Players[1].Health != maxHealth {
		t.Fatalf("player two health = %d, want %d", state.Players[1].Health, maxHealth)
	}
}

func stepWithCommand(state *State, command InputCommand) {
	frame := NewInputFrame(state.Tick)
	command.Tick = state.Tick
	frame.Set(command)
	state.Step(frame)
}

func TestNewArenaValidation(t *testing.T) {
	tests := []struct {
		name string
		rows []string
	}{
		{name: "empty", rows: nil},
		{name: "ragged", rows: []string{"###", "#P#", "##"}},
		{name: "missing spawn", rows: []string{"###", "#.#", "###"}},
		{name: "duplicate spawn", rows: []string{"####", "#PP#", "####"}},
		{name: "unknown tile", rows: []string{"###", "#@#", "###"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewArena(tt.rows); err == nil {
				t.Fatal("NewArena() error = nil, want validation error")
			}
		})
	}
}
