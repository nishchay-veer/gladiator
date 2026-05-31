package game

import "testing"

func TestInputCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		command InputCommand
		wantErr bool
	}{
		{
			name:    "valid move",
			command: InputCommand{Tick: 7, PlayerID: PlayerOne, MoveX: 1},
		},
		{
			name:    "valid fire",
			command: InputCommand{Tick: 7, PlayerID: PlayerTwo, Buttons: ButtonFire},
		},
		{
			name:    "wrong tick",
			command: InputCommand{Tick: 6, PlayerID: PlayerOne},
			wantErr: true,
		},
		{
			name:    "invalid player",
			command: InputCommand{Tick: 7, PlayerID: PlayerID(99)},
			wantErr: true,
		},
		{
			name:    "axis too large",
			command: InputCommand{Tick: 7, PlayerID: PlayerOne, MoveX: 2},
			wantErr: true,
		},
		{
			name:    "diagonal movement",
			command: InputCommand{Tick: 7, PlayerID: PlayerOne, MoveX: 1, MoveY: 1},
			wantErr: true,
		},
		{
			name:    "invalid aim",
			command: InputCommand{Tick: 7, PlayerID: PlayerOne, Aim: Direction(99), HasAim: true},
			wantErr: true,
		},
		{
			name:    "unknown button",
			command: InputCommand{Tick: 7, PlayerID: PlayerOne, Buttons: Buttons(1 << 7)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.command.Validate(7)
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestInputFrameValidation(t *testing.T) {
	frame := NewInputFrame(12)
	if err := frame.Validate(12); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	frame.Commands[1].PlayerID = PlayerOne
	if err := frame.Validate(12); err == nil {
		t.Fatal("Validate() error = nil, want slot/player mismatch")
	}
}

func TestInputFrameSetNormalizesCommand(t *testing.T) {
	frame := NewInputFrame(4)
	frame.Set(InputCommand{
		Tick:     99,
		PlayerID: PlayerOne,
		MoveX:    2,
		MoveY:    1,
		Buttons:  ButtonFire | Buttons(1<<7),
	})

	command := frame.Commands[0]
	if command.Tick != 4 {
		t.Fatalf("tick = %d, want 4", command.Tick)
	}
	if command.MoveX != 1 || command.MoveY != 0 {
		t.Fatalf("movement = %d,%d, want 1,0", command.MoveX, command.MoveY)
	}
	if command.Buttons != ButtonFire {
		t.Fatalf("buttons = %08b, want %08b", command.Buttons, ButtonFire)
	}
}
