package netplay

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/nishchay-veer/gladiator/internal/game"
	"github.com/nishchay-veer/gladiator/internal/protocol"
)

const (
	defaultSimulationRate = time.Second / 60
	defaultSnapshotRate   = time.Second / 30
	defaultHeartbeatRate  = 500 * time.Millisecond
	defaultPeerTimeout    = 2 * time.Second
	defaultSessionBuffer  = 32
)

type SessionOptions struct {
	SimulationRate time.Duration
	SnapshotRate   time.Duration
	HeartbeatRate  time.Duration
	PeerTimeout    time.Duration
	InputBuffer    int
	SnapshotBuffer int
	LinkSimulation LinkSimulation
}

type PeerStatus struct {
	Connected  bool
	Reason     string
	PlayerName string
}

type HostSession struct {
	Inputs       chan<- game.InputCommand
	Rematches    chan<- struct{}
	Snapshots    <-chan game.Snapshot
	PeerStatus   <-chan PeerStatus
	Errors       <-chan error
	NetworkStats func() NetworkStats
	LinkStats    func() LinkStats
}

type ClientSession struct {
	Join         JoinResult
	Inputs       chan<- game.InputCommand
	Snapshots    <-chan game.Snapshot
	Errors       <-chan error
	NetworkStats func() NetworkStats
	LinkStats    func() LinkStats
}

func (h *Host) StartSession(ctx context.Context, opts SessionOptions) HostSession {
	opts = normalizeSessionOptions(opts)

	localInputs := make(chan game.InputCommand, opts.InputBuffer)
	rematches := make(chan struct{}, opts.InputBuffer)
	remoteInputs := make(chan game.InputCommand, opts.InputBuffer)
	snapshots := make(chan game.Snapshot, opts.SnapshotBuffer)
	peerStatus := make(chan PeerStatus, opts.SnapshotBuffer)
	errors := make(chan error, 1)
	sender := newPacketSender(h.conn, opts.LinkSimulation)

	go h.runSessionPackets(ctx, remoteInputs, peerStatus, errors)
	go h.runSessionTicks(ctx, opts, sender, localInputs, rematches, remoteInputs, snapshots, errors)
	go h.runSessionTimeouts(ctx, opts, peerStatus)

	return HostSession{
		Inputs:       localInputs,
		Rematches:    rematches,
		Snapshots:    snapshots,
		PeerStatus:   peerStatus,
		Errors:       errors,
		NetworkStats: h.Stats,
		LinkStats:    sender.Stats,
	}
}

func (c *Client) StartSession(ctx context.Context, opts SessionOptions) (ClientSession, error) {
	opts = normalizeSessionOptions(opts)

	join, err := c.Join(ctx)
	if err != nil {
		return ClientSession{}, err
	}

	return c.StartJoinedSession(ctx, join, opts), nil
}

func (c *Client) StartJoinedSession(ctx context.Context, join JoinResult, opts SessionOptions) ClientSession {
	opts = normalizeSessionOptions(opts)

	inputs := make(chan game.InputCommand, opts.InputBuffer)
	snapshots := make(chan game.Snapshot, opts.SnapshotBuffer)
	errors := make(chan error, 1)
	sender := newPacketSender(c.conn, opts.LinkSimulation)

	c.mu.Lock()
	c.sessionID = join.SessionID
	c.playerID = join.PlayerID
	c.mu.Unlock()

	offerSnapshot(snapshots, join.Snapshot)
	go c.runSessionReads(ctx, join.SessionID, snapshots, errors)
	go c.runSessionWrites(ctx, join, opts, sender, inputs, errors)

	return ClientSession{
		Join:         join,
		Inputs:       inputs,
		Snapshots:    snapshots,
		Errors:       errors,
		NetworkStats: c.Stats,
		LinkStats:    sender.Stats,
	}
}

func normalizeSessionOptions(opts SessionOptions) SessionOptions {
	if opts.SimulationRate <= 0 {
		opts.SimulationRate = defaultSimulationRate
	}
	if opts.SnapshotRate <= 0 {
		opts.SnapshotRate = defaultSnapshotRate
	}
	if opts.HeartbeatRate <= 0 {
		opts.HeartbeatRate = defaultHeartbeatRate
	}
	if opts.PeerTimeout <= 0 {
		opts.PeerTimeout = defaultPeerTimeout
	}
	if opts.InputBuffer <= 0 {
		opts.InputBuffer = defaultSessionBuffer
	}
	if opts.SnapshotBuffer <= 0 {
		opts.SnapshotBuffer = defaultSessionBuffer
	}
	return opts
}

func (h *Host) runSessionPackets(ctx context.Context, remoteInputs chan game.InputCommand, peerStatus chan PeerStatus, errors chan<- error) {
	buffer := make([]byte, protocol.MaxPacketSize)

	for {
		data, addr, err := readDatagram(ctx, h.conn, buffer)
		if err != nil {
			if sessionDone(ctx, err) {
				return
			}
			reportSessionError(errors, err)
			return
		}

		if h.handleDiscoveryDatagram(addr, data) {
			continue
		}

		packet, err := protocol.Decode(data)
		if err != nil {
			continue
		}

		switch packet.Type {
		case protocol.PacketHello:
			wasConnected := h.RemoteConnected()
			if err := h.handleHello(addr, packet); err != nil {
				reportSessionError(errors, err)
				return
			}
			if !wasConnected && h.RemoteConnected() {
				offerPeerStatus(peerStatus, PeerStatus{
					Connected:  true,
					Reason:     "hello",
					PlayerName: h.RemotePlayerName(),
				})
			}
		case protocol.PacketInput:
			h.queueRemoteInput(addr, packet, remoteInputs)
		case protocol.PacketPing:
			h.handlePing(addr, packet)
		case protocol.PacketDisconnect:
			wasConnected := h.RemoteConnected()
			h.handleDisconnect(addr, packet)
			if wasConnected && !h.RemoteConnected() {
				offerPeerStatus(peerStatus, PeerStatus{Reason: "disconnect"})
			}
		}
	}
}

func (h *Host) runSessionTicks(ctx context.Context, opts SessionOptions, sender *packetSender, localInputs <-chan game.InputCommand, rematches <-chan struct{}, remoteInputs <-chan game.InputCommand, snapshots chan game.Snapshot, errors chan<- error) {
	simulationTicker := time.NewTicker(opts.SimulationRate)
	defer simulationTicker.Stop()

	snapshotTicker := time.NewTicker(opts.SnapshotRate)
	defer snapshotTicker.Stop()

	h.publishCurrentSnapshot(ctx, sender, snapshots, errors)

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-rematches:
			if !ok {
				rematches = nil
				continue
			}
			h.resetMatch()
			h.publishCurrentSnapshot(ctx, sender, snapshots, errors)
		case <-simulationTicker.C:
			local := drainCommand(localInputs, game.NewInputCommand(0, game.PlayerOne))
			remote := drainCommand(remoteInputs, game.NewInputCommand(0, game.PlayerTwo))

			h.mu.Lock()
			frame := game.NewInputFrame(h.state.Tick)
			local.PlayerID = game.PlayerOne
			remote.PlayerID = game.PlayerTwo
			frame.Set(local)
			frame.Set(remote)
			h.state.Step(frame)
			h.mu.Unlock()
		case <-snapshotTicker.C:
			h.publishCurrentSnapshot(ctx, sender, snapshots, errors)
		}
	}
}

func (h *Host) resetMatch() {
	h.mu.Lock()
	defer h.mu.Unlock()

	_ = h.state.ResetMatch()
}

func (h *Host) runSessionTimeouts(ctx context.Context, opts SessionOptions, peerStatus chan PeerStatus) {
	ticker := time.NewTicker(timeoutCheckRate(opts.PeerTimeout))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if h.disconnectTimedOutRemote(now, opts.PeerTimeout) {
				offerPeerStatus(peerStatus, PeerStatus{Reason: "timeout"})
			}
		}
	}
}

func (h *Host) publishCurrentSnapshot(ctx context.Context, sender *packetSender, snapshots chan game.Snapshot, errors chan<- error) {
	h.mu.Lock()
	snapshot := h.state.Snapshot()
	remote := cloneUDPAddr(h.remote)
	var packet protocol.Packet
	if remote != nil {
		ack, ackBits := h.remotePackets.Ack()
		packet = protocol.Packet{
			Type:      protocol.PacketSnapshot,
			SessionID: h.sessionID,
			Sequence:  h.nextSequenceLocked(),
			Ack:       ack,
			AckBits:   ackBits,
			Tick:      snapshot.Tick,
			Payload:   protocol.SnapshotPayload{Snapshot: snapshot},
		}
	}
	h.mu.Unlock()

	offerSnapshot(snapshots, snapshot)
	if remote != nil {
		sender.Send(ctx, remote, packet, errors)
	}
}

func (h *Host) queueRemoteInput(addr *net.UDPAddr, packet protocol.Packet, remoteInputs chan game.InputCommand) {
	payload, ok := packet.Payload.(protocol.InputPayload)
	if !ok {
		return
	}

	h.mu.Lock()
	knownRemote := packet.SessionID == h.sessionID && sameUDPAddr(h.remote, addr) && payload.Command.PlayerID == game.PlayerTwo
	var observation packetObservation
	if knownRemote {
		if h.validateRemoteInputLocked(packet, payload.Command) {
			observation = h.remotePackets.Observe(packet.Sequence)
			h.remoteLastSeen = time.Now()
		} else {
			h.remotePackets.RejectInvalid()
		}
	}
	h.mu.Unlock()

	if knownRemote && observation.Advanced {
		offerCommand(remoteInputs, payload.Command)
	}
}

func (c *Client) runSessionReads(ctx context.Context, sessionID uint64, snapshots chan game.Snapshot, errors chan<- error) {
	for {
		packet, observation, err := c.readPacketFromHost(ctx)
		if err != nil {
			if sessionDone(ctx, err) {
				return
			}
			reportSessionError(errors, err)
			return
		}
		if packet.Type != protocol.PacketSnapshot || packet.SessionID != sessionID || !observation.Advanced {
			continue
		}

		payload, ok := packet.Payload.(protocol.SnapshotPayload)
		if ok {
			offerSnapshot(snapshots, payload.Snapshot)
		}
	}
}

func (c *Client) runSessionWrites(ctx context.Context, join JoinResult, opts SessionOptions, sender *packetSender, inputs <-chan game.InputCommand, errors chan<- error) {
	heartbeatTicker := time.NewTicker(opts.HeartbeatRate)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			sequence, ack, ackBits := c.nextPacketHeader()
			packet := protocol.Packet{
				Type:      protocol.PacketPing,
				SessionID: join.SessionID,
				Sequence:  sequence,
				Ack:       ack,
				AckBits:   ackBits,
				Payload:   protocol.PingPayload{SentAtUnixMillis: time.Now().UnixMilli()},
			}
			sender.Send(ctx, c.hostAddr, packet, errors)
		case command, ok := <-inputs:
			if !ok {
				return
			}
			command.PlayerID = join.PlayerID
			sequence, ack, ackBits := c.nextPacketHeader()
			command.Sequence = sequence
			packet := protocol.Packet{
				Type:      protocol.PacketInput,
				SessionID: join.SessionID,
				Sequence:  command.Sequence,
				Ack:       ack,
				AckBits:   ackBits,
				Tick:      command.Tick,
				Payload:   protocol.InputPayload{Command: command},
			}
			sender.Send(ctx, c.hostAddr, packet, errors)
		}
	}
}

func timeoutCheckRate(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultHeartbeatRate
	}
	rate := timeout / 3
	if rate < time.Millisecond {
		return time.Millisecond
	}
	if rate > defaultHeartbeatRate {
		return defaultHeartbeatRate
	}
	return rate
}

func drainCommand(inputs <-chan game.InputCommand, fallback game.InputCommand) game.InputCommand {
	command := fallback
	for {
		select {
		case next := <-inputs:
			command = next
		default:
			return command
		}
	}
}

func offerCommand(commands chan game.InputCommand, command game.InputCommand) {
	select {
	case commands <- command:
	default:
		select {
		case <-commands:
		default:
		}
		select {
		case commands <- command:
		default:
		}
	}
}

func offerSnapshot(snapshots chan game.Snapshot, snapshot game.Snapshot) {
	select {
	case snapshots <- snapshot:
	default:
		select {
		case <-snapshots:
		default:
		}
		select {
		case snapshots <- snapshot:
		default:
		}
	}
}

func offerPeerStatus(statuses chan PeerStatus, status PeerStatus) {
	select {
	case statuses <- status:
	default:
		select {
		case <-statuses:
		default:
		}
		select {
		case statuses <- status:
		default:
		}
	}
}

func reportSessionError(errors chan<- error, err error) {
	if err == nil {
		return
	}
	select {
	case errors <- err:
	default:
	}
}

func sessionDone(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)
}
