package transport_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/netcatanytransport/ncany/pkg/transport"
)

func TestTCPTransport(t *testing.T) {
	// 1. Listen on a random dynamic port
	tcpTransport := transport.NewTCPTransport()
	listener, err := tcpTransport.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// 2. Accept connection in background
	errc := make(chan error, 1)
	go func() {
		serverConn, err := listener.Accept()
		if err != nil {
			errc <- err
			return
		}
		defer serverConn.Close()

		buf := make([]byte, 1024)
		n, err := serverConn.Read(buf)
		if err != nil {
			errc <- err
			return
		}

		if string(buf[:n]) != "ping" {
			errc <- net.UnknownNetworkError("unexpected payload")
			return
		}

		_, err = serverConn.Write([]byte("pong"))
		errc <- err
	}()

	// 3. Client connect
	clientConn, err := tcpTransport.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer clientConn.Close()

	// 4. Send "ping" and receive "pong"
	_, err = clientConn.Write([]byte("ping"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	buf := make([]byte, 1024)
	if nc, ok := clientConn.(net.Conn); ok {
		nc.SetReadDeadline(time.Now().Add(2 * time.Second))
	}
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(buf[:n]) != "pong" {
		t.Fatalf("Expected pong, got %s", string(buf[:n]))
	}

	// 5. Check background accept error
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("Server side error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("Timeout waiting for server completion")
	}
}
