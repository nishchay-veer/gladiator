# Dev Log

## 2026-05-31

### Phase 0 Kickoff

- Started Phase 0 from `docs/gladiator-phases.md`.
- Local toolchain is Go `1.24.2`, so the initial module uses that local version instead of requiring a newer Go install before the first runnable prototype.
- Added `github.com/gdamore/tcell/v2` for the terminal renderer spike.
- Established the first target command: `go run ./cmd/gladiator play-local`.

### Phase 0 Skeleton and Terminal Spike

- Added `cmd/gladiator`, `internal/cli`, `internal/config`, `internal/build`, `internal/game`, and `internal/termui`.
- Added basic CLI commands: `play-local`, `host`, `join <ip>`, and `version`.
- `host` and `join` are intentional placeholders until the LAN vertical slice in Phase 3.
- Added a small `internal/logging` package around `log/slog` so later network/game debug output has one setup path.
- Added a hardcoded 40x17 arena with wall/floor tiles, spawn markers, and one movable player.
- Added a tcell renderer using alternate-screen terminal behavior, truecolor styles, resize handling, keyboard input, and clean quit with `q`, Esc, or Ctrl-C.
- Verified the TUI path in a pseudo-terminal. The default test terminal is `80x24`, so the Phase 0 minimum was set to `80x24` to support the classic terminal size.
- Fixed HUD drawing to position Unicode glyphs by terminal cell instead of UTF-8 byte offset.
- Added tests around CLI behavior and arena validation/movement.

### Architecture Diagram Cleanup

- Simplified `docs/gladiator-lan-shooter-architecture.excalidraw` into a compact diagram with only the important pieces: host, LAN UDP, joiner, shared game core, terminal frontend, build order, and core decisions.

### Terminal Render Scope Cleanup

- Reduced the terminal render to the current two-player scope: one controllable P1 and one visible P2 placeholder.
- Removed P3/P4 HUD labels, decorative extra spawn markers, the flickering `kills` counter, and phase/debug wording from the in-game terminal UI.

### Phase 1 Local Combat Loop

- Started Phase 1 local gameplay without adding any LAN/networking yet.
- Reworked `internal/game` from loose player/opponent fields into a two-player local match state with `Player`, `Bullet`, health, score, fire cooldowns, and respawn timers.
- Added Space/Enter firing in the terminal UI while keeping WASD/arrows for movement.
- Added bullet rendering, player hit detection, death, score increment, and timed respawn.
- Updated the HUD to show P1/P2 health, P1 cooldown, scores, and FPS. It still avoids P3/P4 labels, `kills`, and phase/debug text.
- Updated user-facing CLI/version text to remove outdated phase labels from runtime output.
- Added gameplay tests for player spawning, wall collision, player overlap blocking, firing cooldown, bullet damage, scoring, and respawn.
- Smoke-tested `go run ./cmd/gladiator play-local` in a pseudo-terminal after the combat HUD changes.

### Modularity Constraint

- Captured modularity as a standing project rule: game simulation, rendering, input, networking, assets, and CLI should stay separated behind small package boundaries.
- Near-term cleanup target: split the growing `internal/game` and `internal/termui` files before adding more gameplay or LAN behavior.

### Phase 1 Closure

- Closed the local gameplay phase with a playable two-player terminal loop.
- Split `internal/game` into focused files: arena, state, player, bullet, bot, and shared types.
- Split `internal/termui` into app loop, input mapping, rendering, and styles.
- Added a simple deterministic P2 bot: it chases P1 when blocked from line of sight, faces P1 when visible, and fires using the same game rules as P1.
- Made P1 able to die in normal local play through P2 bot fire.
- Improved respawn clarity in the HUD with `DOWN` plus a countdown instead of silently hiding dead players.
- Tuned local play slightly: bullet movement is faster, fire cooldown is readable, and HUD text stays scoped to the two-player game.
- Added tests for bot firing, bot killing P1, bot chasing, and key-to-action input mapping.

### Player Visual Consistency

- Updated P1 terminal styling to use the same kind of colored cell background as P2, so both players read as the same shape/size with different team colors.

### Phase 2 Deterministic Core Start

- Started the deterministic game-core phase by moving the primary simulation path from direct UI mutation to command-driven ticks.
- Added `InputCommand`, `InputFrame`, and button flags so movement/fire can be represented as tick-addressed data for future LAN transport.
- Added `State.Step(frame)` for in-place fixed-tick simulation and `State.Next(frame)` for a pure-ish clone-and-step path useful for future prediction/replay tests.
- Added `Snapshot`, `PlayerSnapshot`, and `BulletSnapshot` structs so render/network layers can read game state without mutating it.
- Updated the terminal app loop to collect player input into a pending command and submit that command once per simulation tick.
- Updated the P2 bot to produce a deterministic command instead of being only a direct state mutator.
- Added golden snapshot, repeated input-log determinism, snapshot aliasing, and `Next(frame)` contract tests.
- Bumped the development version to `0.3.0-dev`.

### Phase 2 Closure

- Closed the deterministic game-core phase with command frames as the main tested simulation path.
- Made direct movement/fire helpers internal to `internal/game`; UI and future network code should drive simulation through `InputFrame` and `State.Step`.
- Added command/frame validation for network readiness: tick checks, player slot checks, movement axis limits, diagonal movement rejection, aim validation, and unknown button rejection.
- Added match metadata and timer/score-limit state to snapshots.
- Added stable snapshot equality and FNV-1a hashing helpers for future replay/netcode comparisons.
- Added a replay fixture with a golden snapshot hash.
- Added tests for command validation, frame normalization, match ending, replay hash stability, snapshot equality/hash behavior, and command-driven movement/combat.
- Removed the old exported bot mutator so bot behavior also flows through deterministic commands.

### File Naming Cleanup

- Renamed the deterministic core test file from a phase-numbered filename to `internal/game/determinism_test.go`.
- Standing rule: code filenames should describe behavior or ownership, not roadmap phase numbers.

### Local Duel Rule Fix

- Fixed local play appearing to stop when P2 reached `S5` while P1 was down.
- Root cause: the local match was using the core score limit of 5, so the simulation correctly marked the match over but the terminal UI did not make that obvious.
- Changed the default local duel to open-ended rules: no score limit and no time limit for the playable prototype.
- Kept score-limit and time-limit behavior covered in explicit core tests for future LAN match modes.
- Added a regression test to ensure P1 still respawns after P2 reaches five scores in local play.

### LAN Protocol Foundation

- Started the LAN work with a small protocol slice before wiring sockets.
- Added `internal/protocol` for packet types and binary encode/decode.
- Current packet types: hello, welcome, input, snapshot, ping, and disconnect.
- Packet headers now carry protocol version, packet type, session id, sequence, and tick.
- Payloads reuse the deterministic game core types: `game.InputCommand` for inputs and `game.Snapshot` for state.
- Added round-trip tests for input, snapshot, welcome, hello, ping, and disconnect packets.
- Added malformed packet tests for short packets, bad magic, bad version, unknown type, trailing bytes, nil payloads, mismatched payloads, tick mismatches, and oversized strings.

### UDP Loopback Netplay Slice

- Added `internal/netplay` as the first socket-level LAN slice.
- Implemented a minimal UDP host that owns the authoritative `game.State`.
- Implemented a minimal UDP client that can send `hello`, receive `welcome`, send remote input, and receive a host snapshot.
- Kept this slice separate from the terminal UI and CLI wiring; `host` and `join` commands are still placeholders.
- The host currently assigns the joiner to `PlayerTwo`; the local host side remains `PlayerOne`.
- Added localhost loopback tests for join handshake, remote P2 movement through UDP, host/client snapshot agreement, stale input handling, and rejecting input before join.

### CLI LAN Debug Wiring

- Wired `gladiator host [addr:port]` to the UDP host in `internal/netplay`.
- Wired `gladiator join <ip|host[:port]>` to a plain debug flow: join the host, send one P2 move-left input, and print the authoritative snapshot returned by the host.
- Added address normalization so `gladiator join 192.168.x.x` uses the default LAN port while explicit ports still work.
- Added `RunContext` so tests and future UI flows can cancel host/join work cleanly.
- Kept terminal rendering out of this slice; this is only a command-line networking proof.
- Added a CLI test that starts a localhost host and runs the public `join` command through the full UDP path.

### Continuous Netplay Sessions

- Added continuous host/client session APIs on top of the existing UDP protocol.
- The host session now runs the authoritative game state on a fixed simulation ticker instead of only stepping when a debug input packet arrives.
- Added channel-based input and snapshot streams so the terminal UI can later feed local commands in and render incoming snapshots.
- Host session inputs control P1 locally; client session inputs are sent over UDP as P2 commands.
- Host streams authoritative snapshots back to the client at a snapshot rate.
- Added localhost tests for continuous snapshot streaming, P1 host-local movement, P2 remote movement over UDP, and host/client snapshot agreement.

### Terminal Host Mode Start

- Wired `gladiator host [addr:port]` into the terminal renderer instead of the earlier plain debug server loop.
- Host mode now starts a UDP host session, sends P1 terminal input into the authoritative simulation, and renders host snapshots in the existing arena UI.
- Added a snapshot-to-render-state adapter in `internal/termui` so network snapshots can reuse the current renderer without letting terminal code mutate the game core.
- Preserved `play-local` as its own local bot mode.
- `gladiator join <ip|host[:port]>` is still a plain debug join/one-input flow; playable joiner rendering is the next visible slice.
- Added a focused test for the snapshot adapter preserving arena/spawn data while applying network state.

### Terminal Join Mode Start

- Wired `gladiator join <ip|host[:port]>` into the terminal renderer.
- Join mode now connects to the host, receives its welcome snapshot, opens the same arena UI, controls P2, and renders authoritative snapshots from the host.
- Split client session startup so join can use a short connection timeout while the active terminal session keeps running on the long-lived app context.
- Kept `play-local` and `host` paths intact.
- Removed the older CLI debug join flow that sent one input and printed a text snapshot.
- Added a focused terminal input test to ensure the shared app queues movement for the selected player, including P2 join mode.

### LAN Disconnect And Status

- Added a graceful client disconnect packet on terminal join quit.
- Host sessions now publish peer connected/disconnected status events without coupling the terminal UI to socket internals.
- Host HUD shows `P2 WAIT` until a joiner connects, switches to `P2 LIVE` while P2 is present, and returns to `P2 WAIT` after the joiner quits.
- Reset the host-side remote input sequence on disconnect so a later reconnect is not blocked by stale packet ordering.
- Added a loopback regression test for connect status, disconnect status, and host remote cleanup.
- Added `docs/lan-test-checklist.md` for manual two-machine LAN verification.

### LAN Closure Pass

- Added handshake readiness fields so hello/welcome now carry explicit ready state.
- Expanded welcome metadata with map id and arena hash while keeping protocol version in the packet header.
- Added client heartbeat pings during active netplay sessions.
- Added host-side peer timeout detection for ungraceful joiner exits.
- Host HUD now distinguishes clean waiting state (`P2 WAIT`) from a timed-out peer (`P2 LOST`).
- Added regression coverage for silent peer timeout cleanup.
- Updated the LAN checklist with an ungraceful timeout test.

### Project Journal Scope

- Going forward, `docs/dev-log.md` is the single maintained progress journal for implementation notes, decisions, and milestone context.

### GitHub Prep

- Added a lightweight README and `.gitignore` so the project is ready for its first GitHub push.

### Netcode Hardening Start

- Extended the UDP protocol header with `Ack` and `AckBits` fields so each packet can report the newest peer packet seen plus a 32-packet receive window.
- Added a small netplay receive window that tracks accepted, duplicate, stale, reordered, and estimated-lost packets without pulling in a heavy networking library.
- Wired host and client sessions to fill ack fields on outgoing packets and ignore duplicate/stale gameplay inputs or snapshots.
- Added packet stats accessors on host/client for the future debug overlay.
- Added tests for ack-bit behavior, duplicate/stale drops, sequence wrap handling, and loopback packet stats.

### Netplay Link Simulator

- Added a deterministic local link simulator for continuous netplay sessions.
- Session send paths can now simulate outbound packet loss, base delay, and jitter without changing the game core or protocol payloads.
- Exposed per-session link stats for queued, sent, delayed, and dropped packets so a debug overlay can show local send-side behavior later.
- Added `GLADIATOR_NET_DROP_EVERY`, `GLADIATOR_NET_DELAY_MS`, and `GLADIATOR_NET_JITTER_MS` so host/join runs can opt into simulated bad network conditions from the terminal.
- Added tests for deterministic jitter timing, option normalization, and loopback loss/jitter surfacing through host receive stats.

### Terminal Network Debug Overlay

- Exposed receive-side packet stats and send-side simulated-link stats through host/client sessions.
- Added a terminal HUD debug line that can be toggled with `n` during host or join play.
- The debug line shows received packets, dropped/duplicate/stale counts, estimated loss, sent/queued packets, simulated drops, and delayed packets.
- Kept the overlay out of the default view so normal local and LAN play stays visually clean.
- Added focused tests for the debug snapshot, formatting, and key mapping.
