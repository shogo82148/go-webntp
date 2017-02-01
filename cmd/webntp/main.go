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
	flag.StringVar(&serveHost, "serve", "", "server host name")
	flag.Parse()

	if serveHost != "" {
		if err := serve(serveHost); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := client(flag.Args()); err != nil {
			log.Fatal(err)
		}
	}
}

func serve(host string) error {
	s := &webntp.Server{}
	return http.ListenAndServe(host, s)
}

func client(hosts []string) error {
	c := &webntp.Client{}
	for _, arg := range hosts {
		result, err := c.Get(arg)
		if err != nil {
			fmt.Printf("%s: Error %v\n", arg, err)
		} else {
			fmt.Printf(
				"uri %s, offset %.3f ms, delay %.3f ms",
				arg,
				result.Offset.Seconds()*1000,
				result.Delay.Seconds()*1000,
			)
		}
	}
	return nil
}
