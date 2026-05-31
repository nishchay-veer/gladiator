package game

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
)

type Snapshot struct {
	Tick    uint64
	Match   MatchSnapshot
	Players [2]PlayerSnapshot
	Bullets []BulletSnapshot
}

type MatchSnapshot struct {
	MapID          string
	Mode           string
	TimeLimitTicks uint64
	ScoreLimit     int
	ElapsedTicks   uint64
	Over           bool
	Winner         PlayerID
	HasWinner      bool
}

type PlayerSnapshot struct {
	ID           PlayerID
	Position     Point
	Facing       Direction
	Health       int
	MaxHealth    int
	Score        int
	FireCooldown int
	RespawnTicks int
	Alive        bool
}

type BulletSnapshot struct {
	Position  Point
	Direction Direction
	Owner     PlayerID
	Age       int
	TTL       int
}

func (s State) Snapshot() Snapshot {
	snapshot := Snapshot{
		Tick:  s.Tick,
		Match: s.Match.Snapshot(),
		Players: [2]PlayerSnapshot{
			s.Players[0].Snapshot(),
			s.Players[1].Snapshot(),
		},
		Bullets: make([]BulletSnapshot, len(s.Bullets)),
	}

	for i, bullet := range s.Bullets {
		snapshot.Bullets[i] = BulletSnapshot{
			Position:  bullet.Position,
			Direction: bullet.Direction,
			Owner:     bullet.Owner,
			Age:       bullet.Age,
			TTL:       bullet.TTL,
		}
	}

	return snapshot
}

func (m Match) Snapshot() MatchSnapshot {
	return MatchSnapshot{
		MapID:          m.MapID,
		Mode:           m.Mode,
		TimeLimitTicks: m.TimeLimitTicks,
		ScoreLimit:     m.ScoreLimit,
		ElapsedTicks:   m.ElapsedTicks,
		Over:           m.Over,
		Winner:         m.Winner,
		HasWinner:      m.HasWinner,
	}
}

func (p Player) Snapshot() PlayerSnapshot {
	return PlayerSnapshot{
		ID:           p.ID,
		Position:     p.Position,
		Facing:       p.Facing,
		Health:       p.Health,
		MaxHealth:    p.MaxHealth,
		Score:        p.Score,
		FireCooldown: p.FireCooldown,
		RespawnTicks: p.RespawnTicks,
		Alive:        p.Alive(),
	}
}

func (s Snapshot) Equal(other Snapshot) bool {
	if s.Tick != other.Tick || s.Match != other.Match {
		return false
	}
	if s.Players != other.Players {
		return false
	}
	if len(s.Bullets) != len(other.Bullets) {
		return false
	}
	for i := range s.Bullets {
		if s.Bullets[i] != other.Bullets[i] {
			return false
		}
	}
	return true
}

func (s Snapshot) Hash() uint64 {
	h := fnv.New64a()

	writeUint64(h, s.Tick)
	writeString(h, s.Match.MapID)
	writeString(h, s.Match.Mode)
	writeUint64(h, s.Match.TimeLimitTicks)
	writeInt(h, s.Match.ScoreLimit)
	writeUint64(h, s.Match.ElapsedTicks)
	writeBool(h, s.Match.Over)
	writeInt(h, int(s.Match.Winner))
	writeBool(h, s.Match.HasWinner)

	for _, player := range s.Players {
		writeInt(h, int(player.ID))
		writePoint(h, player.Position)
		writeInt(h, int(player.Facing))
		writeInt(h, player.Health)
		writeInt(h, player.MaxHealth)
		writeInt(h, player.Score)
		writeInt(h, player.FireCooldown)
		writeInt(h, player.RespawnTicks)
		writeBool(h, player.Alive)
	}

	writeInt(h, len(s.Bullets))
	for _, bullet := range s.Bullets {
		writePoint(h, bullet.Position)
		writeInt(h, int(bullet.Direction))
		writeInt(h, int(bullet.Owner))
		writeInt(h, bullet.Age)
		writeInt(h, bullet.TTL)
	}

	return h.Sum64()
}

func writePoint(h hash.Hash64, point Point) {
	writeInt(h, point.X)
	writeInt(h, point.Y)
}

func writeString(h hash.Hash64, value string) {
	writeInt(h, len(value))
	_, _ = h.Write([]byte(value))
}

func writeBool(h hash.Hash64, value bool) {
	if value {
		writeUint64(h, 1)
		return
	}
	writeUint64(h, 0)
}

func writeInt(h hash.Hash64, value int) {
	writeUint64(h, uint64(int64(value)))
}

func writeUint64(h hash.Hash64, value uint64) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], value)
	_, _ = h.Write(buffer[:])
}
