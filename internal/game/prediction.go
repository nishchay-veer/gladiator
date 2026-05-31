package game

const defaultPredictionHistoryCapacity = 180

type PredictionEntry struct {
	Command  InputCommand
	Snapshot Snapshot
}

type PredictionHistory struct {
	capacity int
	entries  []PredictionEntry
}

type ReconciliationResult struct {
	State               State
	AuthoritativeTick   uint64
	TargetTick          uint64
	ReplayedCommands    int
	HadPrediction       bool
	NeedsCorrection     bool
	PredictedPlayer     PlayerSnapshot
	AuthoritativePlayer PlayerSnapshot
}

func NewPredictionHistory(capacity int) *PredictionHistory {
	if capacity <= 0 {
		capacity = defaultPredictionHistoryCapacity
	}
	return &PredictionHistory{capacity: capacity}
}

func (h *PredictionHistory) Record(command InputCommand, snapshot Snapshot) {
	if h == nil {
		return
	}

	h.entries = append(h.entries, PredictionEntry{
		Command:  command,
		Snapshot: snapshot,
	})
	if len(h.entries) > h.capacity {
		copy(h.entries, h.entries[len(h.entries)-h.capacity:])
		h.entries = h.entries[:h.capacity]
	}
}

func (h *PredictionHistory) Reset() {
	if h == nil {
		return
	}
	h.entries = h.entries[:0]
}

func (h *PredictionHistory) Len() int {
	if h == nil {
		return 0
	}
	return len(h.entries)
}

func (h *PredictionHistory) SnapshotAt(tick uint64) (Snapshot, bool) {
	if h == nil {
		return Snapshot{}, false
	}
	for i := len(h.entries) - 1; i >= 0; i-- {
		if h.entries[i].Snapshot.Tick == tick {
			return h.entries[i].Snapshot, true
		}
	}
	return Snapshot{}, false
}

func (h *PredictionHistory) LatestTick() (uint64, bool) {
	if h == nil || len(h.entries) == 0 {
		return 0, false
	}
	return h.entries[len(h.entries)-1].Snapshot.Tick, true
}

func (h *PredictionHistory) Reconcile(base State, authoritative Snapshot, localPlayer PlayerID) ReconciliationResult {
	result := ReconciliationResult{
		State:             base.ApplySnapshot(authoritative),
		AuthoritativeTick: authoritative.Tick,
		TargetTick:        authoritative.Tick,
	}

	index, ok := playerIndex(localPlayer)
	if !ok {
		return result
	}
	result.AuthoritativePlayer = authoritative.Players[index]

	if predicted, found := h.SnapshotAt(authoritative.Tick); found {
		result.HadPrediction = true
		result.PredictedPlayer = predicted.Players[index]
		result.NeedsCorrection = predicted.Players[index] != authoritative.Players[index]
	}

	targetTick, found := h.LatestTick()
	if found && targetTick > result.TargetTick {
		result.TargetTick = targetTick
	}

	if h == nil {
		return result
	}
	for _, entry := range h.entries {
		if entry.Command.PlayerID != localPlayer || entry.Command.Tick < authoritative.Tick {
			continue
		}
		if entry.Command.Tick > result.TargetTick {
			break
		}

		for result.State.Tick < entry.Command.Tick && result.State.Tick < result.TargetTick {
			result.State.Step(NewInputFrame(result.State.Tick))
		}
		if result.State.Tick != entry.Command.Tick || result.State.Tick >= result.TargetTick {
			continue
		}

		command := entry.Command
		command.Tick = result.State.Tick
		frame := NewInputFrame(result.State.Tick)
		frame.Set(command)
		result.State.Step(frame)
		result.ReplayedCommands++
	}

	for result.State.Tick < result.TargetTick {
		result.State.Step(NewInputFrame(result.State.Tick))
	}

	return result
}

func playerIndex(id PlayerID) (int, bool) {
	switch id {
	case PlayerOne:
		return 0, true
	case PlayerTwo:
		return 1, true
	default:
		return 0, false
	}
}
