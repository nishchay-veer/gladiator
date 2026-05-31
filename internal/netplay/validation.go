package netplay

import (
	"gladiator/internal/game"
	"gladiator/internal/protocol"
)

const (
	maxRemoteInputLagTicks  uint64 = 30
	maxRemoteInputLeadTicks uint64 = 30
)

func (h *Host) validateRemoteInputLocked(packet protocol.Packet, command game.InputCommand) bool {
	if packet.Type != protocol.PacketInput {
		return false
	}
	if packet.Tick != command.Tick || packet.Sequence != command.Sequence {
		return false
	}
	if command.PlayerID != game.PlayerTwo {
		return false
	}
	if err := command.Validate(command.Tick); err != nil {
		return false
	}
	return inputTickInWindow(h.state.Tick, command.Tick)
}

func inputTickInWindow(authoritativeTick, inputTick uint64) bool {
	if inputTick > authoritativeTick {
		return inputTick-authoritativeTick <= maxRemoteInputLeadTicks
	}
	return authoritativeTick-inputTick <= maxRemoteInputLagTicks
}
