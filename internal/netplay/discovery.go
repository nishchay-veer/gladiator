package netplay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
)

const (
	discoveryRequestMagic  = "GLADIATOR_DISCOVER_V1"
	discoveryResponseMagic = "GLADIATOR_HOST_V1"
)

type DiscoveryOptions struct {
	LocalAddr  string
	TargetAddr string
	Port       int
}

type DiscoveredHost struct {
	Addr          string
	Port          int
	SessionID     uint64
	MapID         string
	MapHash       uint64
	Tick          uint64
	PeerConnected bool
}

type discoveryResponse struct {
	Magic         string `json:"magic"`
	Port          int    `json:"port"`
	SessionID     uint64 `json:"session_id"`
	MapID         string `json:"map_id"`
	MapHash       uint64 `json:"map_hash"`
	Tick          uint64 `json:"tick"`
	PeerConnected bool   `json:"peer_connected"`
}

func Discover(ctx context.Context, opts DiscoveryOptions) ([]DiscoveredHost, error) {
	localAddr := opts.LocalAddr
	if localAddr == "" {
		localAddr = ":0"
	}
	port := opts.Port
	if port == 0 {
		port = 42424
	}
	targetAddr := opts.TargetAddr
	if targetAddr == "" {
		targetAddr = net.JoinHostPort("255.255.255.255", strconv.Itoa(port))
	}

	udpLocalAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	udpTargetAddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpLocalAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.WriteToUDP([]byte(discoveryRequestMagic), udpTargetAddr); err != nil {
		return nil, err
	}

	buffer := make([]byte, 1024)
	hosts := make([]DiscoveredHost, 0, 4)
	seen := make(map[string]struct{})
	for {
		data, addr, err := readDatagram(ctx, conn, buffer)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return hosts, nil
			}
			return hosts, err
		}

		host, ok := parseDiscoveryResponse(data, addr)
		if !ok {
			continue
		}
		if _, exists := seen[host.Addr]; exists {
			continue
		}
		seen[host.Addr] = struct{}{}
		hosts = append(hosts, host)
	}
}

func (h *Host) handleDiscoveryDatagram(addr *net.UDPAddr, data []byte) bool {
	if !bytes.Equal(bytes.TrimSpace(data), []byte(discoveryRequestMagic)) {
		return false
	}
	_ = h.sendDiscoveryResponse(addr)
	return true
}

func (h *Host) sendDiscoveryResponse(addr *net.UDPAddr) error {
	h.mu.Lock()
	snapshot := h.state.Snapshot()
	mapHash := h.state.Arena.Hash()
	peerConnected := h.remote != nil
	sessionID := h.sessionID
	h.mu.Unlock()

	port := 0
	if local, ok := h.conn.LocalAddr().(*net.UDPAddr); ok {
		port = local.Port
	}

	data, err := json.Marshal(discoveryResponse{
		Magic:         discoveryResponseMagic,
		Port:          port,
		SessionID:     sessionID,
		MapID:         snapshot.Match.MapID,
		MapHash:       mapHash,
		Tick:          snapshot.Tick,
		PeerConnected: peerConnected,
	})
	if err != nil {
		return err
	}
	_, err = h.conn.WriteToUDP(data, addr)
	return err
}

func parseDiscoveryResponse(data []byte, addr *net.UDPAddr) (DiscoveredHost, bool) {
	var response discoveryResponse
	if err := json.Unmarshal(bytes.TrimSpace(data), &response); err != nil {
		return DiscoveredHost{}, false
	}
	if response.Magic != discoveryResponseMagic || addr == nil {
		return DiscoveredHost{}, false
	}

	port := response.Port
	if port == 0 {
		port = addr.Port
	}
	return DiscoveredHost{
		Addr:          net.JoinHostPort(addr.IP.String(), strconv.Itoa(port)),
		Port:          port,
		SessionID:     response.SessionID,
		MapID:         response.MapID,
		MapHash:       response.MapHash,
		Tick:          response.Tick,
		PeerConnected: response.PeerConnected,
	}, true
}
