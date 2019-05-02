package webntp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{Subprotocol},
}

// Server is a webntp server.
type Server struct {
	Upgrader *websocket.Upgrader

	// path for leap-seconds.list cache
	LeapSecondsPath string

	// url for leap-seconds.list
	LeapSecondsURL string

	leapSecondsList atomic.Value
	ctx             context.Context
	cancel          context.CancelFunc
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if websocket.IsWebSocketUpgrade(req) {
		s.handleWebsocket(rw, req)
		return
	}

	now := time.Now()
	leap := s.getLeapSecond(now)
	start := zeroEpochTime
	if q := req.URL.RawQuery; q != "" {
		err := start.UnmarshalJSON([]byte(strings.TrimSpace(q)))
		if err != nil {
			return
		}
	}
	res := &Response{
		ID:           req.Host,
		InitiateTime: start,
		SendTime:     Timestamp(now),
		Time:         Timestamp(now),
		Leap:         leap.Leap,
		Next:         Timestamp(leap.At),
		Step:         leap.Step,
	}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Header().Set("Cache-Control", "no-cache, no-store")
	enc := json.NewEncoder(rw)
	enc.Encode(res)
}

func (s *Server) handleWebsocket(rw http.ResponseWriter, req *http.Request) {
	upgrader := s.Upgrader
	if upgrader == nil {
		upgrader = defaultUpgrader
	}
	conn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Println("upgrade error: ", err)
		return
	}
	defer conn.Close()

	for {
		err := s.handleWebsocketConn(conn, req.Host)
		if _, ok := err.(*websocket.CloseError); ok {
			return
		}
		if err != nil {
			log.Println("websocket error: ", err)
			return
		}
	}
}

func (s *Server) handleWebsocketConn(conn *websocket.Conn, host string) error {
	_, r, err := conn.NextReader()
	if err != nil {
		return err
	}

	// parse the request
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	var start Timestamp
	err = start.UnmarshalJSON(bytes.TrimSpace(buf))
	if err != nil {
		return err
	}

	// send the response
	now := time.Now()
	leap := s.getLeapSecond(now)
	res := &Response{
		ID:           host,
		InitiateTime: start,
		SendTime:     Timestamp(now),
		Time:         Timestamp(now),
		Leap:         leap.Leap,
		Next:         Timestamp(leap.At),
		Step:         leap.Step,
	}
	return conn.WriteJSON(res)
}

// Start starts fetching leap-seconds.list
func (s *Server) Start() error {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// warm up json encoder.
	json.Marshal(&Response{})

	if err := s.readLeapSecondsCache(); err != nil {
		return err
	}
	if s.LeapSecondsURL == "" {
		return nil
	}
	go s.loopLeapSeconds()
	return nil
}

// Close closes the server.
func (s *Server) Close() error {
	s.cancel()
	return nil
}

func (s *Server) getLeapSecond(now time.Time) LeapSecond {
	list, ok := s.leapSecondsList.Load().(*LeapSecondsList)
	if !ok {
		return LeapSecond{
			At: time.Time(zeroEpochTime),
		}
	}
	var i int
	for i = len(list.LeapSeconds); i > 0; i-- {
		if list.LeapSeconds[i-1].At.Before(now) {
			break
		}
	}
	if i == len(list.LeapSeconds) {
		return list.LeapSeconds[i-1]
	}
	return list.LeapSeconds[i]
}

func (s *Server) readLeapSecondsCache() error {
	if s.LeapSecondsPath == "" {
		return nil
	}

	f, err := os.Open(s.LeapSecondsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	list, err := ParseLeapSecondsList(f)
	if err != nil {
		return err
	}
	s.leapSecondsList.Store(list)
	return nil
}

func (s *Server) loopLeapSeconds() {
	err := s.checkAndFetch(s.ctx, time.Now())
	if err != nil {
		log.Println(err)
	}
	timer := time.NewTimer(24 * time.Hour)
	defer timer.Stop()
	for {
		select {
		case now := <-timer.C:
			err := s.checkAndFetch(s.ctx, now)
			if err != nil {
				log.Println(err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// checkAndFetch checks the leap seconds list is expired,
// and fetch new list if needed.
func (s *Server) checkAndFetch(ctx context.Context, now time.Time) error {
	list, ok := s.leapSecondsList.Load().(*LeapSecondsList)
	if !ok || now.After(list.ExpireAt) {
		log.Printf("fetch %s", s.LeapSecondsURL)
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		err := s.fetchLeapSeconds(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) fetchLeapSeconds(ctx context.Context) error {
	// open cache file
	name := fmt.Sprintf("%s.%d", s.LeapSecondsPath, time.Now().Unix())
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(name)
	}()

	// get the new list.
	req, err := http.NewRequest(http.MethodGet, s.LeapSecondsURL, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// write to cache, and parse it.
	r := io.TeeReader(resp.Body, f)
	list, err := ParseLeapSecondsList(r)
	if err != nil {
		return err
	}
	s.leapSecondsList.Store(list)

	// update cache file
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, s.LeapSecondsPath); err != nil {
		return err
	}

	return nil
}
