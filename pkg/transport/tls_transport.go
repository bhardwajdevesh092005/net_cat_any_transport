package transport

import (
	"crypto/tls"
	"net"
	"time"
)

// TLSTransport wraps an underlying transport to provide TLS encryption.
type TLSTransport struct {
	underlying Transport
	config     *tls.Config
}

// NewTLSTransport initializes a TLSTransport overlay map. 
func NewTLSTransport(underlying Transport, config *tls.Config) *TLSTransport {
	return &TLSTransport{
		underlying: underlying,
		config:     config,
	}
}

// Dial dials the underlying transport and performs TLS handshake.
func (t *TLSTransport) Dial(address string) (Conn, error) {
	conn, err := t.underlying.Dial(address)
	if err != nil {
		return nil, err
	}

	// We assume conn is an active connection over which we can overlay TLS
	// But it requires net.Conn. We wrapped net.Conn in our abstract Conn interface.
	// We'll create a lightweight adapter if needed, but since our Conn is io.ReadWriteCloser,
	// crypto/tls Client requires a standard net.Conn.
	
	netConn, ok := conn.(net.Conn)
	if !ok {
		// A rudimentary fallback wrapper to map Conn to net.Conn
		netConn = &connAdapter{Conn: conn}
	}

	tlsConn := tls.Client(netConn, t.config)
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	return &tlsWrapper{Conn: tlsConn}, nil
}

// Listen listens using underlying transport and wraps the listener.
func (t *TLSTransport) Listen(address string) (Listener, error) {
	ln, err := t.underlying.Listen(address)
	if err != nil {
		return nil, err
	}

	return &tlsListener{ln: ln, config: t.config}, nil
}

type tlsListener struct {
	ln     Listener
	config *tls.Config
}

func (l *tlsListener) Accept() (Conn, error) {
	conn, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}

	netConn, ok := conn.(net.Conn)
	if !ok {
		netConn = &connAdapter{Conn: conn}
	}

	tlsConn := tls.Server(netConn, l.config)
	
	// Handshake synchronously (for simplification, though typically this would be async in standard net listener)
	if err := tlsConn.Handshake(); err != nil {
		tlsConn.Close()
		return nil, err
	}

	return &tlsWrapper{Conn: tlsConn}, nil
}

func (l *tlsListener) Close() error {
	return l.ln.Close()
}

func (l *tlsListener) Addr() net.Addr {
	return l.ln.Addr()
}

// tlsWrapper is a simple wrapper ensuring it acts as our custom Conn.
type tlsWrapper struct {
	net.Conn
}

type connAdapter struct {
	Conn
}

// The core Conn implementors need Deadline wrappers if they are to be wrapped safely by TLS.
func (a *connAdapter) Read(b []byte) (n int, err error) { return a.Conn.Read(b) }
func (a *connAdapter) Write(b []byte) (n int, err error) { return a.Conn.Write(b) }
func (a *connAdapter) Close() error { return a.Conn.Close() }
func (a *connAdapter) LocalAddr() net.Addr { return a.Conn.LocalAddr() }
func (a *connAdapter) RemoteAddr() net.Addr { return a.Conn.RemoteAddr() }
func (a *connAdapter) SetDeadline(t time.Time) error { return nil }
func (a *connAdapter) SetReadDeadline(t time.Time) error { return nil }
func (a *connAdapter) SetWriteDeadline(t time.Time) error { return nil }
