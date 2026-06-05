package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nishchay-veer/gladiator/internal/build"
	"github.com/nishchay-veer/gladiator/internal/config"
	"github.com/nishchay-veer/gladiator/internal/netplay"
	"github.com/nishchay-veer/gladiator/internal/termui"
)

const (
	joinTimeout     = 5 * time.Second
	discoverTimeout = 750 * time.Millisecond
)

func Run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return RunContext(ctx, args, stdout, stderr)
}

func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return RunContextWithInput(ctx, args, os.Stdin, stdout, stderr)
}

func RunContextWithInput(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "gladiator %s\n", build.Version)
		return 0
	case "play-local":
		cfg := config.Default()
		if err := termui.PlayLocal(ctx, termui.PlayLocalOptions{Config: cfg}); err != nil {
			fmt.Fprintf(stderr, "play-local: %v\n", err)
			return 1
		}
		return 0
	case "host":
		return runHost(ctx, args[1:], stdin, stdout, stderr)
	case "discover":
		return runDiscover(ctx, args[1:], stdout, stderr)
	case "join":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(stderr, "join requires an IP address: gladiator join <ip>")
			return 2
		}
		return runJoin(ctx, args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runHost(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg := config.Default()
	bindAddr := ":" + strconv.Itoa(cfg.NetworkPort)
	if len(args) > 1 {
		fmt.Fprintln(stderr, "host accepts at most one bind address: gladiator host [addr:port]")
		return 2
	}
	if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
		bindAddr = strings.TrimSpace(args[0])
	}

	playerName, err := promptPlayerName(stdin, stdout, "P1")
	if err != nil {
		fmt.Fprintf(stderr, "host: %v\n", err)
		return 1
	}
	linkSimulation, err := linkSimulationFromEnv()
	if err != nil {
		fmt.Fprintf(stderr, "host: %v\n", err)
		return 2
	}

	err = termui.PlayHost(ctx, termui.PlayHostOptions{
		Config:         cfg,
		Addr:           bindAddr,
		PlayerName:     playerName,
		LinkSimulation: linkSimulation,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "host: %v\n", err)
		return 1
	}
	return 0
}

func runDiscover(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "discover accepts at most one target address: gladiator discover [addr:port]")
		return 2
	}

	cfg := config.Default()

	targetAddr := ""
	if len(args) == 1 {
		targetAddr = strings.TrimSpace(args[0])
		if targetAddr == "" {
			fmt.Fprintln(stderr, "discover target address cannot be empty")
			return 2
		}
	}

	discoverCtx, cancel := context.WithTimeout(ctx, discoverTimeout)
	hosts, err := netplay.Discover(discoverCtx, netplay.DiscoveryOptions{
		TargetAddr: targetAddr,
		Port:       cfg.NetworkPort,
	})
	cancel()
	if err != nil {
		fmt.Fprintf(stderr, "discover: %v\n", err)
		return 1
	}

	if len(hosts) == 0 {
		fmt.Fprintln(stdout, "no hosts found")
		return 0
	}

	for _, host := range hosts {
		peerStatus := "waiting"
		if host.PeerConnected {
			peerStatus = "busy"
		}
		fmt.Fprintf(stdout, "%s map=%s tick=%d peer=%s\n", host.Addr, host.MapID, host.Tick, peerStatus)
	}
	return 0
}

func runJoin(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "join accepts one host address: gladiator join <ip>")
		return 2
	}

	cfg := config.Default()
	hostAddr, err := joinTargetAddress(args[0], cfg.NetworkPort)
	if err != nil {
		fmt.Fprintf(stderr, "join: %v\n", err)
		return 2
	}
	playerName, err := promptPlayerName(stdin, stdout, "P2")
	if err != nil {
		fmt.Fprintf(stderr, "join: %v\n", err)
		return 1
	}
	linkSimulation, err := linkSimulationFromEnv()
	if err != nil {
		fmt.Fprintf(stderr, "join: %v\n", err)
		return 2
	}

	err = termui.PlayJoin(ctx, termui.PlayJoinOptions{
		Config:         cfg,
		HostAddr:       hostAddr,
		PlayerName:     playerName,
		BuildVersion:   build.Version,
		JoinTimeout:    joinTimeout,
		LinkSimulation: linkSimulation,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "join: %v\n", err)
		return 1
	}
	return 0
}

func promptPlayerName(stdin io.Reader, stdout io.Writer, fallback string) (string, error) {
	fmt.Fprintf(stdout, "Player name [%s]: ", fallback)
	text, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return normalizePlayerName(text, fallback), nil
}

func normalizePlayerName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	const maxRunes = 12
	runes := []rune(name)
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return string(runes)
}

func linkSimulationFromEnv() (netplay.LinkSimulation, error) {
	dropEvery, err := nonNegativeEnvInt("GLADIATOR_NET_DROP_EVERY")
	if err != nil {
		return netplay.LinkSimulation{}, err
	}
	delayMillis, err := nonNegativeEnvInt("GLADIATOR_NET_DELAY_MS")
	if err != nil {
		return netplay.LinkSimulation{}, err
	}
	jitterMillis, err := nonNegativeEnvInt("GLADIATOR_NET_JITTER_MS")
	if err != nil {
		return netplay.LinkSimulation{}, err
	}

	return netplay.LinkSimulation{
		DropEvery: dropEvery,
		BaseDelay: time.Duration(delayMillis) * time.Millisecond,
		Jitter:    time.Duration(jitterMillis) * time.Millisecond,
	}, nil
}

func nonNegativeEnvInt(name string) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return parsed, nil
}

func joinTargetAddress(target string, defaultPort int) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("empty host address")
	}

	if _, _, err := net.SplitHostPort(target); err == nil {
		return target, nil
	}
	if ip := net.ParseIP(target); ip != nil {
		return net.JoinHostPort(target, strconv.Itoa(defaultPort)), nil
	}
	if strings.Count(target, ":") == 1 {
		host, portText, found := strings.Cut(target, ":")
		if found && strings.TrimSpace(host) != "" && strings.TrimSpace(portText) != "" {
			return target, nil
		}
	}
	return net.JoinHostPort(target, strconv.Itoa(defaultPort)), nil
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Gladiator

Usage:
  gladiator play-local
  gladiator host [addr:port]
  gladiator discover [addr:port]
  gladiator join <ip|host[:port]>
  gladiator version

Current build:
  play-local opens the local terminal game.
  host opens the LAN terminal host.
  discover searches for LAN hosts, with optional explicit target address.
  join opens the LAN terminal joiner.
  host and join ask for your player name before opening the game.

Network test env:
  GLADIATOR_NET_DROP_EVERY=N drops every Nth outbound session packet.
  GLADIATOR_NET_DELAY_MS=N adds base outbound session delay.
  GLADIATOR_NET_JITTER_MS=N adds deterministic outbound session jitter.
`)
}
