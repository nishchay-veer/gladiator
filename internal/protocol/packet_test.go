package protocol

import (
	"strings"
	"testing"

	"github.com/nishchay-veer/gladiator/internal/game"
)

func TestInputPacketRoundTrip(t *testing.T) {
	command := game.InputCommand{
		Tick:     42,
		PlayerID: game.PlayerTwo,
		Sequence: 99,
		MoveX:    -1,
		Aim:      game.Left,
		HasAim:   true,
		Buttons:  game.ButtonFire,
	}
	packet := Packet{
		Type:      PacketInput,
		SessionID: 1234,
		Sequence:  99,
		Ack:       41,
		AckBits:   0b101,
		Tick:      command.Tick,
		Payload:   InputPayload{Command: command},
	}

	got := roundTripPacket(t, packet)
	if got.Type != packet.Type || got.SessionID != packet.SessionID || got.Sequence != packet.Sequence || got.Ack != packet.Ack || got.AckBits != packet.AckBits || got.Tick != packet.Tick {
		t.Fatalf("packet header mismatch\ngot:  %#v\nwant: %#v", got, packet)
	}

	payload, ok := got.Payload.(InputPayload)
	if !ok {
		t.Fatalf("payload type = %T, want InputPayload", got.Payload)
	}
	if payload.Command != command {
		t.Fatalf("command = %#v, want %#v", payload.Command, command)
	}
}

func TestSnapshotPacketRoundTrip(t *testing.T) {
	snapshot := testSnapshot(t)
	packet := Packet{
		Type:      PacketSnapshot,
		SessionID: 55,
		Sequence:  7,
		Tick:      snapshot.Tick,
		Payload:   SnapshotPayload{Snapshot: snapshot},
	}

	got := roundTripPacket(t, packet)
	payload, ok := got.Payload.(SnapshotPayload)
	if !ok {
		t.Fatalf("payload type = %T, want SnapshotPayload", got.Payload)
	}
	if !payload.Snapshot.Equal(snapshot) {
		t.Fatalf("snapshot mismatch\ngot:  %#v\nwant: %#v", payload.Snapshot, snapshot)
	}
}

func TestWelcomePacketRoundTrip(t *testing.T) {
	snapshot := testSnapshot(t)
	packet := Packet{
		Type:      PacketWelcome,
		SessionID: 55,
		Sequence:  1,
		Tick:      snapshot.Tick,
		Payload: WelcomePayload{
			PlayerID:       game.PlayerTwo,
			HostPlayerName: "hosty",
			MapID:          snapshot.Match.MapID,
			MapHash:        12345,
			Ready:          true,
			Snapshot:       snapshot,
		},
	}

	got := roundTripPacket(t, packet)
	payload, ok := got.Payload.(WelcomePayload)
	if !ok {
		t.Fatalf("payload type = %T, want WelcomePayload", got.Payload)
	}
	if payload.PlayerID != game.PlayerTwo {
		t.Fatalf("player id = %d, want %d", payload.PlayerID, game.PlayerTwo)
	}
	if payload.HostPlayerName != "hosty" {
		t.Fatalf("host player name = %q, want hosty", payload.HostPlayerName)
	}
	if payload.MapID != snapshot.Match.MapID {
		t.Fatalf("map id = %q, want %q", payload.MapID, snapshot.Match.MapID)
	}
	if payload.MapHash != 12345 {
		t.Fatalf("map hash = %d, want 12345", payload.MapHash)
	}
	if !payload.Ready {
		t.Fatal("ready = false, want true")
	}
	if !payload.Snapshot.Equal(snapshot) {
		t.Fatalf("snapshot mismatch\ngot:  %#v\nwant: %#v", payload.Snapshot, snapshot)
	}
}

func TestSmallPayloadPacketRoundTrips(t *testing.T) {
	tests := []struct {
		name    string
		packet  Packet
		payload Payload
	}{
		{
			name: "hello",
			packet: Packet{
				Type:      PacketHello,
				SessionID: 0,
				Sequence:  1,
				Payload: HelloPayload{
					PlayerName:   "nish",
					BuildVersion: "1.1.0",
					Ready:        true,
				},
			},
			payload: HelloPayload{PlayerName: "nish", BuildVersion: "1.1.0", Ready: true},
		},
		{
			name: "ping",
			packet: Packet{
				Type:      PacketPing,
				SessionID: 77,
				Sequence:  2,
				Tick:      12,
				Payload:   PingPayload{SentAtUnixMillis: 1770000000000},
			},
			payload: PingPayload{SentAtUnixMillis: 1770000000000},
		},
		{
			name: "disconnect",
			packet: Packet{
				Type:      PacketDisconnect,
				SessionID: 77,
				Sequence:  3,
				Tick:      12,
				Payload:   DisconnectPayload{Reason: "quit"},
			},
			payload: DisconnectPayload{Reason: "quit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundTripPacket(t, tt.packet)
			if got.Payload != tt.payload {
				t.Fatalf("payload = %#v, want %#v", got.Payload, tt.payload)
			}
		})
	}
}

func TestDecodeRejectsInvalidPacket(t *testing.T) {
	valid, err := Encode(Packet{
		Type:      PacketPing,
		SessionID: 1,
		Sequence:  1,
		Tick:      1,
		Payload:   PingPayload{SentAtUnixMillis: 1},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{name: "short", data: valid[:packetHeaderSize-1]},
		{name: "bad magic", data: mutated(valid, 0, 'X')},
		{name: "bad version", data: mutated(valid, 4, Version+1)},
		{name: "unknown type", data: mutated(valid, 5, 200)},
		{name: "trailing bytes", data: append(append([]byte(nil), valid...), 0xFF)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decode(tt.data); err == nil {
				t.Fatal("Decode() error = nil, want error")
			}
		})
	}
}

func TestEncodeRejectsInvalidPacket(t *testing.T) {
	tests := []struct {
		name   string
		packet Packet
	}{
		{
			name: "nil payload",
			packet: Packet{
				Type: PacketPing,
			},
		},
		{
			name: "payload mismatch",
			packet: Packet{
				Type:    PacketPing,
				Payload: HelloPayload{PlayerName: "p1"},
			},
		},
		{
			name: "input tick mismatch",
			packet: Packet{
				Type: PacketInput,
				Tick: 10,
				Payload: InputPayload{Command: game.InputCommand{
					Tick:     11,
					PlayerID: game.PlayerOne,
				}},
			},
		},
		{
			name: "input sequence mismatch",
			packet: Packet{
				Type:     PacketInput,
				Sequence: 10,
				Tick:     10,
				Payload: InputPayload{Command: game.InputCommand{
					Tick:     10,
					PlayerID: game.PlayerOne,
					Sequence: 11,
				}},
			},
		},
		{
			name: "oversized string",
			packet: Packet{
				Type: PacketHello,
				Payload: HelloPayload{
					PlayerName: strings.Repeat("x", maxStringBytes+1),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Encode(tt.packet); err == nil {
				t.Fatal("Encode() error = nil, want error")
			}
		})
	}
}

func roundTripPacket(t *testing.T, packet Packet) Packet {
	t.Helper()

	data, err := Encode(packet)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if len(data) > MaxPacketSize {
		t.Fatalf("encoded packet size = %d, max %d", len(data), MaxPacketSize)
	}

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return got
}

func testSnapshot(t *testing.T) game.Snapshot {
	t.Helper()

	state, err := game.NewLocalState()
	if err != nil {
		t.Fatalf("NewLocalState() error = %v", err)
	}

	frame := game.NewInputFrame(state.Tick)
	command := game.NewInputCommand(state.Tick, game.PlayerOne)
	command.Buttons = game.ButtonFire
	frame.Set(command)
	state.Step(frame)

	state.Step(game.NewInputFrame(state.Tick))
	return state.Snapshot()
}

func mutated(data []byte, index int, value byte) []byte {
	clone := append([]byte(nil), data...)
	clone[index] = value
	return clone
}
