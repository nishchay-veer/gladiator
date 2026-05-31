package game

func (s State) ApplySnapshot(snapshot Snapshot) State {
	next := s
	next.Tick = snapshot.Tick
	next.Match = Match{
		MapID:          snapshot.Match.MapID,
		Mode:           snapshot.Match.Mode,
		TimeLimitTicks: snapshot.Match.TimeLimitTicks,
		ScoreLimit:     snapshot.Match.ScoreLimit,
		ElapsedTicks:   snapshot.Match.ElapsedTicks,
		Over:           snapshot.Match.Over,
		Winner:         snapshot.Match.Winner,
		HasWinner:      snapshot.Match.HasWinner,
	}

	for i, player := range snapshot.Players {
		next.Players[i] = Player{
			ID:           player.ID,
			Position:     player.Position,
			Spawn:        s.Players[i].Spawn,
			Facing:       player.Facing,
			Health:       player.Health,
			MaxHealth:    player.MaxHealth,
			Score:        player.Score,
			FireCooldown: player.FireCooldown,
			RespawnTicks: player.RespawnTicks,
		}
	}

	next.Bullets = make([]Bullet, len(snapshot.Bullets))
	for i, bullet := range snapshot.Bullets {
		next.Bullets[i] = Bullet{
			Position:  bullet.Position,
			Direction: bullet.Direction,
			Owner:     bullet.Owner,
			Age:       bullet.Age,
			TTL:       bullet.TTL,
		}
	}

	return next
}
