# Gladiator

Gladiator is a Go terminal LAN shooter prototype for two players.

The current build supports:

- Local terminal play with `play-local`
- Host-authoritative UDP LAN play with `host`
- Joiner terminal play with `join <ip|host[:port]>`
- Deterministic game-core tests for commands, snapshots, replay, and netplay

## Requirements

- Go `1.24.2` or compatible local toolchain
- A terminal that supports tcell alternate-screen rendering
- Two terminals or two machines on the same LAN for netplay

## Run

Local duel:

```sh
go run ./cmd/gladiator play-local
```

Host:

```sh
go run ./cmd/gladiator host
```

Join:

```sh
go run ./cmd/gladiator join <host-ip>
```

The default LAN port is `42424`. See `docs/lan-test-checklist.md` for the manual two-machine test flow.

## Test

```sh
go test ./...
```
