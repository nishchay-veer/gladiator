package termui

import (
	"testing"

	"github.com/nishchay-veer/gladiator/internal/config"
	"github.com/nishchay-veer/gladiator/internal/game"
	"github.com/nishchay-veer/gladiator/internal/netplay"
)

func TestActionForRune(t *testing.T) {
	controls := config.Default().Controls

	tests := []struct {
		name string
		key  rune
		want inputAction
	}{
		{name: "up", key: 'w', want: actionMoveUp},
		{name: "down", key: 's', want: actionMoveDown},
		{name: "left", key: 'a', want: actionMoveLeft},
		{name: "right", key: 'd', want: actionMoveRight},
		{name: "fire", key: ' ', want: actionFire},
		{name: "net debug", key: 'n', want: actionToggleNetDebug},
		{name: "rematch", key: 'r', want: actionRematch},
		{name: "quit", key: 'q', want: actionQuit},
		{name: "none", key: '?', want: actionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := actionForRune(tt.key, controls); got != tt.want {
				t.Fatalf("actionForRune(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestRequestRematchResetsLocalHostState(t *testing.T) {
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}
	state.Tick = 20
	state.Players[0].Score = 2

	app := localApp{
		cfg:     config.Default(),
		state:   state,
		player:  game.PlayerOne,
		pending: game.NewInputCommand(state.Tick, game.PlayerOne),
	}

	app.requestRematch()
	if app.state.Tick != 0 {
		t.Fatalf("tick = %d, want 0", app.state.Tick)
	}
	if app.state.Players[0].Score != 0 {
		t.Fatalf("p1 score = %d, want 0", app.state.Players[0].Score)
	}
	if app.pending.Tick != 0 {
		t.Fatalf("pending tick = %d, want 0", app.pending.Tick)
	}
}

func TestRequestRematchSendsHostSignal(t *testing.T) {
	rematches := make(chan struct{}, 1)
	app := localApp{
		player:    game.PlayerOne,
		rematches: rematches,
	}

	app.requestRematch()
	select {
	case <-rematches:
	default:
		t.Fatal("rematch signal not sent")
	}
}

func TestRequestRematchIgnoredForJoiner(t *testing.T) {
	rematches := make(chan struct{}, 1)
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}
	state.Players[0].Score = 2
	app := localApp{
		state:     state,
		player:    game.PlayerTwo,
		rematches: rematches,
	}

	app.requestRematch()
	if app.state.Players[0].Score != 2 {
		t.Fatalf("p1 score = %d, want unchanged 2", app.state.Players[0].Score)
	}
	if len(rematches) != 0 {
		t.Fatal("joiner sent rematch signal")
	}
}

func TestQueueMoveUsesSelectedPlayerFacing(t *testing.T) {
	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	app := localApp{
		cfg:    config.Default(),
		state:  state,
		player: game.PlayerTwo,
	}

	app.queueMove(0, -1)
	if app.pending.PlayerID != game.PlayerTwo {
		t.Fatalf("pending player = %d, want %d", app.pending.PlayerID, game.PlayerTwo)
	}
	if app.pending.Aim != game.Up {
		t.Fatalf("pending aim = %d, want %d", app.pending.Aim, game.Up)
	}
}

func TestPeerStatusText(t *testing.T) {
	if got := peerStatusText(netplay.PeerStatus{}); got != "P2 WAIT" {
		t.Fatalf("peerStatusText(disconnected) = %q, want P2 WAIT", got)
	}
	if got := peerStatusText(netplay.PeerStatus{Connected: true}); got != "P2 LIVE" {
		t.Fatalf("peerStatusText(connected) = %q, want P2 LIVE", got)
	}
	if got := peerStatusText(netplay.PeerStatus{Reason: "timeout"}); got != "P2 LOST" {
		t.Fatalf("peerStatusText(timeout) = %q, want P2 LOST", got)
	}
}

func TestPeerStatusControlsPlayerTwoVisibility(t *testing.T) {
	app := localApp{showPlayer2: false}

	app.applyPeerStatus(netplay.PeerStatus{Connected: true, PlayerName: "Nish"})
	if !app.showPlayer2 {
		t.Fatal("show player two = false, want true after connect")
	}
	if app.playerNames[1] != "Nish" {
		t.Fatalf("player two name = %q, want Nish", app.playerNames[1])
	}

	app.applyPeerStatus(netplay.PeerStatus{Reason: "disconnect"})
	if app.showPlayer2 {
		t.Fatal("show player two = true, want false after disconnect")
	}
	if app.playerNames[1] != "P2" {
		t.Fatalf("player two name = %q, want P2 after disconnect", app.playerNames[1])
	}
}

func TestPlayerNameOrDefault(t *testing.T) {
	if got := playerNameOrDefault("  Nish  ", "P1"); got != "Nish" {
		t.Fatalf("playerNameOrDefault() = %q, want Nish", got)
	}
	if got := playerNameOrDefault("", "P2"); got != "P2" {
		t.Fatalf("playerNameOrDefault(empty) = %q, want P2", got)
	}
	if got := playerNameOrDefault("VeryLongPlayerName", "P1"); got != "VeryLongPlay" {
		t.Fatalf("playerNameOrDefault(long) = %q, want VeryLongPlay", got)
	}
}
