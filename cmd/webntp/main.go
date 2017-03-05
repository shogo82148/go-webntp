package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/shogo82148/go-webntp"
)

var serveHost string
var allowCrossOrigin bool
var leapSecondsPath, leapSecondsURL string
var samples int

func init() {
	// Server options
	flag.StringVar(&serveHost, "serve", "", "server host name")
	flag.BoolVar(&allowCrossOrigin, "allow-cross-origin", false, "allow cross origin request")
	flag.StringVar(&leapSecondsPath, "leap-second-path", "leap-seconds.list", "path for leap-seconds.list cache")
	flag.StringVar(&leapSecondsURL, "leap-second-url", "https://www.ietf.org/timezones/data/leap-seconds.list", "url for leap-seconds.list")

	// Client options
	flag.IntVar(&samples, "p", 2, "Specify the number of samples")
}

func main() {
	flag.Parse()

	if serveHost != "" {
		if err := serve(); err != nil {
			log.Fatal(err)
		}
	} else {
		if samples < 1 || samples > 8 {
			log.Fatalf("invalid samples: %d", samples)
		}
		if err := client(flag.Args()); err != nil {
			log.Fatal(err)
		}
	}
}

func serve() error {
	s := &webntp.Server{
		LeapSecondsPath: leapSecondsPath,
		LeapSecondsURL:  leapSecondsURL,
	}
	if allowCrossOrigin {
		s.Upgrader = &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    []string{webntp.Subprotocol},
			CheckOrigin:     func(*http.Request) bool { return true },
		}
	}
	s.Start()
	return http.ListenAndServe(serveHost, s)
}

func client(hosts []string) error {
	c := &webntp.Client{}
	for _, arg := range hosts {
		result, err := c.GetMulti(arg, samples)
		if err != nil {
			fmt.Printf("%s: Error %v\n", arg, err)
		} else {
			fmt.Printf(
				"uri %s, offset %.6f, delay %.6f\n",
				arg,
				result.Offset.Seconds(),
				result.Delay.Seconds(),
			)
		}
	}
	return nil
}
