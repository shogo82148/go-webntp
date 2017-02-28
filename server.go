package webntp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Server is a webntp server.
type Server struct {
	Upgrader *websocket.Upgrader

	// path for leap-seconds.list cache
	LeapSecondsPath string

	// url for leap-seconds.list
	LeapSecondsURL string

	leapSecondsList atomic.Value
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if websocket.IsWebSocketUpgrade(req) {
		s.handleWebsocket(rw, req)
		return
	}

	now := time.Now()
	leap := s.getLeapSecond(now)
	res := &Response{
		ID:           req.Host,
		InitiateTime: zeroEpochTime,
		SendTime:     Timestamp(now),
		Leap:         leap.Leap,
		Next:         Timestamp(leap.At),
		Step:         leap.Step,
	}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
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
		Leap:         leap.Leap,
		Next:         Timestamp(leap.At),
		Step:         leap.Step,
	}
	return conn.WriteJSON(res)
}

// Start starts fetching leap-seconds.list
func (s *Server) Start() error {
	if s.LeapSecondsURL == "" {
		return nil
	}
	go s.fetchLeapSeconds()
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

func (s *Server) fetchLeapSeconds() {
	resp, err := http.Get(s.LeapSecondsURL)
	if err != nil {
		log.Printf("error while getting from %s: %v", s.LeapSecondsURL, err)
		return
	}
	defer resp.Body.Close()
	list, err := ParseLeapSecondsList(resp.Body)
	if err != nil {
		log.Printf("parse leap-sencond.list error: %v", err)
	}
	s.leapSecondsList.Store(list)
}
