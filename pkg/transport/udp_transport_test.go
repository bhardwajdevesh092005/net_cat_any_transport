package transport_test

import (
	"context"
	"testing"
	"time"

	"github.com/netcatanytransport/ncany/pkg/transport"
)

func TestUDPTransport(t *testing.T) {
	// 1. Listen on a random dynamic UDP port
	udpTransport := transport.NewUDPTransport()
	listener, err := udpTransport.Listen("127.0.0.1:0")
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
			t.Errorf("Expected ping, got %s", string(buf[:n]))
		}

		// Because it's UDP and we are reading from the generic conn without ReadFrom,
		// we don't naturally know the sender's address in this simple setup unless
		// the transport implements it. We will just pass since we received "ping".
		errc <- nil
	}()

	// 3. Client connect
	clientConn, err := udpTransport.Dial(addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer clientConn.Close()

	// 4. Send "ping"
	_, err = clientConn.Write([]byte("ping"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
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
