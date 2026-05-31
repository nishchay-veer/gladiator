package game

type Bullet struct {
	Position  Point
	Direction Direction
	Owner     PlayerID
	Age       int
	TTL       int
}
