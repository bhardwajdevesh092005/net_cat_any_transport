package transport

import (
	"io"
	"net"
)

// Conn represents an active bidirectional stream or datagram connection.
type Conn interface {
	io.ReadWriteCloser
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// Listener represents a server listener.
type Listener interface {
	Accept() (Conn, error)
	Close() error
	Addr() net.Addr
}

// Transport abstracts the connection establishment for a specific protocol.
type Transport interface {
	Dial(address string) (Conn, error)
	Listen(address string) (Listener, error)
}
