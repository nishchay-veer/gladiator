package netplay

import (
	"context"
	"net"
	"sync"
	"time"

	"gladiator/internal/protocol"
)

type LinkSimulation struct {
	DropEvery int
	BaseDelay time.Duration
	Jitter    time.Duration
}

type LinkStats struct {
	PacketsQueued  uint64
	PacketsSent    uint64
	PacketsDropped uint64
	PacketsDelayed uint64
}

type packetSender struct {
	conn       *net.UDPConn
	simulation LinkSimulation

	mu    sync.Mutex
	count uint64
	stats LinkStats
}

func newPacketSender(conn *net.UDPConn, simulation LinkSimulation) *packetSender {
	return &packetSender{
		conn:       conn,
		simulation: simulation.normalized(),
	}
}

func (s *packetSender) Send(ctx context.Context, addr *net.UDPAddr, packet protocol.Packet, errors chan<- error) {
	data, err := protocol.Encode(packet)
	if err != nil {
		reportSessionError(errors, err)
		return
	}

	drop, delay := s.planSend()
	if drop {
		return
	}
	if delay <= 0 {
		s.write(ctx, data, addr, errors)
		return
	}

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.write(ctx, data, addr, errors)
		}
	}()
}

func (s *packetSender) Stats() LinkStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.stats
}

func (s *packetSender) planSend() (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	index := s.count
	s.stats.PacketsQueued++

	if s.simulation.DropEvery > 0 && index%uint64(s.simulation.DropEvery) == 0 {
		s.stats.PacketsDropped++
		return true, 0
	}

	delay := s.simulation.delayFor(index)
	if delay > 0 {
		s.stats.PacketsDelayed++
	}
	return false, delay
}

func (s *packetSender) write(ctx context.Context, data []byte, addr *net.UDPAddr, errors chan<- error) {
	_, err := s.conn.WriteToUDP(data, addr)
	if err != nil {
		if sessionDone(ctx, err) {
			return
		}
		reportSessionError(errors, err)
		return
	}

	s.mu.Lock()
	s.stats.PacketsSent++
	s.mu.Unlock()
}

func (s LinkSimulation) normalized() LinkSimulation {
	if s.DropEvery < 0 {
		s.DropEvery = 0
	}
	if s.BaseDelay < 0 {
		s.BaseDelay = 0
	}
	if s.Jitter < 0 {
		s.Jitter = 0
	}
	return s
}

func (s LinkSimulation) delayFor(index uint64) time.Duration {
	s = s.normalized()
	delay := s.BaseDelay
	if s.Jitter <= 0 {
		return delay
	}

	switch index % 4 {
	case 1:
		return delay
	case 2:
		return delay + s.Jitter/2
	case 3:
		return delay + s.Jitter
	default:
		return delay + s.Jitter/4
	}
}
