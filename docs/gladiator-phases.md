# Gladiator Build Phases

This roadmap turns the architecture into execution phases. The main rule: each phase should produce something runnable or testable before we widen the scope.

## Phase 0: Project Foundation

Goal: create the project skeleton and prove the terminal can support the visual style.

Deliverables:

- Go module, `cmd/gladiator`, and initial `internal/...` package layout.
- Basic CLI commands: `play-local`, `host`, `join`, `version`.
- Logging and config defaults.
- Terminal capability spike with tcell: truecolor, alt screen, keyboard input, resize handling.
- One hardcoded arena drawn in the screenshot style.

Done when:

- `go run ./cmd/gladiator play-local` opens a terminal arena and exits cleanly.
- We know the minimum terminal size and best-supported terminals.
- `go test ./...` passes.

Avoid:

- LAN networking.
- Full art pipeline.
- Complex gameplay.

## Phase 1: Local Playable Prototype

Goal: make the game feel real on one machine before networking complicates it.

Deliverables:

- Fixed-step loop at 60 Hz.
- Player movement, facing, collision against walls, and screen bounds.
- Basic HUD: player health, ammo/cooldown, score placeholder, FPS.
- Keyboard controls with configurable keymap.
- Simple bullets, hit detection, damage, death, and respawn.

Done when:

- One local player can move, shoot, hit walls, die, and respawn.
- The arena renders without flicker on supported terminals.
- Game logic is separated from rendering.

Avoid:

- Network prediction.
- Multiple maps.
- Fancy animation beyond what helps prove the look.

## Phase 2: Deterministic Game Core

Goal: make the simulation reliable enough to drive multiplayer.

Deliverables:

- `internal/game` owns all match state.
- Fixed-point or integer world coordinates.
- Pure-ish tick function: previous state + inputs -> next state.
- Player, projectile, wall, spawn, score, cooldown, and match timer systems.
- Golden tick tests for movement, shooting, collision, hits, and respawns.
- Snapshot structs that can be rendered without mutating game state.

Done when:

- The same input log produces the same match result every run.
- Game tests cover the scary parts: collision, firing cadence, deaths, and respawns.
- Render code can be swapped without changing simulation code.

Avoid:

- Real network transport.
- Serialization optimization.
- Extra weapons or powerups.

## Phase 3: LAN Vertical Slice

Goal: get two machines, or two terminals on one machine, playing a crude but real multiplayer match.

Deliverables:

- `host` starts an authoritative UDP server and local player.
- `join <ip>` connects as player 2.
- Manual IP flow first; discovery comes later.
- Handshake with protocol version, map id/hash, player id, and ready state.
- Client sends `InputCmd` packets every simulation tick.
- Host sends authoritative snapshots 20-30 times per second.
- Joiner interpolates remote state and applies basic correction for local player.

Done when:

- Two terminals can move and shoot each other over localhost.
- Two machines on the same LAN can play by IP address.
- Disconnects and timeouts do not crash the process.

Avoid:

- Matchmaking.
- Internet play.
- Perfect lag compensation.

## Phase 4: Netcode Hardening

Goal: make LAN multiplayer stable instead of merely lucky.

Deliverables:

- Sequence numbers, ack bitfields, packet stats, and duplicate/drop handling.
- Tiny reliable lane for lobby, ready, map, match start, and match end events.
- Jitter/loss simulator for local testing.
- Debug overlay: ping, packet loss, remote tick, local tick drift, snapshot age.
- Client-side prediction history and smoother reconciliation.
- Server validation for movement, fire rate, collisions, and damage.

Done when:

- The game remains playable under simulated LAN packet loss and jitter.
- Bad or out-of-order packets are ignored safely.
- Debug overlay makes network problems visible.

Avoid:

- Heavy networking libraries unless the homegrown layer becomes a problem.
- Anti-cheat beyond server authority and validation.

## Phase 5: Visual Polish and Asset Pipeline

Goal: get close to the screenshot’s polished terminal pixel look.

Deliverables:

- Palette, tile, and glyph rules for the arena style.
- Asset converter for maps and optional sprite metadata.
- Better walls, floor noise, spawn markers, projectiles, muzzle flashes, hit effects, and death effects.
- Multiple arena layouts with embedded assets.
- Terminal fallbacks: truecolor, 256-color, and monochrome-ish emergency mode.
- Optional simple audio toggle.

Done when:

- The game has a recognizable visual identity.
- New maps can be added without editing Go code manually.
- The terminal output still performs well and remains readable.

Avoid:

- Native GUI work unless terminal graphics hit a hard ceiling.
- Too many maps before one map feels excellent.

## Phase 6: UX, Packaging, and Release

Goal: make it easy for someone else to run the game on macOS and other machines.

Deliverables:

- Lobby flow: host, discover, join, ready, start match, rematch, quit.
- UDP broadcast or mDNS discovery, with manual IP fallback.
- Config file for controls, player name, renderer settings, and network port.
- CI: tests, race tests where useful, lint, and builds.
- Cross-platform binaries for macOS, Linux, and Windows.
- macOS notes for firewall prompt, permissions, and later codesigning.

Done when:

- A friend on the same Wi-Fi can download/run the binary and join a match.
- Release artifacts are reproducible.
- Basic troubleshooting is documented.

Avoid:

- App Store packaging.
- Online account systems.
- Premature installers.

## Phase 7: Expansion

Goal: add bigger features after the core loop is strong.

Options:

- Native Ebitengine frontend using the same `internal/game` simulation.
- Bots for local practice.
- Replays using recorded input streams.
- More weapons, pickups, hazards, and game modes.
- Spectator mode.
- Internet play via relay/NAT traversal.

Done when:

- Each expansion ships behind its own small milestone.
- Terminal LAN play remains the stable baseline.

## Suggested Immediate Next Step

Start Phase 0 with a minimal Go project and terminal rendering spike. The first concrete target should be:

```sh
go run ./cmd/gladiator play-local
```

That command should open an alternate-screen terminal arena, render the screenshot-like border/floor/HUD, move one player with the keyboard, and quit cleanly.
