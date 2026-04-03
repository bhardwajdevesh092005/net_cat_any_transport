package transport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// PskAuthConn wraps a connection and performs a simple plaintext PSK assertion handshake before delegating standard bytes.
type PskAuthConn struct {
	Conn
	password []byte
	isServer bool
}

// NewPskAuthConn establishes a shared-secret wrapper. Expects immediate handshake execution.
func NewPskAuthConn(conn Conn, password string, isServer bool) *PskAuthConn {
	return &PskAuthConn{
		Conn:     conn,
		password: []byte(password),
		isServer: isServer,
	}
}

// Handshake verifies the password immediately.
func (p *PskAuthConn) Handshake() error {
	if !p.isServer {
		// Client: send password
		msg := fmt.Sprintf("AUTH:%s\n", p.password)
		_, err := p.Conn.Write([]byte(msg))
		return err
	}

	// Server: read password
	buf := make([]byte, len("AUTH:")+len(p.password)+1)
	_, err := io.ReadFull(p.Conn, buf)
	if err != nil {
		return fmt.Errorf("authentication read failed: %w", err)
	}

	expected := fmt.Sprintf("AUTH:%s\n", p.password)
	if !bytes.Equal(buf, []byte(expected)) {
		return errors.New("authentication failed: invalid PSK password")
	}

	return nil
}
