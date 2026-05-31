package netplay

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nishchay-veer/gladiator/internal/game"
	"github.com/nishchay-veer/gladiator/internal/protocol"
)

func TestLoopbackJoinAndInputSnapshot(t *testing.T) {
	host := startLoopbackHost(t)
	client := dialLoopbackClient(t, host)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	welcome, err := client.Join(ctx)
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}
	if welcome.SessionID != 4242 {
		t.Fatalf("session id = %d, want 4242", welcome.SessionID)
	}
	if welcome.PlayerID != game.PlayerTwo {
		t.Fatalf("player id = %d, want %d", welcome.PlayerID, game.PlayerTwo)
	}
	if welcome.MapID != "local-arena-01" {
		t.Fatalf("map id = %q, want local-arena-01", welcome.MapID)
	}
	if welcome.MapHash == 0 {
		t.Fatal("map hash = 0, want non-zero")
	}
	if !welcome.Ready {
		t.Fatal("ready = false, want true")
	}
	if welcome.Snapshot.Tick != 0 {
		t.Fatalf("welcome tick = %d, want 0", welcome.Snapshot.Tick)
	}

	command := game.NewInputCommand(welcome.Snapshot.Tick, welcome.PlayerID)
	command.MoveX = -1
	command.Aim = game.Left
	command.HasAim = true

	snapshot, err := client.SendInput(ctx, command)
	if err != nil {
		t.Fatalf("SendInput() error = %v", err)
	}
	if snapshot.Tick != 1 {
		t.Fatalf("snapshot tick = %d, want 1", snapshot.Tick)
	}

	wantPosition := game.Point{X: 34, Y: 14}
	if snapshot.Players[1].Position != wantPosition {
		t.Fatalf("player two position = %+v, want %+v", snapshot.Players[1].Position, wantPosition)
	}
	if !host.Snapshot().Equal(snapshot) {
		t.Fatalf("host snapshot and client snapshot diverged\nhost:   %#v\nclient: %#v", host.Snapshot(), snapshot)
	}
}

func TestLoopbackHostReturnsSnapshotForStaleInput(t *testing.T) {
	host := startLoopbackHost(t)
	client := dialLoopbackClient(t, host)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	welcome, err := client.Join(ctx)
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}

	command := game.NewInputCommand(welcome.Snapshot.Tick, welcome.PlayerID)
	command.MoveX = -1
	command.Aim = game.Left
	command.HasAim = true

	first, err := client.SendInput(ctx, command)
	if err != nil {
		t.Fatalf("first SendInput() error = %v", err)
	}

	stale, err := client.SendInput(ctx, command)
	if err != nil {
		t.Fatalf("stale SendInput() error = %v", err)
	}
	if stale.Tick != first.Tick {
		t.Fatalf("stale snapshot tick = %d, want %d", stale.Tick, first.Tick)
	}
	if !stale.Equal(first) {
		t.Fatalf("stale input advanced state\ngot:  %#v\nwant: %#v", stale, first)
	}
}

func TestLoopbackPacketStatsTrackPeerSequences(t *testing.T) {
	host := startLoopbackHost(t)
	client := dialLoopbackClient(t, host)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	welcome, err := client.Join(ctx)
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}

	if stats := host.Stats(); stats.Ack != 1 || stats.PacketsReceived != 1 {
		t.Fatalf("host stats after join = %#v, want ack 1 with one received packet", stats)
	}
	if stats := client.Stats(); stats.Ack != 1 || stats.PacketsReceived != 1 {
		t.Fatalf("client stats after join = %#v, want ack 1 with one received packet", stats)
	}

	command := game.NewInputCommand(welcome.Snapshot.Tick, welcome.PlayerID)
	command.MoveX = -1
	command.Aim = game.Left
	command.HasAim = true
	if _, err := client.SendInput(ctx, command); err != nil {
		t.Fatalf("SendInput() error = %v", err)
	}

	if stats := host.Stats(); stats.Ack != 2 || stats.PacketsReceived != 2 {
		t.Fatalf("host stats after input = %#v, want ack 2 with two received packets", stats)
	}
	if stats := client.Stats(); stats.Ack != 2 || stats.PacketsReceived != 2 {
		t.Fatalf("client stats after snapshot = %#v, want ack 2 with two received packets", stats)
	}
}

func TestHostQueueRemoteInputDropsDuplicateSequences(t *testing.T) {
	remote := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}
	host := newValidationHost(t, remote, 0)
	remoteInputs := make(chan game.InputCommand, 2)

	command := game.NewInputCommand(0, game.PlayerTwo)
	command.Sequence = 1
	packet := protocol.Packet{
		Type:      protocol.PacketInput,
		SessionID: 99,
		Sequence:  1,
		Tick:      command.Tick,
		Payload:   protocol.InputPayload{Command: command},
	}

	host.queueRemoteInput(remote, packet, remoteInputs)
	host.queueRemoteInput(remote, packet, remoteInputs)

	if got := len(remoteInputs); got != 1 {
		t.Fatalf("queued inputs = %d, want 1", got)
	}
	stats := host.Stats()
	if stats.DuplicatePackets != 1 || stats.PacketsDropped != 1 {
		t.Fatalf("host stats = %#v, want one duplicate drop", stats)
	}
}

func TestHostQueueRemoteInputRejectsInvalidCommands(t *testing.T) {
	tests := []struct {
		name        string
		currentTick uint64
		mutate      func(*protocol.Packet, *game.InputCommand)
	}{
		{
			name: "axis out of range",
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				command.MoveX = 2
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
		{
			name: "diagonal movement",
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				command.MoveX = 1
				command.MoveY = 1
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
		{
			name: "unknown button",
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				command.Buttons = game.Buttons(1 << 7)
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
		{
			name: "sequence mismatch",
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				command.Sequence = packet.Sequence + 1
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
		{
			name: "tick mismatch",
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				command.Tick = packet.Tick + 1
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
		{
			name:        "too stale",
			currentTick: maxRemoteInputLagTicks + 1,
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				packet.Tick = 0
				command.Tick = 0
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
		{
			name: "too far ahead",
			mutate: func(packet *protocol.Packet, command *game.InputCommand) {
				packet.Tick = maxRemoteInputLeadTicks + 1
				command.Tick = packet.Tick
				packet.Payload = protocol.InputPayload{Command: *command}
			},
		},
	}

	remote := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := newValidationHost(t, remote, tt.currentTick)
			remoteInputs := make(chan game.InputCommand, 1)
			packet, command := remoteInputPacket(tt.currentTick, 1)
			tt.mutate(&packet, &command)

			host.queueRemoteInput(remote, packet, remoteInputs)

			if got := len(remoteInputs); got != 0 {
				t.Fatalf("queued inputs = %d, want 0", got)
			}
			stats := host.Stats()
			if stats.InvalidPackets != 1 {
				t.Fatalf("invalid packets = %d, want 1", stats.InvalidPackets)
			}
			if stats.PacketsDropped != 1 {
				t.Fatalf("dropped packets = %d, want 1", stats.PacketsDropped)
			}
		})
	}
}

func TestHostQueueRemoteInputAcceptsTickWindowEdges(t *testing.T) {
	remote := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}
	currentTick := uint64(100)
	host := newValidationHost(t, remote, currentTick)
	remoteInputs := make(chan game.InputCommand, 2)

	lagPacket, _ := remoteInputPacket(currentTick-maxRemoteInputLagTicks, 1)
	host.queueRemoteInput(remote, lagPacket, remoteInputs)

	leadPacket, _ := remoteInputPacket(currentTick+maxRemoteInputLeadTicks, 2)
	host.queueRemoteInput(remote, leadPacket, remoteInputs)

	if got := len(remoteInputs); got != 2 {
		t.Fatalf("queued inputs = %d, want 2", got)
	}
	if stats := host.Stats(); stats.InvalidPackets != 0 || stats.PacketsDropped != 0 {
		t.Fatalf("host stats = %#v, want no validation drops", stats)
	}
}

func TestClientRequiresJoinBeforeInput(t *testing.T) {
	host := startLoopbackHost(t)
	client := dialLoopbackClient(t, host)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	command := game.NewInputCommand(0, game.PlayerTwo)
	if _, err := client.SendInput(ctx, command); err == nil {
		t.Fatal("SendInput() error = nil, want error before Join")
	}
}

func TestContinuousHostSessionAppliesLocalInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := ListenHost(HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 9001,
	})
	if err != nil {
		t.Fatalf("ListenHost() error = %v", err)
	}
	defer host.Close()

	session := host.StartSession(ctx, fastSessionOptions())
	initial := waitForSnapshot(t, session.Snapshots, func(snapshot game.Snapshot) bool {
		return true
	}, session.Errors)

	command := game.NewInputCommand(initial.Tick, game.PlayerOne)
	command.MoveX = 1
	command.Aim = game.Right
	command.HasAim = true
	sendSessionInput(t, session.Inputs, command)

	moved := waitForSnapshot(t, session.Snapshots, func(snapshot game.Snapshot) bool {
		return snapshot.Players[0].Position.X > initial.Players[0].Position.X
	}, session.Errors)
	if moved.Players[0].Position != (game.Point{X: 2, Y: 1}) {
		t.Fatalf("player one position = %+v, want {X:2 Y:1}", moved.Players[0].Position)
	}
}

func TestContinuousClientSessionStreamsSnapshotsAndRemoteInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := ListenHost(HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 9002,
	})
	if err != nil {
		t.Fatalf("ListenHost() error = %v", err)
	}
	defer host.Close()

	hostSession := host.StartSession(ctx, fastSessionOptions())
	client := dialLoopbackClient(t, host)
	defer client.Close()

	clientSession, err := client.StartSession(ctx, fastSessionOptions())
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if clientSession.Join.PlayerID != game.PlayerTwo {
		t.Fatalf("player id = %d, want %d", clientSession.Join.PlayerID, game.PlayerTwo)
	}
	if !clientSession.Join.Ready {
		t.Fatal("join ready = false, want true")
	}

	initial := waitForSnapshot(t, clientSession.Snapshots, func(snapshot game.Snapshot) bool {
		return snapshot.Tick >= clientSession.Join.Snapshot.Tick
	}, hostSession.Errors, clientSession.Errors)

	streamed := waitForSnapshot(t, clientSession.Snapshots, func(snapshot game.Snapshot) bool {
		return snapshot.Tick > initial.Tick
	}, hostSession.Errors, clientSession.Errors)
	if streamed.Tick <= initial.Tick {
		t.Fatalf("streamed tick = %d, want > %d", streamed.Tick, initial.Tick)
	}

	command := game.NewInputCommand(streamed.Tick, clientSession.Join.PlayerID)
	command.MoveX = -1
	command.Aim = game.Left
	command.HasAim = true
	sendSessionInput(t, clientSession.Inputs, command)

	moved := waitForSnapshot(t, clientSession.Snapshots, func(snapshot game.Snapshot) bool {
		return snapshot.Players[1].Position.X < initial.Players[1].Position.X
	}, hostSession.Errors, clientSession.Errors)
	if !host.Snapshot().Equal(moved) {
		t.Fatalf("host snapshot and client snapshot diverged\nhost:   %#v\nclient: %#v", host.Snapshot(), moved)
	}
}

func TestContinuousSessionSimulatesLossAndJitter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := ListenHost(HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 9005,
	})
	if err != nil {
		t.Fatalf("ListenHost() error = %v", err)
	}
	defer host.Close()

	hostOpts := fastSessionOptions()
	hostOpts.PeerTimeout = time.Second
	hostSession := host.StartSession(ctx, hostOpts)

	client := dialLoopbackClient(t, host)
	defer client.Close()

	clientOpts := fastSessionOptions()
	clientOpts.HeartbeatRate = time.Hour
	clientOpts.LinkSimulation = LinkSimulation{
		DropEvery: 2,
		BaseDelay: time.Millisecond,
		Jitter:    time.Millisecond,
	}
	clientSession, err := client.StartSession(ctx, clientOpts)
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	waitForPeerStatus(t, hostSession.PeerStatus, true, hostSession.Errors, clientSession.Errors)

	for i := 0; i < 3; i++ {
		command := game.NewInputCommand(0, clientSession.Join.PlayerID)
		command.MoveX = -1
		command.Aim = game.Left
		command.HasAim = true
		sendSessionInput(t, clientSession.Inputs, command)
	}

	waitForCondition(t, func() bool {
		linkStats := clientSession.LinkStats()
		networkStats := host.Stats()
		return linkStats.PacketsDropped >= 1 &&
			linkStats.PacketsDelayed >= 1 &&
			networkStats.EstimatedLostPackets >= 1
	}, hostSession.Errors, clientSession.Errors)
}

func TestContinuousSessionReportsPeerDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := ListenHost(HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 9003,
	})
	if err != nil {
		t.Fatalf("ListenHost() error = %v", err)
	}
	defer host.Close()

	hostSession := host.StartSession(ctx, fastSessionOptions())
	client := dialLoopbackClient(t, host)
	defer client.Close()

	clientSession, err := client.StartSession(ctx, fastSessionOptions())
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	waitForPeerStatus(t, hostSession.PeerStatus, true, hostSession.Errors, clientSession.Errors)
	if !host.RemoteConnected() {
		t.Fatal("host remote connected = false, want true")
	}

	disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), time.Second)
	err = client.Disconnect(disconnectCtx, "quit")
	cancelDisconnect()
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	waitForPeerStatus(t, hostSession.PeerStatus, false, hostSession.Errors, clientSession.Errors)
	if host.RemoteConnected() {
		t.Fatal("host remote connected = true, want false")
	}
}

func TestContinuousSessionTimesOutSilentPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := ListenHost(HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 9004,
	})
	if err != nil {
		t.Fatalf("ListenHost() error = %v", err)
	}
	defer host.Close()

	opts := fastSessionOptions()
	opts.PeerTimeout = 30 * time.Millisecond
	opts.HeartbeatRate = 5 * time.Millisecond

	hostSession := host.StartSession(ctx, opts)
	client := dialLoopbackClient(t, host)
	defer client.Close()

	clientCtx, cancelClient := context.WithCancel(ctx)
	clientSession, err := client.StartSession(clientCtx, opts)
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	waitForPeerStatus(t, hostSession.PeerStatus, true, hostSession.Errors, clientSession.Errors)
	cancelClient()

	status := waitForPeerStatus(t, hostSession.PeerStatus, false, hostSession.Errors)
	if status.Reason != "timeout" {
		t.Fatalf("peer status reason = %q, want timeout", status.Reason)
	}
	if host.RemoteConnected() {
		t.Fatal("host remote connected = true, want false")
	}
}

func startLoopbackHost(t *testing.T) *Host {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	host, err := ListenHost(HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 4242,
	})
	if err != nil {
		cancel()
		t.Fatalf("ListenHost() error = %v", err)
	}

	errc := make(chan error, 1)
	go func() {
		errc <- host.Serve(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		_ = host.Close()
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("host Serve() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("host did not stop")
		}
	})

	return host
}

func dialLoopbackClient(t *testing.T, host *Host) *Client {
	t.Helper()

	client, err := DialClient(ClientOptions{
		LocalAddr:    "127.0.0.1:0",
		HostAddr:     host.Addr().String(),
		PlayerName:   "joiner",
		BuildVersion: "test",
	})
	if err != nil {
		t.Fatalf("DialClient() error = %v", err)
	}
	return client
}

func fastSessionOptions() SessionOptions {
	return SessionOptions{
		SimulationRate: 5 * time.Millisecond,
		SnapshotRate:   5 * time.Millisecond,
		HeartbeatRate:  5 * time.Millisecond,
		PeerTimeout:    250 * time.Millisecond,
	}
}

func waitForSnapshot(t *testing.T, snapshots <-chan game.Snapshot, want func(game.Snapshot) bool, errors ...<-chan error) game.Snapshot {
	t.Helper()

	deadline := time.After(time.Second)
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()

	for {
		if err := pollSessionErrors(errors...); err != nil {
			t.Fatalf("session error = %v", err)
		}

		select {
		case snapshot := <-snapshots:
			if want(snapshot) {
				return snapshot
			}
		case <-poll.C:
		case <-deadline:
			t.Fatal("timed out waiting for snapshot")
		}
	}
}

func waitForPeerStatus(t *testing.T, statuses <-chan PeerStatus, connected bool, errors ...<-chan error) PeerStatus {
	t.Helper()

	deadline := time.After(time.Second)
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()

	for {
		if err := pollSessionErrors(errors...); err != nil {
			t.Fatalf("session error = %v", err)
		}

		select {
		case status := <-statuses:
			if status.Connected == connected {
				return status
			}
		case <-poll.C:
		case <-deadline:
			t.Fatalf("timed out waiting for peer connected=%v", connected)
		}
	}
}

func waitForCondition(t *testing.T, want func() bool, errors ...<-chan error) {
	t.Helper()

	deadline := time.After(time.Second)
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()

	for {
		if err := pollSessionErrors(errors...); err != nil {
			t.Fatalf("session error = %v", err)
		}
		if want() {
			return
		}

		select {
		case <-poll.C:
		case <-deadline:
			t.Fatal("timed out waiting for condition")
		}
	}
}

func pollSessionErrors(errors ...<-chan error) error {
	for _, errors := range errors {
		select {
		case err := <-errors:
			return err
		default:
		}
	}
	return nil
}

func sendSessionInput(t *testing.T, inputs chan<- game.InputCommand, command game.InputCommand) {
	t.Helper()

	select {
	case inputs <- command:
	case <-time.After(time.Second):
		t.Fatal("timed out sending session input")
	}
}

func newValidationHost(t *testing.T, remote *net.UDPAddr, tick uint64) *Host {
	t.Helper()

	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}
	state.Tick = tick
	return &Host{
		sessionID: 99,
		state:     state,
		remote:    remote,
	}
}

func remoteInputPacket(tick uint64, sequence uint32) (protocol.Packet, game.InputCommand) {
	command := game.NewInputCommand(tick, game.PlayerTwo)
	command.Sequence = sequence
	packet := protocol.Packet{
		Type:      protocol.PacketInput,
		SessionID: 99,
		Sequence:  sequence,
		Tick:      tick,
		Payload:   protocol.InputPayload{Command: command},
	}
	return packet, command
}
