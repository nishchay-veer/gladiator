package netplay

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"gladiator/internal/game"
	"gladiator/internal/protocol"
)

type HostOptions struct {
	Addr      string
	SessionID uint64
}

type Host struct {
	conn      *net.UDPConn
	sessionID uint64

	mu       sync.Mutex
	state    game.State
	remote   *net.UDPAddr
	sequence uint32

	remoteInputSequence uint32
	remoteLastSeen      time.Time
}

func ListenHost(opts HostOptions) (*Host, error) {
	addr := opts.Addr
	if addr == "" {
		addr = ":42424"
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	state, err := game.NewLocalState()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	sessionID := opts.SessionID
	if sessionID == 0 {
		sessionID = uint64(time.Now().UnixNano())
	}

	return &Host{
		conn:      conn,
		sessionID: sessionID,
		state:     state,
	}, nil
}

func (h *Host) Serve(ctx context.Context) error {
	buffer := make([]byte, protocol.MaxPacketSize)

	for {
		data, addr, err := readDatagram(ctx, h.conn, buffer)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			if errors.Is(err, net.ErrClosed) && ctx.Err() != nil {
				return nil
			}
			return err
		}

		packet, err := protocol.Decode(data)
		if err != nil {
			continue
		}
		if err := h.handlePacket(addr, packet); err != nil {
			return err
		}
	}
}

func (h *Host) Close() error {
	return h.conn.Close()
}

func (h *Host) Addr() net.Addr {
	return h.conn.LocalAddr()
}

func (h *Host) Snapshot() game.Snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.state.Snapshot()
}

func (h *Host) RemoteConnected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.remote != nil
}

func (h *Host) handlePacket(addr *net.UDPAddr, packet protocol.Packet) error {
	switch packet.Type {
	case protocol.PacketHello:
		return h.handleHello(addr, packet)
	case protocol.PacketInput:
		return h.handleInput(addr, packet)
	case protocol.PacketPing:
		h.handlePing(addr, packet)
		return nil
	case protocol.PacketDisconnect:
		h.handleDisconnect(addr, packet)
		return nil
	default:
		return nil
	}
}

func (h *Host) handleHello(addr *net.UDPAddr, packet protocol.Packet) error {
	payload, ok := packet.Payload.(protocol.HelloPayload)
	if !ok || !payload.Ready {
		return nil
	}

	h.mu.Lock()
	if h.remote != nil && !sameUDPAddr(h.remote, addr) {
		h.mu.Unlock()
		return nil
	}

	if h.remote == nil {
		h.remoteInputSequence = 0
	}
	h.remote = cloneUDPAddr(addr)
	h.remoteLastSeen = time.Now()
	snapshot := h.state.Snapshot()
	mapHash := h.state.Arena.Hash()
	response := protocol.Packet{
		Type:      protocol.PacketWelcome,
		SessionID: h.sessionID,
		Sequence:  h.nextSequenceLocked(),
		Tick:      snapshot.Tick,
		Payload: protocol.WelcomePayload{
			PlayerID: game.PlayerTwo,
			MapID:    snapshot.Match.MapID,
			MapHash:  mapHash,
			Ready:    true,
			Snapshot: snapshot,
		},
	}
	h.mu.Unlock()

	return sendPacket(h.conn, addr, response)
}

func (h *Host) handleInput(addr *net.UDPAddr, packet protocol.Packet) error {
	payload, ok := packet.Payload.(protocol.InputPayload)
	if !ok {
		return nil
	}

	h.mu.Lock()
	if packet.SessionID != h.sessionID || !sameUDPAddr(h.remote, addr) || payload.Command.PlayerID != game.PlayerTwo {
		h.mu.Unlock()
		return nil
	}

	if payload.Command.Tick == h.state.Tick {
		frame := game.NewInputFrame(h.state.Tick)
		frame.Set(payload.Command)
		h.state.Step(frame)
	}

	snapshot := h.state.Snapshot()
	response := protocol.Packet{
		Type:      protocol.PacketSnapshot,
		SessionID: h.sessionID,
		Sequence:  h.nextSequenceLocked(),
		Tick:      snapshot.Tick,
		Payload:   protocol.SnapshotPayload{Snapshot: snapshot},
	}
	h.mu.Unlock()

	return sendPacket(h.conn, addr, response)
}

func (h *Host) handlePing(addr *net.UDPAddr, packet protocol.Packet) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if packet.SessionID == h.sessionID && sameUDPAddr(h.remote, addr) {
		h.remoteLastSeen = time.Now()
	}
}

func (h *Host) handleDisconnect(addr *net.UDPAddr, packet protocol.Packet) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if packet.SessionID == h.sessionID && sameUDPAddr(h.remote, addr) {
		h.remote = nil
		h.remoteInputSequence = 0
		h.remoteLastSeen = time.Time{}
	}
}

func (h *Host) disconnectTimedOutRemote(now time.Time, timeout time.Duration) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if timeout <= 0 || h.remote == nil || h.remoteLastSeen.IsZero() {
		return false
	}
	if now.Sub(h.remoteLastSeen) < timeout {
		return false
	}

	h.remote = nil
	h.remoteInputSequence = 0
	h.remoteLastSeen = time.Time{}
	return true
}

func (h *Host) nextSequenceLocked() uint32 {
	h.sequence++
	return h.sequence
}
