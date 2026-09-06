//go:build !windows

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/shogo82148/go-webntp"
	"github.com/shogo82148/server-starter/listener"
	"golang.org/x/sys/unix"
)

func serve(ctx context.Context) error {
	s := webntp.NewServer()
	srv := &http.Server{
		Addr:         serveHost,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      s,
	}
	idleWSConnsClosed := make(chan struct{})
	srv.RegisterOnShutdown(func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to shutdown the webntp server", slog.Any("error", err))
		}
		close(idleWSConnsClosed)
	})

	// handle OS interrupt signal for graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		// wait for signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, unix.SIGINT, unix.SIGTERM)
		<-sig
		signal.Stop(sig)

		slog.InfoContext(ctx, "received a signal, shutting down the server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to shutdown the server", slog.Any("error", err))
		}
		close(idleConnsClosed)
	}()

	// start the HTTP server and listen for incoming requests
	slog.InfoContext(ctx, "start to serve")
	if listener.IsUnderStartServer() {
		ll, err := listener.ListenAll()
		if err != nil {
			return err
		}
		ch := make(chan error, len(ll))
		for _, l := range ll {
			go func() {
				if err := srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
					slog.ErrorContext(ctx, "failed to serve on listener", slog.Any("error", err))
					ch <- err
				}
			}()
		}
		select {
		case err := <-ch:
			return err
		case <-idleConnsClosed:
		}
	} else {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		<-idleConnsClosed
	}
	<-idleWSConnsClosed
	return nil
}
