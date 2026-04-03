package engine

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/netcatanytransport/ncany/pkg/transport"
)

// EngineConfig holds options for the engine execution.
type EngineConfig struct {
	Exec   string // Command to spawn and pipe to/from (e.g., "/bin/sh")
	ZeroIO bool   // If true, just connect and immediately disconnect (port scan)
}

// Engine manages the bidirectional data flow between the local I/O and the remote connection.
type Engine struct {
	Conn   transport.Conn
	In     io.Reader
	Out    io.Writer
	Config EngineConfig
}

// New creates a new engine with standard Unix I/O and the provided connection.
func New(conn transport.Conn, config EngineConfig) *Engine {
	return &Engine{
		Conn:   conn,
		In:     os.Stdin,
		Out:    os.Stdout,
		Config: config,
	}
}

// Run starts the engine. It blocks until complete.
func (e *Engine) Run(ctx context.Context) error {
	defer e.Conn.Close()

	if e.Config.ZeroIO {
		// Just connect and disconnect (this is mostly useful for clients assessing ports)
		return nil
	}

	if e.Config.Exec != "" {
		return e.runExec(ctx)
	}

	return e.runStandardIO(ctx)
}

func (e *Engine) runStandardIO(ctx context.Context) error {
	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(e.Conn, e.In)
		if err != nil && err != io.EOF {
			errc <- err
			return
		}
		errc <- nil
	}()

	go func() {
		_, err := io.Copy(e.Out, e.Conn)
		if err != nil && err != io.EOF {
			errc <- err
			return
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func (e *Engine) runExec(ctx context.Context) error {
	// Execute command securely
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", e.Config.Exec)

	cmd.Stdin = e.Conn
	cmd.Stdout = e.Conn
	cmd.Stderr = e.Conn

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}
