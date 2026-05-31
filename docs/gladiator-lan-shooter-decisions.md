# Gladiator LAN Shooter Decisions

This note pairs with `docs/gladiator-lan-shooter-architecture.excalidraw`.

## Recommended Stack

- Language: Go 1.26.x.
- Primary renderer: `github.com/gdamore/tcell/v2` for a portable terminal game with truecolor, Unicode, mouse support where available, and raw input.
- Optional native renderer later: Ebitengine, using the same `internal/game` simulation package behind a `Renderer` interface.
- Networking: Go standard library UDP via `net.UDPConn`; add UDP broadcast or mDNS discovery, with manual IP as the reliable fallback.
- Network model: host authoritative. One player hosts, the other joins. Clients send inputs; the host validates and sends snapshots.
- Asset tooling: Aseprite for pixel sprites, Tiled or LDtk for maps, a small converter that turns art into terminal palettes/glyphs/collision data, and `go:embed` for shipping assets.
- Build and release: `go test`, race tests, protocol compatibility tests, packet-loss simulation, cross-compilation, and GoReleaser when packaging starts.

## Core Decisions

- Use fixed-step simulation at 60 Hz. Render at up to 60 Hz. Send authoritative snapshots at 20-30 Hz.
- Use fixed-point integer coordinates so movement, collision, and reconciliation stay predictable.
- Keep UDP packets small and versioned: magic, protocol version, sequence, ack bitfield, tick, packet type, payload, checksum if needed.
- Use unreliable UDP for input and snapshots. Add a tiny reliable lane only for lobby, ready, map, and match-start events.
- Keep terminal graphics tile-first: 2-cell-wide square tiles, half-blocks, braille dots, truecolor palettes, dirty-cell rendering, and a hard viewport minimum.
- Design first for two players. Add spectators, bots, and more players only after the host-authoritative loop feels good.
- Keep the codebase modular: game simulation should not know about terminal rendering, terminal UI should not own game rules, networking should only exchange commands/snapshots, and packages should stay small enough to test directly.

## Suggested MVP Order

1. Local arena render, player movement, collision, and HUD.
2. Fixed-step game simulation with bullets, walls, deaths, respawns, and scoring.
3. Host/join UDP handshake on LAN with manual IP.
4. Input commands, authoritative snapshots, interpolation, and local prediction.
5. Discovery, lobby, ready state, timeout/reconnect UI, and error messages.
6. Asset converter and visual polish matching the screenshot style.
7. Optional audio, replay/debug tooling, bots, and Ebitengine native frontend.

## Current Source Checks

- Go downloads currently show Go 1.26.3 as the stable release line: https://go.dev/dl/
- Tcell documents terminal portability, truecolor support, and input/mouse support: https://github.com/gdamore/tcell and https://pkg.go.dev/github.com/gdamore/tcell/v2
- Ebitengine is the practical Go-native option for a later non-terminal 2D frontend: https://ebitengine.org/
- Excalidraw scene exports are JSON with `type`, `version`, `source`, `elements`, `appState`, and `files`: https://excalidraw-excalidraw.mintlify.app/api/types/data
