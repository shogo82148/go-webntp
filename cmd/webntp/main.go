package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/shogo82148/go-webntp"
)

func main() {
	var serveHost string
	var leapSecondsPath, leapSecondsURL string
	var samples int
	flag.StringVar(&serveHost, "serve", "", "server host name")
	flag.StringVar(&leapSecondsPath, "leap-second-path", "leap-seconds.list", "path for leap-seconds.list cache")
	flag.StringVar(&leapSecondsURL, "leap-second-url", "https://www.ietf.org/timezones/data/leap-seconds.list", "url for leap-seconds.list")
	flag.IntVar(&samples, "p", 2, "Specify the number of samples")
	flag.Parse()

	if serveHost != "" {
		if err := serve(serveHost, leapSecondsPath, leapSecondsURL); err != nil {
			log.Fatal(err)
		}
	} else {
		if samples < 1 || samples > 8 {
			log.Fatalf("invalid samples: %d", samples)
		}
		if err := client(samples, flag.Args()); err != nil {
			log.Fatal(err)
		}
	}
}

func serve(host, leapSecondsPath, leapSecondsURL string) error {
	s := &webntp.Server{
		LeapSecondsPath: leapSecondsPath,
		LeapSecondsURL:  leapSecondsURL,
	}
	s.Start()
	return http.ListenAndServe(host, s)
}

func client(samples int, hosts []string) error {
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
