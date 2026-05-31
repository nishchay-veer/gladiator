package config

import "time"

type Config struct {
	PlayerName       string
	NetworkPort      int
	SimulationRate   time.Duration
	SnapshotRate     time.Duration
	RenderRate       time.Duration
	MinimumTermWidth int
	MinimumTermRows  int
	Controls         Controls
}

type Controls struct {
	Up    []rune
	Down  []rune
	Left  []rune
	Right []rune
	Fire  []rune
	Quit  []rune
}

func Default() Config {
	return Config{
		PlayerName:       "P1",
		NetworkPort:      42424,
		SimulationRate:   time.Second / 60,
		SnapshotRate:     time.Second / 30,
		RenderRate:       time.Second / 60,
		MinimumTermWidth: 80,
		MinimumTermRows:  24,
		Controls: Controls{
			Up:    []rune{'w', 'W', 'k', 'K'},
			Down:  []rune{'s', 'S', 'j', 'J'},
			Left:  []rune{'a', 'A', 'h', 'H'},
			Right: []rune{'d', 'D', 'l', 'L'},
			Fire:  []rune{' '},
			Quit:  []rune{'q', 'Q'},
		},
	}
}
