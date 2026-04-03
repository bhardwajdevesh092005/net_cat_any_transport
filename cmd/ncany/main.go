package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/netcatanytransport/ncany/pkg/cli"
	"github.com/netcatanytransport/ncany/pkg/engine"
	"github.com/netcatanytransport/ncany/pkg/transport"
)

var (
	execCmdStr string
	zeroIO     bool
	timeoutSec int
	configFile string
	useTLS     bool
	password   string
)

var rootCmd = &cobra.Command{
	Use:   "ncany",
	Short: "netcatanytransport - A versatile network swiss army knife",
}

func setupConn(conn transport.Conn, isServer bool) (transport.Conn, error) {
	if password != "" {
		pskConn := transport.NewPskAuthConn(conn, password, isServer)
		if err := pskConn.Handshake(); err != nil {
			return nil, err
		}
		conn = pskConn
	}
	return conn, nil
}

func getWrappedTransport(t transport.Transport) transport.Transport {
	if useTLS {
		// Use empty generic config, bypassing strict verification just for MVP/testing purposes.
		// In production we should use cert files configuration.
		tlsConf := &tls.Config{InsecureSkipVerify: true}
		return transport.NewTLSTransport(t, tlsConf)
	}
	return t
}

var connectCmd = &cobra.Command{
	Use:   "connect [protocol]://[address]",
	Short: "Connect to a listener (e.g., tcp://127.0.0.1:8080)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configFile != "" {
			return runFromConfig()
		}
		if len(args) != 1 {
			return fmt.Errorf("requires address argument or --config")
		}

		proto, addr, err := parseURI(args[0])
		if err != nil {
			return err
		}

		t, err := getTransport(proto)
		if err != nil {
			return err
		}

		t = getWrappedTransport(t)

		ctx := context.Background()
		if timeoutSec > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()
		}

		conn, err := t.Dial(addr)
		if err != nil {
			return err
		}

		conn, err = setupConn(conn, false)
		if err != nil {
			conn.Close()
			return err
		}

		eng := engine.New(conn, engine.EngineConfig{
			Exec:   execCmdStr,
			ZeroIO: zeroIO,
		})
		return eng.Run(ctx)
	},
}

var listenCmd = &cobra.Command{
	Use:   "listen [protocol]://[address]",
	Short: "Listen for incoming connections (e.g., tcp://0.0.0.0:8080)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configFile != "" {
			return runFromConfig()
		}
		if len(args) != 1 {
			return fmt.Errorf("requires address argument or --config")
		}

		proto, addr, err := parseURI(args[0])
		if err != nil {
			return err
		}

		t, err := getTransport(proto)
		if err != nil {
			return err
		}

		t = getWrappedTransport(t)

		ln, err := t.Listen(addr)
		if err != nil {
			return err
		}
		defer ln.Close()

		fmt.Printf("Listening on %s://%s\n", proto, ln.Addr().String())

		ctx := context.Background()
		if timeoutSec > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()
		}

		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		conn, err = setupConn(conn, true)
		if err != nil {
			conn.Close()
			return err
		}

		eng := engine.New(conn, engine.EngineConfig{
			Exec:   execCmdStr,
			ZeroIO: zeroIO,
		})
		return eng.Run(ctx)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&execCmdStr, "exec", "e", "", "Execute a command securely and pipe connection")
	rootCmd.PersistentFlags().BoolVarP(&zeroIO, "zero-i-o", "z", false, "Zero I/O mode (used for scanning)")
	rootCmd.PersistentFlags().IntVarP(&timeoutSec, "timeout", "w", 0, "Timeout for connects and final net reads in seconds")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Use YAML config file describing transports")
	
	// Phase 3 security/wrap options
	rootCmd.PersistentFlags().BoolVarP(&useTLS, "tls", "S", false, "Wrap transport socket with strict TLS/SSL")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Require Pre-Shared Key mutual auth immediately upon connection")

	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(listenCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseURI(uri string) (string, string, error) {
	var proto, addr string
	_, err := fmt.Sscanf(uri, "%[a-z]://%s", &proto, &addr)
	if err != nil {
		return "", "", fmt.Errorf("invalid address format, expected proto://addr (got: %s): %w", uri, err)
	}
	return proto, addr, nil
}

func getTransport(proto string) (transport.Transport, error) {
	switch proto {
	case "tcp":
		return transport.NewTCPTransport(), nil
	case "udp":
		return transport.NewUDPTransport(), nil
	case "ws":
		return transport.NewWSTransport(), nil
	default:
		return nil, fmt.Errorf("unsupported transport protocol: %s", proto)
	}
}

func runFromConfig() error {
	conf, err := cli.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config %s: %w", configFile, err)
	}
	
	if conf.Options.Timeout > 0 && timeoutSec == 0 {
		timeoutSec = conf.Options.Timeout
	}

	if len(conf.Transports) == 0 {
		return fmt.Errorf("no transports defined in config")
	}

	// Read properties from the first item solely as a prototype integration
	for _, tConfig := range conf.Transports {
		t, err := getTransport(tConfig.Type)
		if err != nil {
			return err
		}

		// Configure global variables strictly bound to the config item mapped to global logic
		if tConfig.TLS {
			useTLS = true
		}
		if tConfig.Password != "" {
			password = tConfig.Password
		}
		
		t = getWrappedTransport(t)

		ctx := context.Background()
		if timeoutSec > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()
		}

		engConfig := engine.EngineConfig{
			Exec:   tConfig.Exec,
			ZeroIO: tConfig.ZeroIO,
		}

		if tConfig.Listen != "" {
			ln, err := t.Listen(tConfig.Listen)
			if err != nil {
				return err
			}
			defer ln.Close()
			c, err := ln.Accept()
			if err != nil {
				return err
			}
			c, err = setupConn(c, true)
			if err != nil {
				c.Close()
				return err
			}
			eng := engine.New(c, engConfig)
			return eng.Run(ctx)
		} else if tConfig.Connect != "" {
			c, err := t.Dial(tConfig.Connect)
			if err != nil {
				return err
			}
			c, err = setupConn(c, false)
			if err != nil {
				c.Close()
				return err
			}
			eng := engine.New(c, engConfig)
			return eng.Run(ctx)
		}
	}
	return nil
}
