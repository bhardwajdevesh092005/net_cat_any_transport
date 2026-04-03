package transport

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

// WSTransport implements Transport for WebSockets.
type WSTransport struct{}

func NewWSTransport() *WSTransport {
	return &WSTransport{}
}

// Dial connects to a ws:// or wss:// endpoint via websocket upgrade.
func (t *WSTransport) Dial(address string) (Conn, error) {
	// Typically address for ws needs a full URL, e.g. ws://127.0.0.1:8080/
	urlStr := fmt.Sprintf("ws://%s/", address)
	c, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		return nil, err
	}
	return &wsConn{c: c}, nil
}

// Listen creates an HTTP server that upgrades incoming connections to websockets.
func (t *WSTransport) Listen(address string) (Listener, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	ln := &wsListener{
		ln:     l,
		conns:  make(chan Conn),
		closed: make(chan struct{}),
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		
		select {
		case ln.conns <- &wsConn{c: c}:
		case <-ln.closed:
			c.Close()
		}
	})

	srv := &http.Server{Handler: mux}
	ln.srv = srv
	
	go srv.Serve(l)

	return ln, nil
}

// wsConn wraps a *websocket.Conn to implement transport.Conn
type wsConn struct {
	c          *websocket.Conn
	readReader io.Reader
}

func (w *wsConn) Read(p []byte) (int, error) {
	if w.readReader == nil {
		_, r, err := w.c.NextReader()
		if err != nil {
			return 0, err
		}
		w.readReader = r
	}

	n, err := w.readReader.Read(p)
	if err == io.EOF {
		w.readReader = nil
		// Treat end of message as just needing next message, don't return EOF
		// but return n so caller can process it. Wait, io.Reader paradigm says
		// returning (n > 0, EOF) is valid. Or we can just return (n, nil)
		// and let the next Read call get EOF or block on NextReader.
		if n > 0 {
			return n, nil
		}
		// If n == 0, go to next reader recursively
		return w.Read(p)
	}
	return n, err
}

func (w *wsConn) Write(p []byte) (int, error) {
	// Write as a single binary message
	err := w.c.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsConn) Close() error {
	return w.c.Close()
}

func (w *wsConn) LocalAddr() net.Addr {
	return w.c.LocalAddr()
}

func (w *wsConn) RemoteAddr() net.Addr {
	return w.c.RemoteAddr()
}

// wsListener wraps the http server to act as a transport.Listener
type wsListener struct {
	ln     net.Listener
	srv    *http.Server
	conns  chan Conn
	closed chan struct{}
}

func (l *wsListener) Accept() (Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *wsListener) Close() error {
	close(l.closed)
	return l.srv.Close()
}

func (l *wsListener) Addr() net.Addr {
	return l.ln.Addr()
}
