package termui

import (
	"fmt"

	"github.com/nishchay-veer/gladiator/internal/netplay"
)

type netDebugSource struct {
	NetworkStats func() netplay.NetworkStats
	LinkStats    func() netplay.LinkStats
}

type netDebugStats struct {
	Active  bool
	Network netplay.NetworkStats
	Link    netplay.LinkStats
}

func (s netDebugSource) snapshot() netDebugStats {
	stats := netDebugStats{}
	if s.NetworkStats != nil {
		stats.Active = true
		stats.Network = s.NetworkStats()
	}
	if s.LinkStats != nil {
		stats.Active = true
		stats.Link = s.LinkStats()
	}
	return stats
}

func netDebugLine(stats netDebugStats) string {
	if !stats.Active {
		return ""
	}

	return fmt.Sprintf(" NET rx %d d%d bad%d dup%d old%d loss~%d | tx %d/%d simd%d delay%d ",
		stats.Network.PacketsReceived,
		stats.Network.PacketsDropped,
		stats.Network.InvalidPackets,
		stats.Network.DuplicatePackets,
		stats.Network.StalePackets,
		stats.Network.EstimatedLostPackets,
		stats.Link.PacketsSent,
		stats.Link.PacketsQueued,
		stats.Link.PacketsDropped,
		stats.Link.PacketsDelayed)
}
