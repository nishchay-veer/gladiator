# Gladiator

Gladiator is a Go terminal LAN shooting game for two players.

The current build supports:

- Local terminal play with `play-local`
- Host-authoritative UDP LAN play with `host`
- Joiner terminal play with `join <ip|host[:port]>`
- Deterministic game-core tests for commands, snapshots, replay, and netplay

## Requirements

- A terminal that supports tcell alternate-screen rendering
- Two terminals or two machines on the same LAN for netplay
- Go `1.24.2` or compatible local toolchain only if building from source

## Install

Download a release archive for your OS from:

```text
https://github.com/nishchay-veer/gladiator/releases
```

Unpack it, put `gladiator` somewhere on your `PATH`, then run:

```sh
gladiator version
```

On macOS, the first unsigned release may be blocked by Gatekeeper. If you trust the GitHub release you downloaded, remove the download quarantine flag:

```sh
xattr -d com.apple.quarantine ./gladiator
./gladiator version
```

From source with Go:

```sh
go install github.com/nishchay-veer/gladiator/cmd/gladiator@latest
gladiator version
```

From a local clone:

```sh
go run ./cmd/gladiator version
```

See `docs/release.md` for the release checklist.

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

Network test knobs:

```sh
GLADIATOR_NET_DROP_EVERY=5 GLADIATOR_NET_DELAY_MS=20 GLADIATOR_NET_JITTER_MS=10 go run ./cmd/gladiator join <host-ip>
```

These apply to outbound session packets for `host` and `join`.
Press `n` during `host` or `join` to toggle the compact network debug line.

## Test

```sh
go test ./...
```
