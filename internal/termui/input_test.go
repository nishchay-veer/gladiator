package termui

import (
	"testing"

	"gladiator/internal/config"
	"gladiator/internal/game"
	"gladiator/internal/netplay"
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
