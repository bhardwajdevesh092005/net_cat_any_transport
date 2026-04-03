package transport

import (
	"net"
)

// TCPTransport implements the Transport interface for TCP.
type TCPTransport struct{}

// NewTCPTransport creates a new TCP Transport.
func NewTCPTransport() *TCPTransport {
	return &TCPTransport{}
}

// Dial connects to a remote TCP address.
func (t *TCPTransport) Dial(address string) (Conn, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Listen creates a listening TCP socket.
func (t *TCPTransport) Listen(address string) (Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &tcpListener{ln: ln}, nil
}

// tcpListener wraps the net.Listener to return our Conn interface.
type tcpListener struct {
	ln net.Listener
}

func (l *tcpListener) Accept() (Conn, error) {
	conn, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (l *tcpListener) Close() error {
	return l.ln.Close()
}

func (l *tcpListener) Addr() net.Addr {
	return l.ln.Addr()
}
