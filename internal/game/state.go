package game

import "fmt"

type State struct {
	Match   Match
	Arena   Arena
	Players [2]Player
	Bullets []Bullet
	Tick    uint64
}

func NewLocalState() (State, error) {
	arena, err := NewArena(localLayout)
	if err != nil {
		return State{}, err
	}

	opponentSpawn := Point{X: 35, Y: 14}
	if arena.IsBlocked(opponentSpawn) {
		return State{}, fmt.Errorf("opponent spawn is blocked at %d,%d", opponentSpawn.X, opponentSpawn.Y)
	}

	return State{
		Match: newLocalMatch(),
		Arena: arena,
		Players: [2]Player{
			newPlayer(PlayerOne, arena.Spawn, Right),
			newPlayer(PlayerTwo, opponentSpawn, Left),
		},
	}, nil
}

func (s *State) Player(id PlayerID) *Player {
	switch id {
	case PlayerOne:
		return &s.Players[0]
	case PlayerTwo:
		return &s.Players[1]
	default:
		return nil
	}
}

func (s *State) movePlayer(id PlayerID, dx, dy int) bool {
	if dx == 0 && dy == 0 {
		return false
	}

	player := s.Player(id)
	if player == nil || !player.Alive() {
		return false
	}

	player.Facing = directionFromDelta(dx, dy, player.Facing)
	next := Point{X: player.Position.X + dx, Y: player.Position.Y + dy}
	if s.Arena.IsBlocked(next) || s.occupiedByOtherPlayer(id, next) {
		return false
	}

	player.Position = next
	return true
}

func (s *State) fire(id PlayerID) bool {
	player := s.Player(id)
	if player == nil || !player.CanFire() {
		return false
	}

	player.FireCooldown = fireCooldownTicks
	spawn := stepPoint(player.Position, player.Facing)
	if s.Arena.IsBlocked(spawn) {
		return true
	}

	if victim := s.playerAt(spawn, id); victim != nil {
		s.damagePlayer(victim.ID, id, bulletDamage)
		return true
	}

	s.Bullets = append(s.Bullets, Bullet{
		Position:  spawn,
		Direction: player.Facing,
		Owner:     id,
		TTL:       bulletTimeToLive,
	})
	return true
}

func (s State) Clone() State {
	clone := s
	clone.Bullets = append([]Bullet(nil), s.Bullets...)
	return clone
}

func (s State) Next(frame InputFrame) State {
	next := s.Clone()
	next.Step(frame)
	return next
}

func (s *State) Step(frame InputFrame) {
	if s.Match.Over {
		s.Tick++
		return
	}

	for _, command := range frame.normalized(s.Tick).Commands {
		s.applyCommand(command)
	}

	s.advancePlayers()
	s.advanceBullets()
	s.Match.advance(s.Players)
	s.Tick++
}

func (s *State) applyCommand(command InputCommand) {
	player := s.Player(command.PlayerID)
	if player == nil || !player.Alive() {
		return
	}

	if command.HasAim {
		player.Facing = command.Aim
	}
	if command.MoveX != 0 || command.MoveY != 0 {
		s.movePlayer(command.PlayerID, command.MoveX, command.MoveY)
	}
	if command.WantsFire() {
		s.fire(command.PlayerID)
	}
}

func (s *State) advancePlayers() {
	for i := range s.Players {
		player := &s.Players[i]
		if player.FireCooldown > 0 {
			player.FireCooldown--
		}
		if player.RespawnTicks > 0 {
			player.RespawnTicks--
			if player.RespawnTicks == 0 {
				player.Health = player.MaxHealth
				player.Position = player.Spawn
			}
		}
	}
}

func (s *State) advanceBullets() {
	active := s.Bullets[:0]
	for _, bullet := range s.Bullets {
		bullet.Age++
		bullet.TTL--
		if bullet.TTL <= 0 {
			continue
		}

		if bullet.Age%bulletMoveEvery == 0 {
			next := stepPoint(bullet.Position, bullet.Direction)
			if s.Arena.IsBlocked(next) {
				continue
			}
			if victim := s.playerAt(next, bullet.Owner); victim != nil {
				s.damagePlayer(victim.ID, bullet.Owner, bulletDamage)
				continue
			}
			bullet.Position = next
		}

		active = append(active, bullet)
	}
	s.Bullets = active
}

func (s *State) damagePlayer(victimID, attackerID PlayerID, damage int) {
	victim := s.Player(victimID)
	if victim == nil || !victim.Alive() {
		return
	}

	victim.Health -= damage
	if victim.Health > 0 {
		return
	}

	victim.Health = 0
	victim.RespawnTicks = respawnDelayTicks
	victim.Position = victim.Spawn

	attacker := s.Player(attackerID)
	if attacker != nil && attacker.ID != victim.ID {
		attacker.Score++
	}
}

func (s *State) playerAt(position Point, except PlayerID) *Player {
	for i := range s.Players {
		player := &s.Players[i]
		if player.ID == except || !player.Alive() {
			continue
		}
		if player.Position == position {
			return player
		}
	}
	return nil
}

func (s *State) occupiedByOtherPlayer(mover PlayerID, position Point) bool {
	for i := range s.Players {
		player := &s.Players[i]
		if player.ID == mover || !player.Alive() {
			continue
		}
		if player.Position == position {
			return true
		}
	}
	return false
}
