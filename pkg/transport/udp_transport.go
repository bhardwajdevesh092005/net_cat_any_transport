package transport

import (
	"net"
	"sync"
)

// UDPTransport implements the Transport interface for UDP.
type UDPTransport struct{}

// NewUDPTransport creates a new UDP Transport.
func NewUDPTransport() *UDPTransport {
	return &UDPTransport{}
}

// Dial creates a UDP connection. Note that in Go, net.Dial for UDP creates a "connected" UDP socket.
func (t *UDPTransport) Dial(address string) (Conn, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Listen creates a listening UDP socket and returns a simple listener.
// For UDP, Accept() will just return the UDPConn itself that reads/writes data.
func (t *UDPTransport) Listen(address string) (Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return &udpListener{conn: conn}, nil
}

type udpListener struct {
	conn   *net.UDPConn
	closed bool
	mu     sync.Mutex
}

// Accept returns the underlying UDPConn wrapped as a single Conn.
// Subsequent calls to Accept will block until the listener is closed, as UDP is connectionless.
func (l *udpListener) Accept() (Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil, net.ErrClosed
	}

	// In a simple UDP implementation (like traditional netcat),
	// the listener just provides the UDP socket directly for read/write.
	l.closed = true
	return l.conn, nil
}

func (l *udpListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		return l.conn.Close()
	}
	return nil
}

func (l *udpListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
