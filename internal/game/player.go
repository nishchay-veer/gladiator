package game

type Player struct {
	ID           PlayerID
	Position     Point
	Spawn        Point
	Facing       Direction
	Health       int
	MaxHealth    int
	Score        int
	FireCooldown int
	RespawnTicks int
}

func (p Player) Alive() bool {
	return p.Health > 0 && p.RespawnTicks == 0
}

func (p Player) CanFire() bool {
	return p.Alive() && p.FireCooldown == 0
}

func newPlayer(id PlayerID, spawn Point, facing Direction) Player {
	return Player{
		ID:        id,
		Position:  spawn,
		Spawn:     spawn,
		Facing:    facing,
		Health:    maxHealth,
		MaxHealth: maxHealth,
	}
}
