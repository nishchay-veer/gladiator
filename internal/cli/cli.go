package cli

import (
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

	"gladiator/internal/build"
	"gladiator/internal/config"
	"gladiator/internal/netplay"
	"gladiator/internal/termui"
)

const joinTimeout = 5 * time.Second

func Run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return RunContext(ctx, args, stdout, stderr)
}

func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
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
		if err := termui.PlayLocal(ctx, termui.PlayLocalOptions{Config: config.Default()}); err != nil {
			fmt.Fprintf(stderr, "play-local: %v\n", err)
			return 1
		}
		return 0
	case "host":
		return runHost(ctx, args[1:], stdout, stderr)
	case "join":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(stderr, "join requires an IP address: gladiator join <ip>")
			return 2
		}
		return runJoin(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runHost(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cfg := config.Default()
	bindAddr := ":" + strconv.Itoa(cfg.NetworkPort)
	if len(args) > 1 {
		fmt.Fprintln(stderr, "host accepts at most one bind address: gladiator host [addr:port]")
		return 2
	}
	if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
		bindAddr = strings.TrimSpace(args[0])
	}

	linkSimulation, err := linkSimulationFromEnv()
	if err != nil {
		fmt.Fprintf(stderr, "host: %v\n", err)
		return 2
	}

	err = termui.PlayHost(ctx, termui.PlayHostOptions{
		Config:         config.Default(),
		Addr:           bindAddr,
		LinkSimulation: linkSimulation,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "host: %v\n", err)
		return 1
	}
	return 0
}

func runJoin(ctx context.Context, args []string, stdout, stderr io.Writer) int {
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
	linkSimulation, err := linkSimulationFromEnv()
	if err != nil {
		fmt.Fprintf(stderr, "join: %v\n", err)
		return 2
	}

	err = termui.PlayJoin(ctx, termui.PlayJoinOptions{
		Config:         config.Default(),
		HostAddr:       hostAddr,
		PlayerName:     "P2",
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
  gladiator join <ip|host[:port]>
  gladiator version

Current build:
  play-local opens the local terminal game.
  host opens the LAN terminal host.
  join opens the LAN terminal joiner.

Network test env:
  GLADIATOR_NET_DROP_EVERY=N drops every Nth outbound session packet.
  GLADIATOR_NET_DELAY_MS=N adds base outbound session delay.
  GLADIATOR_NET_JITTER_MS=N adds deterministic outbound session jitter.
`)
}
