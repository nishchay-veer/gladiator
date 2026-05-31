package netplay

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/nishchay-veer/gladiator/internal/protocol"
)

const readPollInterval = 20 * time.Millisecond

func readDatagram(ctx context.Context, conn *net.UDPConn, buffer []byte) ([]byte, *net.UDPAddr, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		_ = conn.SetReadDeadline(time.Now().Add(readPollInterval))
		n, addr, err := conn.ReadFromUDP(buffer)
		if err == nil {
			return buffer[:n], addr, nil
		}
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
			continue
		}
		if errors.Is(err, net.ErrClosed) {
			return nil, nil, err
		}
		return nil, nil, err
	}
}

func sendPacket(conn *net.UDPConn, addr *net.UDPAddr, packet protocol.Packet) error {
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}

	_, err = conn.WriteToUDP(data, addr)
	return err
}

func sameUDPAddr(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Port == b.Port && a.IP.Equal(b.IP)
}

func cloneUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}

	clone := *addr
	if addr.IP != nil {
		clone.IP = append(net.IP(nil), addr.IP...)
	}
	return &clone
}
