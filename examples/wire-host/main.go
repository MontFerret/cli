// wire-host exposes an application-configured runtime for trusted local development.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/buildinfo"
	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/uapi"
	"github.com/MontFerret/wire/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (err error) {
	version, err := buildinfo.FerretVersion()
	if err != nil {
		return err
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}

	engine, err := ferret.New(ferret.WithFSRoot(root), ferret.WithFunctionsRegistrar(func(ns runtime.Namespace) {
		ns.Namespace("DEMO").Function().A1().Add("IDENTITY", func(_ context.Context, value runtime.Value) (runtime.Value, error) {
			return value, nil
		})
	}))
	if err != nil {
		return err
	}

	defer func() { err = errors.Join(err, engine.Close()) }()
	hosted := uapi.Wrap(engine, api.Version(version))
	defer func() { err = errors.Join(err, hosted.Close()) }()
	wire, err := server.NewServer(hosted)
	if err != nil {
		return err
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = errors.Join(err, wire.Shutdown(ctx))
	}()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("tcp://%s\n", listener.Addr())
	served := make(chan error, 1)
	go func() { served <- wire.Serve(context.Background(), listener) }()

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = wire.Shutdown(shutdown)

	select {
	case serveErr := <-served:
		return errors.Join(err, serveErr)
	case <-shutdown.Done():
		return errors.Join(err, shutdown.Err())
	}
}
