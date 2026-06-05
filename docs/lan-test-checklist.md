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
- Enter a player name at the prompt.
- Expected: terminal arena opens, P1 is controllable, HUD shows `P2 WAIT`.

## Join Machine

- Discover LAN hosts if broadcast works on the network: `gladiator discover`
- If discovery finds the host, use the printed address in the join command.
- Start the joiner with the host IP: `go run ./cmd/gladiator join <host-ip>`
- Enter a player name at the prompt.
- Expected: terminal arena opens, joiner controls P2, host HUD changes to `P2 LIVE`.
- Expected: both player names appear above their moving positions.
- Move/fire on both machines and confirm both screens show the same health, score, and positions after a moment.

## Disconnect Check

- Quit the joiner with `q` or Esc.
- Expected: host keeps running and HUD returns to `P2 WAIT`.
- Join again from the second machine.
- Expected: P2 can reconnect and move again without restarting the host.

## Rematch Check

- While host and joiner are connected, press `r` on the host.
- Expected: scores reset to zero on both machines and the duel continues.
- Press `r` on the joiner.
- Expected: nothing resets; rematch authority stays with the host.

## Timeout Check

- Start host and join normally.
- Stop the joiner without using the in-game quit path, such as closing the terminal tab or killing the process from another shell.
- Expected: host keeps running and HUD changes to `P2 LOST` after roughly two seconds.
- Start the joiner again.
- Expected: host changes back to `P2 LIVE` and P2 can move/fire again.

## If It Fails

- Try an explicit port: `go run ./cmd/gladiator join <host-ip>:42424`
- If discovery returns `no hosts found`, use manual IP join; some Wi-Fi networks block UDP broadcast.
- Confirm both machines can reach each other on the LAN.
- Check the host terminal did not exit with an error.
- If the joiner times out, check firewall settings for UDP `42424`.
