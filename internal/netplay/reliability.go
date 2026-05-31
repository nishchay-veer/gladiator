package netplay

type NetworkStats struct {
	PacketsReceived      uint64
	PacketsDropped       uint64
	DuplicatePackets     uint64
	StalePackets         uint64
	InvalidPackets       uint64
	ReorderedPackets     uint64
	EstimatedLostPackets uint64
	LatestSequence       uint32
	Ack                  uint32
	AckBits              uint32
}

type packetObservation struct {
	Accepted  bool
	Advanced  bool
	Duplicate bool
	Stale     bool
	Reordered bool
	Gap       uint32
}

type packetWindow struct {
	seen    bool
	latest  uint32
	ackBits uint32
	stats   NetworkStats
}

func (w *packetWindow) Observe(sequence uint32) packetObservation {
	w.stats.PacketsReceived++

	if !w.seen {
		w.seen = true
		w.latest = sequence
		return packetObservation{Accepted: true, Advanced: true}
	}

	if sequenceGreater(sequence, w.latest) {
		shift := sequence - w.latest
		gap := shift - 1
		if shift > 32 {
			w.ackBits = 0
		} else {
			w.ackBits = (w.ackBits << shift) | (uint32(1) << (shift - 1))
		}
		w.latest = sequence
		w.stats.EstimatedLostPackets += uint64(gap)
		return packetObservation{Accepted: true, Advanced: true, Gap: gap}
	}

	delta := w.latest - sequence
	if delta == 0 {
		w.stats.DuplicatePackets++
		w.stats.PacketsDropped++
		return packetObservation{Duplicate: true}
	}
	if delta > 32 {
		w.stats.StalePackets++
		w.stats.PacketsDropped++
		return packetObservation{Stale: true}
	}

	bit := uint32(1) << (delta - 1)
	if w.ackBits&bit != 0 {
		w.stats.DuplicatePackets++
		w.stats.PacketsDropped++
		return packetObservation{Duplicate: true}
	}

	w.ackBits |= bit
	w.stats.ReorderedPackets++
	if w.stats.EstimatedLostPackets > 0 {
		w.stats.EstimatedLostPackets--
	}
	return packetObservation{Accepted: true, Reordered: true}
}

func (w *packetWindow) RejectInvalid() {
	w.stats.PacketsReceived++
	w.stats.PacketsDropped++
	w.stats.InvalidPackets++
}

func (w packetWindow) Ack() (uint32, uint32) {
	if !w.seen {
		return 0, 0
	}
	return w.latest, w.ackBits
}

func (w packetWindow) Stats() NetworkStats {
	stats := w.stats
	if w.seen {
		stats.LatestSequence = w.latest
		stats.Ack = w.latest
		stats.AckBits = w.ackBits
	}
	return stats
}

func sequenceGreater(a, b uint32) bool {
	return a != b && int32(a-b) > 0
}
