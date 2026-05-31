package netplay

import "testing"

func TestPacketWindowTracksAckBitsAcrossGaps(t *testing.T) {
	var window packetWindow

	if obs := window.Observe(10); !obs.Accepted || !obs.Advanced {
		t.Fatalf("first observation = %#v, want accepted advanced", obs)
	}

	obs := window.Observe(13)
	if !obs.Accepted || !obs.Advanced || obs.Gap != 2 {
		t.Fatalf("gap observation = %#v, want accepted advanced gap 2", obs)
	}

	ack, bits := window.Ack()
	if ack != 13 {
		t.Fatalf("ack = %d, want 13", ack)
	}
	if bits != 0b100 {
		t.Fatalf("ack bits = %05b, want 00100", bits)
	}

	obs = window.Observe(12)
	if !obs.Accepted || !obs.Reordered || obs.Advanced {
		t.Fatalf("late observation = %#v, want accepted reordered", obs)
	}

	_, bits = window.Ack()
	if bits != 0b101 {
		t.Fatalf("ack bits after late packet = %05b, want 00101", bits)
	}

	stats := window.Stats()
	if stats.PacketsReceived != 3 {
		t.Fatalf("received = %d, want 3", stats.PacketsReceived)
	}
	if stats.ReorderedPackets != 1 {
		t.Fatalf("reordered = %d, want 1", stats.ReorderedPackets)
	}
	if stats.EstimatedLostPackets != 1 {
		t.Fatalf("estimated lost = %d, want 1", stats.EstimatedLostPackets)
	}
}

func TestPacketWindowDropsDuplicatesAndStalePackets(t *testing.T) {
	var window packetWindow

	window.Observe(100)
	duplicate := window.Observe(100)
	if !duplicate.Duplicate || duplicate.Accepted {
		t.Fatalf("duplicate observation = %#v, want duplicate drop", duplicate)
	}

	window.Observe(140)
	stale := window.Observe(99)
	if !stale.Stale || stale.Accepted {
		t.Fatalf("stale observation = %#v, want stale drop", stale)
	}

	stats := window.Stats()
	if stats.DuplicatePackets != 1 {
		t.Fatalf("duplicates = %d, want 1", stats.DuplicatePackets)
	}
	if stats.StalePackets != 1 {
		t.Fatalf("stale = %d, want 1", stats.StalePackets)
	}
	if stats.PacketsDropped != 2 {
		t.Fatalf("dropped = %d, want 2", stats.PacketsDropped)
	}
}

func TestSequenceGreaterHandlesWrap(t *testing.T) {
	if !sequenceGreater(1, ^uint32(0)) {
		t.Fatal("sequence 1 should be newer than max uint32 after wrap")
	}
	if sequenceGreater(^uint32(0), 1) {
		t.Fatal("max uint32 should be older than 1 after wrap")
	}
}
