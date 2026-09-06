package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/shogo82148/go-webntp"
)

// the version of webntp. It is set by goreleaser.
var version string

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

	if showVersion {
		fmt.Println(getVersion())
		return
	}
	if help {
		flag.PrintDefaults()
		return
	}
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

func getVersion() string {
	var revision string
	var time string
	var modified bool

	if info, ok := debug.ReadBuildInfo(); ok {
		if version == "" {
			version = info.Main.Version
		}
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				revision = kv.Value
			case "vcs.time":
				time = kv.Value
			case "vcs.modified":
				if b, err := strconv.ParseBool(kv.Value); err == nil {
					modified = b
				}
			}
		}
	}

	var buf strings.Builder
	buf.WriteString("webntp version ")
	if version != "" {
		buf.WriteString(version)
	} else {
		buf.WriteString("unknown")
	}
	if revision != "" {
		buf.WriteString(" (")
		buf.WriteString(revision)
		buf.WriteString(" at ")
		buf.WriteString(time)
		if modified {
			buf.WriteString(", modified")
		}
		buf.WriteString(")")
	}
	buf.WriteString(", built with ")
	buf.WriteString(runtime.Version())
	buf.WriteString(" for ")
	buf.WriteString(runtime.GOOS)
	buf.WriteString("/")
	buf.WriteString(runtime.GOARCH)

	return buf.String()
}
