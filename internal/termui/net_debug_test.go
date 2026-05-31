package termui

import (
	"strings"
	"testing"

	"gladiator/internal/netplay"
)

func TestNetDebugSourceSnapshot(t *testing.T) {
	source := netDebugSource{
		NetworkStats: func() netplay.NetworkStats {
			return netplay.NetworkStats{
				PacketsReceived:      10,
				PacketsDropped:       2,
				EstimatedLostPackets: 1,
			}
		},
		LinkStats: func() netplay.LinkStats {
			return netplay.LinkStats{
				PacketsQueued:  9,
				PacketsSent:    7,
				PacketsDropped: 2,
			}
		},
	}

	got := source.snapshot()
	if !got.Active {
		t.Fatal("snapshot active = false, want true")
	}
	if got.Network.PacketsReceived != 10 {
		t.Fatalf("received = %d, want 10", got.Network.PacketsReceived)
	}
	if got.Link.PacketsQueued != 9 {
		t.Fatalf("queued = %d, want 9", got.Link.PacketsQueued)
	}
}

func TestNetDebugLine(t *testing.T) {
	line := netDebugLine(netDebugStats{
		Active: true,
		Network: netplay.NetworkStats{
			PacketsReceived:      12,
			PacketsDropped:       3,
			DuplicatePackets:     1,
			StalePackets:         2,
			EstimatedLostPackets: 4,
		},
		Link: netplay.LinkStats{
			PacketsQueued:  11,
			PacketsSent:    8,
			PacketsDropped: 3,
			PacketsDelayed: 5,
		},
	})

	for _, want := range []string{"rx 12", "d3", "dup1", "old2", "loss~4", "tx 8/11", "simd3", "delay5"} {
		if !strings.Contains(line, want) {
			t.Fatalf("netDebugLine() = %q, want to contain %q", line, want)
		}
	}
}

func TestNetDebugLineInactive(t *testing.T) {
	if got := netDebugLine(netDebugStats{}); got != "" {
		t.Fatalf("netDebugLine(inactive) = %q, want empty", got)
	}
}
