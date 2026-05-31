package game

import "fmt"

type Buttons uint8

const (
	ButtonFire Buttons = 1 << iota
)

const knownButtons = ButtonFire

type InputCommand struct {
	Tick     uint64
	PlayerID PlayerID
	Sequence uint32
	MoveX    int
	MoveY    int
	Aim      Direction
	HasAim   bool
	Buttons  Buttons
}

type InputFrame struct {
	Tick     uint64
	Commands [2]InputCommand
}

func NewInputCommand(tick uint64, playerID PlayerID) InputCommand {
	return InputCommand{
		Tick:     tick,
		PlayerID: playerID,
	}
}

func NewInputFrame(tick uint64) InputFrame {
	return InputFrame{
		Tick: tick,
		Commands: [2]InputCommand{
			NewInputCommand(tick, PlayerOne),
			NewInputCommand(tick, PlayerTwo),
		},
	}
}

func (f *InputFrame) Set(command InputCommand) {
	switch command.PlayerID {
	case PlayerOne:
		f.Commands[0] = command.normalized(f.Tick)
	case PlayerTwo:
		f.Commands[1] = command.normalized(f.Tick)
	}
}

func (f InputFrame) Validate(expectedTick uint64) error {
	if f.Tick != expectedTick {
		return fmt.Errorf("input frame tick %d does not match expected tick %d", f.Tick, expectedTick)
	}

	for index, command := range f.Commands {
		expectedPlayer := PlayerID(index)
		if command.PlayerID != expectedPlayer {
			return fmt.Errorf("input command slot %d belongs to player %d", index, command.PlayerID)
		}
		if err := command.Validate(expectedTick); err != nil {
			return err
		}
	}

	return nil
}

func (f InputFrame) normalized(tick uint64) InputFrame {
	normalized := NewInputFrame(tick)
	normalized.Set(f.Commands[0])
	normalized.Set(f.Commands[1])
	return normalized
}

func (c InputCommand) Validate(expectedTick uint64) error {
	if c.Tick != expectedTick {
		return fmt.Errorf("input command tick %d does not match expected tick %d", c.Tick, expectedTick)
	}
	if !validPlayerID(c.PlayerID) {
		return fmt.Errorf("invalid player id %d", c.PlayerID)
	}
	if c.MoveX < -1 || c.MoveX > 1 || c.MoveY < -1 || c.MoveY > 1 {
		return fmt.Errorf("movement axis out of range: %d,%d", c.MoveX, c.MoveY)
	}
	if c.MoveX != 0 && c.MoveY != 0 {
		return fmt.Errorf("diagonal movement is not allowed: %d,%d", c.MoveX, c.MoveY)
	}
	if c.HasAim && !validDirection(c.Aim) {
		return fmt.Errorf("invalid aim direction %d", c.Aim)
	}
	if c.Buttons&^knownButtons != 0 {
		return fmt.Errorf("unknown button bits set: %08b", c.Buttons)
	}

	return nil
}

func (c InputCommand) WantsFire() bool {
	return c.Buttons&ButtonFire != 0
}

func (c InputCommand) normalized(tick uint64) InputCommand {
	c.Tick = tick
	c.MoveX = clampAxis(c.MoveX)
	c.MoveY = clampAxis(c.MoveY)
	c.Buttons &= knownButtons

	if c.MoveX != 0 {
		c.MoveY = 0
	}
	if !validDirection(c.Aim) {
		c.HasAim = false
		c.Aim = Right
	}

	return c
}

func clampAxis(axis int) int {
	switch {
	case axis > 0:
		return 1
	case axis < 0:
		return -1
	default:
		return 0
	}
}

func validDirection(direction Direction) bool {
	return direction >= Right && direction <= Up
}

func validPlayerID(id PlayerID) bool {
	return id == PlayerOne || id == PlayerTwo
}
