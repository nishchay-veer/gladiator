package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nishchay-veer/gladiator/internal/netplay"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(nil) code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "gladiator play-local") {
		t.Fatalf("help output did not mention play-local:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunDiscoverFindsExplicitLoopbackHost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	host, err := netplay.ListenHost(netplay.HostOptions{
		Addr:      "127.0.0.1:0",
		SessionID: 777,
	})
	if err != nil {
		cancel()
		t.Fatalf("ListenHost() error = %v", err)
	}

	errc := make(chan error, 1)
	go func() {
		errc <- host.Serve(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		_ = host.Close()
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("host Serve() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("host did not stop")
		}
	})

	var stdout, stderr bytes.Buffer
	code := RunContext(context.Background(), []string{"discover", host.Addr().String()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("RunContext(discover) code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), host.Addr().String()) {
		t.Fatalf("stdout = %q, want host address %q", stdout.String(), host.Addr().String())
	}
	if !strings.Contains(stdout.String(), "map=local-arena-01") {
		t.Fatalf("stdout = %q, want map id", stdout.String())
	}
}

func TestPromptPlayerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback string
		want     string
	}{
		{name: "entered", input: "Nish\n", fallback: "P1", want: "Nish"},
		{name: "default", input: "\n", fallback: "P2", want: "P2"},
		{name: "trim and cap", input: "  VeryLongPlayerName  \n", fallback: "P1", want: "VeryLongPlay"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			got, err := promptPlayerName(strings.NewReader(tt.input), &stdout, tt.fallback)
			if err != nil {
				t.Fatalf("promptPlayerName() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("promptPlayerName() = %q, want %q", got, tt.want)
			}
			if !strings.Contains(stdout.String(), "Player name") {
				t.Fatalf("prompt output = %q, want prompt", stdout.String())
			}
		})
	}
}

func TestRunJoinRequiresAddress(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"join"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run(join) code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "join requires an IP address") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunHostRejectsTooManyArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := RunContext(context.Background(), []string{"host", "127.0.0.1:0", "extra"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("RunContext(host extra) code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "host accepts at most one bind address") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunJoinRejectsTooManyArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := RunContext(context.Background(), []string{"join", "127.0.0.1:0", "extra"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("RunContext(join extra) code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "join accepts one host address") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"nope"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run(nope) code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestJoinTargetAddress(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "ipv4 default port", target: "192.168.1.20", want: "192.168.1.20:42424"},
		{name: "hostname default port", target: "gladiator.local", want: "gladiator.local:42424"},
		{name: "provided port", target: "localhost:9999", want: "localhost:9999"},
		{name: "ipv6 default port", target: "::1", want: "[::1]:42424"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := joinTargetAddress(tt.target, 42424)
			if err != nil {
				t.Fatalf("joinTargetAddress() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("joinTargetAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLinkSimulationFromEnv(t *testing.T) {
	t.Setenv("GLADIATOR_NET_DROP_EVERY", "3")
	t.Setenv("GLADIATOR_NET_DELAY_MS", "12")
	t.Setenv("GLADIATOR_NET_JITTER_MS", "5")

	got, err := linkSimulationFromEnv()
	if err != nil {
		t.Fatalf("linkSimulationFromEnv() error = %v", err)
	}
	if got.DropEvery != 3 {
		t.Fatalf("drop every = %d, want 3", got.DropEvery)
	}
	if got.BaseDelay != 12*time.Millisecond {
		t.Fatalf("base delay = %s, want 12ms", got.BaseDelay)
	}
	if got.Jitter != 5*time.Millisecond {
		t.Fatalf("jitter = %s, want 5ms", got.Jitter)
	}
}

func TestLinkSimulationFromEnvRejectsInvalidValue(t *testing.T) {
	t.Setenv("GLADIATOR_NET_DROP_EVERY", "-1")

	if _, err := linkSimulationFromEnv(); err == nil {
		t.Fatal("linkSimulationFromEnv() error = nil, want invalid value error")
	}
}
