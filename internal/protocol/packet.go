package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nishchay-veer/gladiator/internal/game"
)

const (
	Version       uint8 = 1
	MaxPacketSize       = 1200

	maxStringBytes      = 256
	maxBulletsPerPacket = 256
	minInt32            = -1 << 31
	maxInt32            = 1<<31 - 1
	packetHeaderSize    = 34
)

var packetMagic = [4]byte{'G', 'L', 'A', 'D'}

type PacketType uint8

const (
	PacketHello PacketType = iota + 1
	PacketWelcome
	PacketInput
	PacketSnapshot
	PacketPing
	PacketDisconnect
)

type Packet struct {
	Type      PacketType
	SessionID uint64
	Sequence  uint32
	Ack       uint32
	AckBits   uint32
	Tick      uint64
	Payload   Payload
}

type Payload interface {
	packetPayload()
}

type HelloPayload struct {
	PlayerName   string
	BuildVersion string
	Ready        bool
}

type WelcomePayload struct {
	PlayerID game.PlayerID
	MapID    string
	MapHash  uint64
	Ready    bool
	Snapshot game.Snapshot
}

type InputPayload struct {
	Command game.InputCommand
}

type SnapshotPayload struct {
	Snapshot game.Snapshot
}

type PingPayload struct {
	SentAtUnixMillis int64
}

type DisconnectPayload struct {
	Reason string
}

func (HelloPayload) packetPayload()      {}
func (WelcomePayload) packetPayload()    {}
func (InputPayload) packetPayload()      {}
func (SnapshotPayload) packetPayload()   {}
func (PingPayload) packetPayload()       {}
func (DisconnectPayload) packetPayload() {}

func Encode(packet Packet) ([]byte, error) {
	if err := packet.validate(); err != nil {
		return nil, err
	}

	writer := packetWriter{}
	writer.bytes(packetMagic[:])
	writer.u8(Version)
	writer.u8(uint8(packet.Type))
	writer.u64(packet.SessionID)
	writer.u32(packet.Sequence)
	writer.u32(packet.Ack)
	writer.u32(packet.AckBits)
	writer.u64(packet.Tick)
	writePayload(&writer, packet)

	if writer.err != nil {
		return nil, writer.err
	}
	if writer.bufLen() > MaxPacketSize {
		return nil, fmt.Errorf("packet is %d bytes, max is %d", writer.bufLen(), MaxPacketSize)
	}

	return writer.buf, nil
}

func Decode(data []byte) (Packet, error) {
	if len(data) > MaxPacketSize {
		return Packet{}, fmt.Errorf("packet is %d bytes, max is %d", len(data), MaxPacketSize)
	}
	if len(data) < packetHeaderSize {
		return Packet{}, fmt.Errorf("packet too short: %d bytes", len(data))
	}

	reader := packetReader{data: data}
	if got := reader.bytes(len(packetMagic)); string(got) != string(packetMagic[:]) {
		return Packet{}, errors.New("invalid packet magic")
	}
	if version := reader.u8(); version != Version {
		return Packet{}, fmt.Errorf("unsupported protocol version %d", version)
	}

	packet := Packet{
		Type:      PacketType(reader.u8()),
		SessionID: reader.u64(),
		Sequence:  reader.u32(),
		Ack:       reader.u32(),
		AckBits:   reader.u32(),
		Tick:      reader.u64(),
	}

	payload, err := readPayload(&reader, packet)
	if err != nil {
		return Packet{}, err
	}
	if reader.err != nil {
		return Packet{}, reader.err
	}
	if !reader.done() {
		return Packet{}, fmt.Errorf("packet has %d trailing bytes", reader.remaining())
	}

	packet.Payload = payload
	if err := packet.validate(); err != nil {
		return Packet{}, err
	}
	return packet, nil
}

func (p Packet) validate() error {
	if !validPacketType(p.Type) {
		return fmt.Errorf("unknown packet type %d", p.Type)
	}
	if p.Payload == nil {
		return errors.New("packet payload is nil")
	}

	switch payload := p.Payload.(type) {
	case HelloPayload:
		if p.Type != PacketHello {
			return payloadMismatch(p.Type, payload)
		}
	case WelcomePayload:
		if p.Type != PacketWelcome {
			return payloadMismatch(p.Type, payload)
		}
		if !validPlayerID(payload.PlayerID) {
			return fmt.Errorf("invalid welcome player id %d", payload.PlayerID)
		}
		if payload.MapID != payload.Snapshot.Match.MapID {
			return fmt.Errorf("welcome map id %q does not match snapshot map id %q", payload.MapID, payload.Snapshot.Match.MapID)
		}
		if payload.Snapshot.Tick != p.Tick {
			return fmt.Errorf("welcome snapshot tick %d does not match packet tick %d", payload.Snapshot.Tick, p.Tick)
		}
	case InputPayload:
		if p.Type != PacketInput {
			return payloadMismatch(p.Type, payload)
		}
		if payload.Command.Tick != p.Tick {
			return fmt.Errorf("input command tick %d does not match packet tick %d", payload.Command.Tick, p.Tick)
		}
		if payload.Command.Sequence != p.Sequence {
			return fmt.Errorf("input command sequence %d does not match packet sequence %d", payload.Command.Sequence, p.Sequence)
		}
		if err := payload.Command.Validate(p.Tick); err != nil {
			return err
		}
	case SnapshotPayload:
		if p.Type != PacketSnapshot {
			return payloadMismatch(p.Type, payload)
		}
		if payload.Snapshot.Tick != p.Tick {
			return fmt.Errorf("snapshot tick %d does not match packet tick %d", payload.Snapshot.Tick, p.Tick)
		}
	case PingPayload:
		if p.Type != PacketPing {
			return payloadMismatch(p.Type, payload)
		}
	case DisconnectPayload:
		if p.Type != PacketDisconnect {
			return payloadMismatch(p.Type, payload)
		}
	default:
		return fmt.Errorf("unsupported payload type %T", p.Payload)
	}

	return nil
}

func writePayload(writer *packetWriter, packet Packet) {
	switch payload := packet.Payload.(type) {
	case HelloPayload:
		writer.string(payload.PlayerName)
		writer.string(payload.BuildVersion)
		writer.bool(payload.Ready)
	case WelcomePayload:
		writer.playerID(payload.PlayerID)
		writer.string(payload.MapID)
		writer.u64(payload.MapHash)
		writer.bool(payload.Ready)
		writer.snapshot(payload.Snapshot)
	case InputPayload:
		writer.command(payload.Command)
	case SnapshotPayload:
		writer.snapshot(payload.Snapshot)
	case PingPayload:
		writer.i64(payload.SentAtUnixMillis)
	case DisconnectPayload:
		writer.string(payload.Reason)
	}
}

func readPayload(reader *packetReader, packet Packet) (Payload, error) {
	switch packet.Type {
	case PacketHello:
		return HelloPayload{
			PlayerName:   reader.string(),
			BuildVersion: reader.string(),
			Ready:        reader.bool(),
		}, nil
	case PacketWelcome:
		return WelcomePayload{
			PlayerID: reader.playerID(),
			MapID:    reader.string(),
			MapHash:  reader.u64(),
			Ready:    reader.bool(),
			Snapshot: reader.snapshot(),
		}, nil
	case PacketInput:
		return InputPayload{Command: reader.command()}, nil
	case PacketSnapshot:
		return SnapshotPayload{Snapshot: reader.snapshot()}, nil
	case PacketPing:
		return PingPayload{SentAtUnixMillis: reader.i64()}, nil
	case PacketDisconnect:
		return DisconnectPayload{Reason: reader.string()}, nil
	default:
		return nil, fmt.Errorf("unknown packet type %d", packet.Type)
	}
}

func validPacketType(packetType PacketType) bool {
	return packetType >= PacketHello && packetType <= PacketDisconnect
}

func validPlayerID(id game.PlayerID) bool {
	return id == game.PlayerOne || id == game.PlayerTwo
}

func payloadMismatch(packetType PacketType, payload Payload) error {
	return fmt.Errorf("packet type %d does not match payload %T", packetType, payload)
}

type packetWriter struct {
	buf []byte
	err error
}

func (w *packetWriter) bufLen() int {
	return len(w.buf)
}

func (w *packetWriter) bytes(value []byte) {
	if w.err != nil {
		return
	}
	w.buf = append(w.buf, value...)
}

func (w *packetWriter) u8(value uint8) {
	w.bytes([]byte{value})
}

func (w *packetWriter) u16(value uint16) {
	var buffer [2]byte
	binary.BigEndian.PutUint16(buffer[:], value)
	w.bytes(buffer[:])
}

func (w *packetWriter) u32(value uint32) {
	var buffer [4]byte
	binary.BigEndian.PutUint32(buffer[:], value)
	w.bytes(buffer[:])
}

func (w *packetWriter) u64(value uint64) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], value)
	w.bytes(buffer[:])
}

func (w *packetWriter) i64(value int64) {
	w.u64(uint64(value))
}

func (w *packetWriter) i32(value int) {
	if value < minInt32 || value > maxInt32 {
		w.err = fmt.Errorf("int value %d cannot fit in int32", value)
		return
	}
	w.u32(uint32(int32(value)))
}

func (w *packetWriter) bool(value bool) {
	if value {
		w.u8(1)
		return
	}
	w.u8(0)
}

func (w *packetWriter) string(value string) {
	if len(value) > maxStringBytes {
		w.err = fmt.Errorf("string is %d bytes, max is %d", len(value), maxStringBytes)
		return
	}
	w.u16(uint16(len(value)))
	w.bytes([]byte(value))
}

func (w *packetWriter) playerID(id game.PlayerID) {
	w.u8(uint8(id))
}

func (w *packetWriter) direction(direction game.Direction) {
	w.u8(uint8(direction))
}

func (w *packetWriter) point(point game.Point) {
	w.i32(point.X)
	w.i32(point.Y)
}

func (w *packetWriter) command(command game.InputCommand) {
	w.u64(command.Tick)
	w.playerID(command.PlayerID)
	w.u32(command.Sequence)
	w.i32(command.MoveX)
	w.i32(command.MoveY)
	w.direction(command.Aim)
	w.bool(command.HasAim)
	w.u8(uint8(command.Buttons))
}

func (w *packetWriter) snapshot(snapshot game.Snapshot) {
	w.u64(snapshot.Tick)
	w.string(snapshot.Match.MapID)
	w.string(snapshot.Match.Mode)
	w.u64(snapshot.Match.TimeLimitTicks)
	w.i32(snapshot.Match.ScoreLimit)
	w.u64(snapshot.Match.ElapsedTicks)
	w.bool(snapshot.Match.Over)
	w.playerID(snapshot.Match.Winner)
	w.bool(snapshot.Match.HasWinner)

	for _, player := range snapshot.Players {
		w.playerID(player.ID)
		w.point(player.Position)
		w.direction(player.Facing)
		w.i32(player.Health)
		w.i32(player.MaxHealth)
		w.i32(player.Score)
		w.i32(player.FireCooldown)
		w.i32(player.RespawnTicks)
		w.bool(player.Alive)
	}

	if len(snapshot.Bullets) > maxBulletsPerPacket {
		w.err = fmt.Errorf("snapshot has %d bullets, max is %d", len(snapshot.Bullets), maxBulletsPerPacket)
		return
	}
	w.u16(uint16(len(snapshot.Bullets)))
	for _, bullet := range snapshot.Bullets {
		w.point(bullet.Position)
		w.direction(bullet.Direction)
		w.playerID(bullet.Owner)
		w.i32(bullet.Age)
		w.i32(bullet.TTL)
	}
}

type packetReader struct {
	data   []byte
	offset int
	err    error
}

func (r *packetReader) remaining() int {
	return len(r.data) - r.offset
}

func (r *packetReader) done() bool {
	return r.remaining() == 0
}

func (r *packetReader) bytes(size int) []byte {
	if r.err != nil {
		return nil
	}
	if size < 0 || r.remaining() < size {
		r.err = fmt.Errorf("packet ended early: need %d bytes, have %d", size, r.remaining())
		return nil
	}
	value := r.data[r.offset : r.offset+size]
	r.offset += size
	return value
}

func (r *packetReader) u8() uint8 {
	value := r.bytes(1)
	if value == nil {
		return 0
	}
	return value[0]
}

func (r *packetReader) u16() uint16 {
	value := r.bytes(2)
	if value == nil {
		return 0
	}
	return binary.BigEndian.Uint16(value)
}

func (r *packetReader) u32() uint32 {
	value := r.bytes(4)
	if value == nil {
		return 0
	}
	return binary.BigEndian.Uint32(value)
}

func (r *packetReader) u64() uint64 {
	value := r.bytes(8)
	if value == nil {
		return 0
	}
	return binary.BigEndian.Uint64(value)
}

func (r *packetReader) i64() int64 {
	return int64(r.u64())
}

func (r *packetReader) i32() int {
	return int(int32(r.u32()))
}

func (r *packetReader) bool() bool {
	value := r.u8()
	switch value {
	case 0:
		return false
	case 1:
		return true
	default:
		r.err = fmt.Errorf("invalid bool byte %d", value)
		return false
	}
}

func (r *packetReader) string() string {
	size := int(r.u16())
	if size > maxStringBytes {
		r.err = fmt.Errorf("string is %d bytes, max is %d", size, maxStringBytes)
		return ""
	}
	return string(r.bytes(size))
}

func (r *packetReader) playerID() game.PlayerID {
	return game.PlayerID(r.u8())
}

func (r *packetReader) direction() game.Direction {
	return game.Direction(r.u8())
}

func (r *packetReader) point() game.Point {
	return game.Point{X: r.i32(), Y: r.i32()}
}

func (r *packetReader) command() game.InputCommand {
	return game.InputCommand{
		Tick:     r.u64(),
		PlayerID: r.playerID(),
		Sequence: r.u32(),
		MoveX:    r.i32(),
		MoveY:    r.i32(),
		Aim:      r.direction(),
		HasAim:   r.bool(),
		Buttons:  game.Buttons(r.u8()),
	}
}

func (r *packetReader) snapshot() game.Snapshot {
	snapshot := game.Snapshot{
		Tick: r.u64(),
		Match: game.MatchSnapshot{
			MapID:          r.string(),
			Mode:           r.string(),
			TimeLimitTicks: r.u64(),
			ScoreLimit:     r.i32(),
			ElapsedTicks:   r.u64(),
			Over:           r.bool(),
			Winner:         r.playerID(),
			HasWinner:      r.bool(),
		},
	}

	for i := range snapshot.Players {
		snapshot.Players[i] = game.PlayerSnapshot{
			ID:           r.playerID(),
			Position:     r.point(),
			Facing:       r.direction(),
			Health:       r.i32(),
			MaxHealth:    r.i32(),
			Score:        r.i32(),
			FireCooldown: r.i32(),
			RespawnTicks: r.i32(),
			Alive:        r.bool(),
		}
	}

	bulletCount := int(r.u16())
	if bulletCount > maxBulletsPerPacket {
		r.err = fmt.Errorf("snapshot has %d bullets, max is %d", bulletCount, maxBulletsPerPacket)
		return snapshot
	}
	snapshot.Bullets = make([]game.BulletSnapshot, bulletCount)
	for i := range snapshot.Bullets {
		snapshot.Bullets[i] = game.BulletSnapshot{
			Position:  r.point(),
			Direction: r.direction(),
			Owner:     r.playerID(),
			Age:       r.i32(),
			TTL:       r.i32(),
		}
	}

	return snapshot
}
