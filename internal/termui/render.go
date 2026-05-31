package termui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"gladiator/internal/game"
)

func (a *localApp) draw() {
	a.updateFPS()
	a.screen.Clear()

	width, height := a.screen.Size()
	if width < a.cfg.MinimumTermWidth || height < a.cfg.MinimumTermRows {
		a.drawTooSmall(width, height)
		a.screen.Show()
		return
	}

	arenaCols := a.state.Arena.Width * 2
	originX := (width - arenaCols) / 2
	if originX < 0 {
		originX = 0
	}
	originY := 1

	a.drawArena(originX, originY)
	a.drawHUD(originX, originY+a.state.Arena.Height)
	a.screen.Show()
}

func (a *localApp) drawTooSmall(width, height int) {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	lines := []string{
		"Gladiator needs a larger terminal.",
		fmt.Sprintf("Current: %dx%d", width, height),
		fmt.Sprintf("Minimum: %dx%d", a.cfg.MinimumTermWidth, a.cfg.MinimumTermRows),
		"Resize or press q/Esc/Ctrl-C to quit.",
	}
	for y, line := range lines {
		drawText(a.screen, 2, 2+y, style, line)
	}
}

func (a *localApp) drawArena(originX, originY int) {
	styles := arenaStyles()

	for y := 0; y < a.state.Arena.Height; y++ {
		for x := 0; x < a.state.Arena.Width; x++ {
			p := game.Point{X: x, Y: y}
			cellX := originX + x*2
			cellY := originY + y

			if a.drawPlayerAt(p, cellX, cellY, styles) {
				continue
			}
			if a.drawWallAt(p, cellX, cellY, styles) {
				continue
			}
			if a.drawBulletAt(p, cellX, cellY, styles) {
				continue
			}
			drawFloor(a.screen, cellX, cellY, x, y, styles)
		}
	}
}

func (a *localApp) drawPlayerAt(p game.Point, x, y int, styles renderStyles) bool {
	p1 := a.state.Players[0]
	if p1.Alive() && a.correction.PositionFor(p1) == p {
		drawTile(a.screen, x, y, styles.player1, playerRune(p1.Facing), ' ')
		return true
	}

	p2 := a.state.Players[1]
	if a.showPlayer2 && p2.Alive() && a.correction.PositionFor(p2) == p {
		drawTile(a.screen, x, y, styles.player2, playerRune(p2.Facing), ' ')
		return true
	}

	return false
}

func (a *localApp) drawWallAt(p game.Point, x, y int, styles renderStyles) bool {
	if a.state.Arena.TileAt(p) != game.Wall {
		return false
	}

	if p.X == 0 || p.Y == 0 || p.X == a.state.Arena.Width-1 || p.Y == a.state.Arena.Height-1 {
		drawTile(a.screen, x, y, styles.wallEdge, '█', '█')
		return true
	}

	drawTile(a.screen, x, y, styles.wall, '█', '█')
	return true
}

func (a *localApp) drawBulletAt(p game.Point, x, y int, styles renderStyles) bool {
	for _, shot := range a.state.Bullets {
		if shot.Position == p {
			drawTile(a.screen, x, y, styles.bullet, bulletRune(shot.Direction), ' ')
			return true
		}
	}
	return false
}

func (a *localApp) drawHUD(x, y int) {
	styles := hudStyles()
	p1 := a.state.Players[0]
	p2 := a.state.Players[1]
	p2Status := "WAIT"
	p2Score := 0
	if a.showPlayer2 {
		p2Status = playerStatus(p2, a.cfg.SimulationRate)
		p2Score = p2.Score
	}
	line := fmt.Sprintf(" P1 %s  CD %s  S%d | P2 %s  S%d | FPS %03d ",
		playerStatus(p1, a.cfg.SimulationRate), cooldownText(p1), p1.Score,
		p2Status, p2Score, a.fps)
	if a.status != "" {
		line += "| " + a.status + " "
	}

	drawText(a.screen, x, y, styles.primary, line)
	drawText(a.screen, x, y+1, styles.help, " WASD/Arrows move   Space/Enter shoot   q/Esc quits ")
	if a.showNetDebug {
		if line := netDebugLine(a.netDebug); line != "" {
			drawTextClipped(a.screen, x, y+2, styles.debug, line, a.state.Arena.Width*2)
		}
	}
}

func drawFloor(screen tcell.Screen, x, y, arenaX, arenaY int, styles renderStyles) {
	if (arenaX+arenaY)%7 == 0 {
		drawTile(screen, x, y, styles.floorB, '·', ' ')
		return
	}
	drawTile(screen, x, y, styles.floorA, ' ', ' ')
}

func drawTile(screen tcell.Screen, x, y int, style tcell.Style, left, right rune) {
	screen.SetContent(x, y, left, nil, style)
	screen.SetContent(x+1, y, right, nil, style)
}

func drawText(screen tcell.Screen, x, y int, style tcell.Style, text string) {
	col := 0
	for _, r := range text {
		screen.SetContent(x+col, y, r, nil, style)
		col++
	}
}

func drawTextClipped(screen tcell.Screen, x, y int, style tcell.Style, text string, maxCells int) {
	if maxCells <= 0 {
		return
	}

	col := 0
	for _, r := range text {
		if col >= maxCells {
			return
		}
		screen.SetContent(x+col, y, r, nil, style)
		col++
	}
}

func playerRune(direction game.Direction) rune {
	switch direction {
	case game.Up:
		return '▲'
	case game.Down:
		return '▼'
	case game.Left:
		return '◀'
	default:
		return '▶'
	}
}

func bulletRune(direction game.Direction) rune {
	return '•'
}

func playerStatus(player game.Player, tickRate time.Duration) string {
	if player.Alive() {
		return healthText(player)
	}

	seconds := float64(player.RespawnTicks) * tickRate.Seconds()
	return fmt.Sprintf("DOWN %.1fs", seconds)
}

func healthText(player game.Player) string {
	text := ""
	for i := 0; i < player.MaxHealth; i++ {
		if i < player.Health {
			text += "♥"
		} else {
			text += "·"
		}
	}
	return text
}

func cooldownText(player game.Player) string {
	if !player.Alive() {
		return "--"
	}
	if player.FireCooldown == 0 {
		return "OK"
	}
	return fmt.Sprintf("%02d", player.FireCooldown)
}

func (a *localApp) updateFPS() {
	now := time.Now()
	if a.fpsWindow.IsZero() {
		a.fpsWindow = now
	}

	a.frames++
	if now.Sub(a.fpsWindow) < time.Second {
		return
	}

	a.fps = a.frames
	a.frames = 0
	a.fpsWindow = now
}
