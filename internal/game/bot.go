package game

func (s *State) BotCommand(botID, targetID PlayerID, tick uint64) InputCommand {
	command := NewInputCommand(tick, botID)
	bot := s.Player(botID)
	target := s.Player(targetID)
	if bot == nil || target == nil || !bot.Alive() || !target.Alive() {
		return command
	}

	if direction, ok := s.Arena.ClearLine(bot.Position, target.Position); ok {
		command.Aim = direction
		command.HasAim = true
		command.Buttons |= ButtonFire
		return command
	}

	if tick%botMoveEveryTicks != 0 {
		return command
	}

	for _, move := range chaseMoves(bot.Position, target.Position) {
		next := Point{X: bot.Position.X + move.X, Y: bot.Position.Y + move.Y}
		if !s.Arena.IsBlocked(next) && !s.occupiedByOtherPlayer(botID, next) {
			command.MoveX = move.X
			command.MoveY = move.Y
			command.Aim = directionFromDelta(move.X, move.Y, bot.Facing)
			command.HasAim = true
			return command
		}
	}

	return command
}

func chaseMoves(from, to Point) []Point {
	dx := sign(to.X - from.X)
	dy := sign(to.Y - from.Y)

	horizontal := Point{X: dx, Y: 0}
	vertical := Point{X: 0, Y: dy}

	if abs(to.X-from.X) >= abs(to.Y-from.Y) {
		return []Point{horizontal, vertical, {X: -dx, Y: 0}, {X: 0, Y: -dy}}
	}
	return []Point{vertical, horizontal, {X: 0, Y: -dy}, {X: -dx, Y: 0}}
}

func sign(value int) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}
