package netplay

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/nishchay-veer/gladiator/internal/game"
	"github.com/nishchay-veer/gladiator/internal/protocol"
)

type ClientOptions struct {
	LocalAddr    string
	HostAddr     string
	PlayerName   string
	BuildVersion string
}

type Client struct {
	conn     *net.UDPConn
	hostAddr *net.UDPAddr

	playerName   string
	buildVersion string

	mu        sync.Mutex
	sessionID uint64
	playerID  game.PlayerID
	sequence  uint32

	hostPackets packetWindow
}

type JoinResult struct {
	SessionID      uint64
	PlayerID       game.PlayerID
	HostPlayerName string
	MapID          string
	MapHash        uint64
	Ready          bool
	Snapshot       game.Snapshot
}

func DialClient(opts ClientOptions) (*Client, error) {
	if opts.HostAddr == "" {
		return nil, errors.New("host address is required")
	}

	localAddr := opts.LocalAddr
	if localAddr == "" {
		localAddr = ":0"
	}

	udpLocalAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	hostAddr, err := net.ResolveUDPAddr("udp", opts.HostAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpLocalAddr)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:         conn,
		hostAddr:     hostAddr,
		playerName:   opts.PlayerName,
		buildVersion: opts.BuildVersion,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Stats() NetworkStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.hostPackets.Stats()
}

func (c *Client) Join(ctx context.Context) (JoinResult, error) {
	sequence, ack, ackBits := c.nextPacketHeader()
	packet := protocol.Packet{
		Type:     protocol.PacketHello,
		Sequence: sequence,
		Ack:      ack,
		AckBits:  ackBits,
		Payload: protocol.HelloPayload{
			PlayerName:   c.playerName,
			BuildVersion: c.buildVersion,
			Ready:        true,
		},
	}
	if err := sendPacket(c.conn, c.hostAddr, packet); err != nil {
		return JoinResult{}, err
	}

	for {
		response, err := c.readFromHost(ctx)
		if err != nil {
			return JoinResult{}, err
		}
		if response.Type != protocol.PacketWelcome {
			continue
		}

		payload, ok := response.Payload.(protocol.WelcomePayload)
		if !ok {
			continue
		}

		c.mu.Lock()
		c.sessionID = response.SessionID
		c.playerID = payload.PlayerID
		c.mu.Unlock()

		return JoinResult{
			SessionID:      response.SessionID,
			PlayerID:       payload.PlayerID,
			HostPlayerName: payload.HostPlayerName,
			MapID:          payload.MapID,
			MapHash:        payload.MapHash,
			Ready:          payload.Ready,
			Snapshot:       payload.Snapshot,
		}, nil
	}
}

func (c *Client) SendInput(ctx context.Context, command game.InputCommand) (game.Snapshot, error) {
	c.mu.Lock()
	if c.sessionID == 0 {
		c.mu.Unlock()
		return game.Snapshot{}, errors.New("client has not joined a host")
	}
	sessionID := c.sessionID
	command.Sequence = c.nextSequenceLocked()
	ack, ackBits := c.hostPackets.Ack()
	packet := protocol.Packet{
		Type:      protocol.PacketInput,
		SessionID: sessionID,
		Sequence:  command.Sequence,
		Ack:       ack,
		AckBits:   ackBits,
		Tick:      command.Tick,
		Payload:   protocol.InputPayload{Command: command},
	}
	c.mu.Unlock()

	if err := sendPacket(c.conn, c.hostAddr, packet); err != nil {
		return game.Snapshot{}, err
	}

	for {
		response, observation, err := c.readPacketFromHost(ctx)
		if err != nil {
			return game.Snapshot{}, err
		}
		if response.Type != protocol.PacketSnapshot || response.SessionID != sessionID || !observation.Advanced {
			continue
		}

		payload, ok := response.Payload.(protocol.SnapshotPayload)
		if !ok {
			continue
		}
		return payload.Snapshot, nil
	}
}

func (c *Client) Disconnect(ctx context.Context, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	if c.sessionID == 0 {
		c.mu.Unlock()
		return nil
	}
	sessionID := c.sessionID
	sequence := c.nextSequenceLocked()
	ack, ackBits := c.hostPackets.Ack()
	c.sessionID = 0
	c.playerID = 0
	c.mu.Unlock()

	packet := protocol.Packet{
		Type:      protocol.PacketDisconnect,
		SessionID: sessionID,
		Sequence:  sequence,
		Ack:       ack,
		AckBits:   ackBits,
		Payload:   protocol.DisconnectPayload{Reason: reason},
	}
	return sendPacket(c.conn, c.hostAddr, packet)
}

func (c *Client) readFromHost(ctx context.Context) (protocol.Packet, error) {
	packet, _, err := c.readPacketFromHost(ctx)
	return packet, err
}

func (c *Client) readPacketFromHost(ctx context.Context) (protocol.Packet, packetObservation, error) {
	buffer := make([]byte, protocol.MaxPacketSize)

	for {
		data, addr, err := readDatagram(ctx, c.conn, buffer)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, net.ErrClosed) {
				return protocol.Packet{}, packetObservation{}, err
			}
			return protocol.Packet{}, packetObservation{}, err
		}
		if !sameUDPAddr(addr, c.hostAddr) {
			continue
		}

		packet, err := protocol.Decode(data)
		if err != nil {
			continue
		}

		c.mu.Lock()
		observation := c.hostPackets.Observe(packet.Sequence)
		c.mu.Unlock()

		return packet, observation, nil
	}
}

func (c *Client) nextSequenceLocked() uint32 {
	c.sequence++
	return c.sequence
}

func (c *Client) nextPacketHeader() (uint32, uint32, uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sequence := c.nextSequenceLocked()
	ack, ackBits := c.hostPackets.Ack()
	return sequence, ack, ackBits
}
