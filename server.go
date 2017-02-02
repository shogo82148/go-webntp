package webntp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Server struct {
	Upgrader *websocket.Upgrader
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if websocket.IsWebSocketUpgrade(req) {
		s.handleWebsocket(rw, req)
		return
	}

	res := &Response{
		ID:           req.Host,
		InitiateTime: zeroEpochTime,
		SendTime:     Timestamp(time.Now()),
		Leap:         0,
		Next:         zeroEpochTime,
		Step:         0,
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
	res := &Response{
		ID:           host,
		InitiateTime: start,
		SendTime:     Timestamp(time.Now()),
		Leap:         0,
		Next:         zeroEpochTime,
		Step:         0,
	}
	return conn.WriteJSON(res)
}
