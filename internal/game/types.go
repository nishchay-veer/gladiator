package game

type Tile byte

const (
	Floor Tile = '.'
	Wall  Tile = '#'
)

type Direction int

const (
	Right Direction = iota
	Down
	Left
	Up
)

type PlayerID int

const (
	PlayerOne PlayerID = iota
	PlayerTwo
)

const (
	maxHealth           = 4
	fireCooldownTicks   = 20
	respawnDelayTicks   = 90
	bulletDamage        = 1
	bulletMoveEvery     = 2
	bulletTimeToLive    = 90
	botMoveEveryTicks   = 18
	botFireEveryTicks   = 45
	botFireWindupTicks  = 15
	openEndedTimeLimit  = 0
	openEndedScoreLimit = 0
	defaultArenaWidth   = 40
	defaultArenaHeight  = 17
)

type Point struct {
	X int
	Y int
}

func directionFromDelta(dx, dy int, fallback Direction) Direction {
	switch {
	case dx > 0:
		return Right
	case dx < 0:
		return Left
	case dy > 0:
		return Down
	case dy < 0:
		return Up
	default:
		return fallback
	}
}

func stepPoint(point Point, direction Direction) Point {
	switch direction {
	case Up:
		return Point{X: point.X, Y: point.Y - 1}
	case Down:
		return Point{X: point.X, Y: point.Y + 1}
	case Left:
		return Point{X: point.X - 1, Y: point.Y}
	default:
		return Point{X: point.X + 1, Y: point.Y}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
