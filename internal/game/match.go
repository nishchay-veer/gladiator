package game

type Match struct {
	MapID          string
	Mode           string
	TimeLimitTicks uint64
	ScoreLimit     int
	ElapsedTicks   uint64
	Over           bool
	Winner         PlayerID
	HasWinner      bool
}

func newLocalMatch() Match {
	return Match{
		MapID:          "local-arena-01",
		Mode:           "duel",
		TimeLimitTicks: openEndedTimeLimit,
		ScoreLimit:     openEndedScoreLimit,
	}
}

func (m *Match) advance(players [2]Player) {
	if m.Over {
		return
	}

	m.ElapsedTicks++
	if m.ScoreLimit > 0 {
		for _, player := range players {
			if player.Score >= m.ScoreLimit {
				m.Over = true
				m.Winner = player.ID
				m.HasWinner = true
				return
			}
		}
	}

	if m.TimeLimitTicks > 0 && m.ElapsedTicks >= m.TimeLimitTicks {
		m.Over = true
		switch {
		case players[0].Score > players[1].Score:
			m.Winner = players[0].ID
			m.HasWinner = true
		case players[1].Score > players[0].Score:
			m.Winner = players[1].ID
			m.HasWinner = true
		default:
			m.HasWinner = false
		}
	}
}
