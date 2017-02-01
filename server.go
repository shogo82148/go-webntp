package webntp

import (
	"encoding/json"
	"net/http"
	"time"
)

type Server struct {
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	res := &Response{
		ID:           req.Host,
		InitiateTime: 0,
		SendTime:     timeToTimestamp(time.Now()),
		Leap:         0,
		Next:         0,
		Step:         0,
	}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(rw)
	enc.Encode(res)
}
