package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/shogo82148/go-webntp"
)

var showVersion bool
var help bool
var serveHost string

func init() {
	// Global options
	flag.BoolVar(&help, "help", false, "show help")
	flag.BoolVar(&showVersion, "version", false, "show the version")

	// Server options
	flag.StringVar(&serveHost, "serve", "", "server host name")
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	ctx := context.Background()
	flag.Parse()

	if serveHost == "" && flag.NArg() == 0 {
		flag.PrintDefaults()
		os.Exit(2)
		return
	}

	if serveHost != "" {
		if err := serve(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to serve", slog.Any("error", err))
			os.Exit(1)
		}
	} else {
		if err := client(ctx, flag.Args()); err != nil {
			slog.ErrorContext(ctx, "failed to get time", slog.Any("error", err))
			os.Exit(1)
		}
	}
}

func serve(ctx context.Context) error {
	s := webntp.NewServer()
	return http.ListenAndServe(serveHost, s)
}

func client(ctx context.Context, hosts []string) error {
	best := webntp.Result{
		Delay: math.MaxInt64,
	}
	bestHost := ""

	c := webntp.NewClient()
	for _, host := range hosts {
		result, err := c.Get(ctx, host)
		if err != nil {
			slog.ErrorContext(
				ctx, "failed to get time from host",
				slog.String("server", host), slog.Any("error", err),
			)
			continue
		}
		slog.InfoContext(
			ctx, "got time from host",
			slog.String("server", host),
			slog.Float64("offset", result.Offset.Seconds()),
			slog.Float64("delay", result.Delay.Seconds()),
		)
		if result.Delay < best.Delay {
			best = result
			bestHost = host
		}
	}
	if bestHost == "" {
		return errors.New("failed to get time from any host")
	}

	local := time.Now()
	remote := local.Add(best.Offset)
	slog.InfoContext(
		ctx, "best time",
		slog.String("server", bestHost),
		slog.Time("local", local),
		slog.Time("remote", remote),
		slog.Float64("offset", best.Offset.Seconds()),
		slog.Float64("delay", best.Delay.Seconds()),
	)
	return nil
}
