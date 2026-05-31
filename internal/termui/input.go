package termui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/nishchay-veer/gladiator/internal/config"
	"github.com/nishchay-veer/gladiator/internal/game"
)

type inputAction int

const (
	actionNone inputAction = iota
	actionQuit
	actionMoveUp
	actionMoveDown
	actionMoveLeft
	actionMoveRight
	actionFire
	actionToggleNetDebug
	actionResize
)

func (a *localApp) handleEvent(event tcell.Event) bool {
	switch a.actionForEvent(event) {
	case actionQuit:
		return true
	case actionResize:
		a.screen.Sync()
	case actionMoveUp:
		a.queueMove(0, -1)
	case actionMoveDown:
		a.queueMove(0, 1)
	case actionMoveLeft:
		a.queueMove(-1, 0)
	case actionMoveRight:
		a.queueMove(1, 0)
	case actionFire:
		a.pending.Buttons |= game.ButtonFire
	case actionToggleNetDebug:
		a.showNetDebug = !a.showNetDebug
	}

	return false
}

func (a *localApp) queueMove(dx, dy int) {
	player := a.inputPlayer()
	a.pending.PlayerID = a.player
	a.pending.MoveX = dx
	a.pending.MoveY = dy
	a.pending.Aim = directionForMove(dx, dy, player.Facing)
	a.pending.HasAim = true
}

func (a *localApp) consumePendingCommand() game.InputCommand {
	command := a.pending
	command.Tick = a.state.Tick
	command.PlayerID = a.player
	a.pending = game.NewInputCommand(a.state.Tick+1, a.player)
	return command
}

func (a *localApp) sendPendingCommand(inputs chan<- game.InputCommand) (game.InputCommand, bool) {
	command := a.consumePendingCommand()
	select {
	case inputs <- command:
		return command, true
	default:
		return command, false
	}
}

func (a *localApp) inputPlayer() game.Player {
	switch a.player {
	case game.PlayerTwo:
		return a.state.Players[1]
	default:
		return a.state.Players[0]
	}
}

func (a *localApp) actionForEvent(event tcell.Event) inputAction {
	switch ev := event.(type) {
	case *tcell.EventResize:
		return actionResize
	case *tcell.EventKey:
		return a.actionForKey(ev)
	default:
		return actionNone
	}
}

func (a *localApp) actionForKey(event *tcell.EventKey) inputAction {
	switch event.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return actionQuit
	case tcell.KeyUp:
		return actionMoveUp
	case tcell.KeyDown:
		return actionMoveDown
	case tcell.KeyLeft:
		return actionMoveLeft
	case tcell.KeyRight:
		return actionMoveRight
	case tcell.KeyEnter:
		return actionFire
	case tcell.KeyRune:
		return actionForRune(event.Rune(), a.cfg.Controls)
	default:
		return actionNone
	}
}

func actionForRune(r rune, controls config.Controls) inputAction {
	switch {
	case r == 'n' || r == 'N':
		return actionToggleNetDebug
	case runeIn(r, controls.Quit):
		return actionQuit
	case runeIn(r, controls.Up):
		return actionMoveUp
	case runeIn(r, controls.Down):
		return actionMoveDown
	case runeIn(r, controls.Left):
		return actionMoveLeft
	case runeIn(r, controls.Right):
		return actionMoveRight
	case runeIn(r, controls.Fire):
		return actionFire
	default:
		return actionNone
	}
}

func runeIn(target rune, runes []rune) bool {
	for _, candidate := range runes {
		if candidate == target {
			return true
		}
	}
	return false
}

func directionForMove(dx, dy int, fallback game.Direction) game.Direction {
	switch {
	case dx > 0:
		return game.Right
	case dx < 0:
		return game.Left
	case dy > 0:
		return game.Down
	case dy < 0:
		return game.Up
	default:
		return fallback
	}
}
