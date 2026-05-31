# Release And Install Notes

The release path is GitHub-only for now. No Homebrew tap, Scoop bucket, WinGet package, or distro package is required before the first public build.

The target user experience after installation is:

```sh
gladiator host
gladiator join <host-ip>
```

## User Install

Users should download the right archive from GitHub Releases:

```text
https://github.com/nishchay-veer/gladiator/releases
```

Release artifacts:

- `gladiator_*_darwin_amd64.tar.gz` for Intel macOS
- `gladiator_*_darwin_arm64.tar.gz` for Apple Silicon macOS
- `gladiator_*_linux_amd64.tar.gz` for Linux amd64
- `gladiator_*_linux_arm64.tar.gz` for Linux arm64
- `gladiator_*_windows_amd64.zip` for Windows amd64
- `checksums.txt` for verifying downloads

After unpacking, the user puts the binary somewhere on their `PATH` and runs:

```sh
gladiator version
```

Users with Go installed can also install from source:

```sh
go install github.com/nishchay-veer/gladiator/cmd/gladiator@latest
```

## Release Path

1. Build and test locally:

```sh
make test
make build VERSION=1.0.0
bin/gladiator version
```

2. Optional GoReleaser config check if GoReleaser is installed locally:

```sh
goreleaser check
goreleaser release --snapshot --clean
```

3. Create a GitHub release by pushing a tag:

```sh
git tag v1.0.0
git push origin v1.0.0
```

4. GitHub Actions runs tests, then GoReleaser publishes release archives and checksums to the GitHub repo.

## Release Checklist

- `go test ./...` passes.
- `make build VERSION=1.0.0` prints `gladiator 1.0.0` from `gladiator version`.
- `make snapshot VERSION=1.0.0` cross-builds the supported OS/architecture matrix.
- The GitHub tag is `v1.0.0` for the first release.
- The GitHub release includes macOS, Linux, Windows, and checksum artifacts.
- macOS firewall hosting notes are visible in the README or release notes.
- A public license is chosen before calling the project generally installable.
