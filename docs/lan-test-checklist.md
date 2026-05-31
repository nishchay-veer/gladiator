# LAN Test Checklist

Use this when checking the current two-player terminal LAN build on real machines.

## Before Starting

- Both machines are on the same Wi-Fi or wired LAN.
- The host machine firewall allows incoming UDP on port `42424`.
- Both machines are running the same checkout/build.
- Terminal size is at least `80x24`.

## Host Machine

- Find the host IP.
  - macOS Wi-Fi: `ipconfig getifaddr en0`
  - macOS wired: `ipconfig getifaddr en1`
- Start the host: `go run ./cmd/gladiator host`
- Expected: terminal arena opens, P1 is controllable, HUD shows `P2 WAIT`.

## Join Machine

- Start the joiner with the host IP: `go run ./cmd/gladiator join <host-ip>`
- Expected: terminal arena opens, joiner controls P2, host HUD changes to `P2 LIVE`.
- Move/fire on both machines and confirm both screens show the same health, score, and positions after a moment.

## Disconnect Check

- Quit the joiner with `q` or Esc.
- Expected: host keeps running and HUD returns to `P2 WAIT`.
- Join again from the second machine.
- Expected: P2 can reconnect and move again without restarting the host.

## Timeout Check

- Start host and join normally.
- Stop the joiner without using the in-game quit path, such as closing the terminal tab or killing the process from another shell.
- Expected: host keeps running and HUD changes to `P2 LOST` after roughly two seconds.
- Start the joiner again.
- Expected: host changes back to `P2 LIVE` and P2 can move/fire again.

## If It Fails

- Try an explicit port: `go run ./cmd/gladiator join <host-ip>:42424`
- Confirm both machines can reach each other on the LAN.
- Check the host terminal did not exit with an error.
- If the joiner times out, check firewall settings for UDP `42424`.
