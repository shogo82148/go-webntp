package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shogo82148/go-webntp"
	"github.com/shogo82148/go-webntp/ntpdshm"
)

var help bool
var serveHost string
var allowCrossOrigin bool
var leapSecondsPath, leapSecondsURL string
var samples int
var shmUnits uint

func init() {
	flag.BoolVar(&help, "help", false, "show help")

	// Server options
	flag.StringVar(&serveHost, "serve", "", "server host name")
	flag.BoolVar(&allowCrossOrigin, "allow-cross-origin", false, "allow cross origin request")
	flag.StringVar(&leapSecondsPath, "leap-second-path", "leap-seconds.list", "path for leap-seconds.list cache")
	flag.StringVar(&leapSecondsURL, "leap-second-url", "https://www.ietf.org/timezones/data/leap-seconds.list", "url for leap-seconds.list")

	// Client options
	flag.IntVar(&samples, "p", 4, "Specify the number of samples")
	flag.UintVar(&shmUnits, "shm", 0, "ntpd shared-memory-segment")
}

func main() {
	flag.Parse()

	if serveHost == "" && flag.NArg() == 0 {
		help = true
	}
	if help {
		flag.PrintDefaults()
		return
	}

	if serveHost != "" {
		if err := serve(); err != nil {
			log.Fatal(err)
		}
	} else if shmUnits == 0 {
		if samples < 1 || samples > 8 {
			log.Fatalf("invalid samples: %d", samples)
		}
		if _, err := client(flag.Args()); err != nil {
			log.Fatal(err)
		}
	} else {
		if samples < 1 || samples > 8 {
			log.Fatalf("invalid samples: %d", samples)
		}

		// init random source.
		var s int64
		if err := binary.Read(crand.Reader, binary.LittleEndian, &s); err != nil {
			s = time.Now().UnixNano()
		}
		r := rand.New(rand.NewSource(s))

		for {
			var err error
			var result webntp.Result
			if result, err = client(flag.Args()); err != nil {
				log.Println(err)
			}
			if err = setClock(result); err != nil {
				log.Println(err)
			}
			d := r.Int63n(int64(2 * time.Second))
			time.Sleep(59*time.Second + time.Duration(d))
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

func client(hosts []string) (webntp.Result, error) {
	best := webntp.Result{
		Delay: 1<<63 - 1,
	}
	bestHost := ""

	c := &webntp.Client{}
	for _, arg := range hosts {
		result, err := c.GetMulti(arg, samples)
		if err != nil {
			fmt.Printf("%s: Error %v\n", arg, err)
		} else {
			fmt.Printf(
				"server %s, offset %.6f, delay %.6f\n",
				arg,
				result.Offset.Seconds(),
				result.Delay.Seconds(),
			)
			if result.Delay < best.Delay {
				best = result
				bestHost = arg
			}
		}
	}

	local := time.Now()
	remote := local.Add(best.Offset)

	fmt.Printf("%s, server %s, offset %.6f\n", remote, bestHost, best.Offset.Seconds())
	return best, nil
}

func setClock(result webntp.Result) error {
	var precision int32
	if delay := result.Delay; delay > 0 && delay < time.Second {
		for delay < time.Second {
			delay *= 2
			precision--
		}
	} else {
		for delay > time.Second {
			delay /= 2
			precision++
		}
	}

	local := time.Now()
	remote := local.Add(result.Offset)

	shm, err := ntpdshm.Get(shmUnits)
	if err != nil {
		return err
	}
	shm.Lock()
	defer shm.Unlock()
	shm.IncrCount()
	shm.SetClockTimeStamp(remote)
	shm.SetReceiveTimeStamp(local)
	shm.SetPrecision(precision)

	// set leap second indicator
	leap := result.NextLeap.Sub(remote)
	if leap <= 0 {
		shm.SetLeap(ntpdshm.LeapNoWarning)
		return nil
	}
	if leap > 24*time.Hour {
		shm.SetLeap(ntpdshm.LeapNoWarning)
		return nil
	}
	if result.Step > 0 {
		shm.SetLeap(ntpdshm.LeapAddSecond)
	} else if result.Step < 0 {
		shm.SetLeap(ntpdshm.LeapDelSecond)
	} else {
		shm.SetLeap(ntpdshm.LeapNotInSync)
	}

	return nil
}
