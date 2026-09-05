package webntp

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"time"
)

var lastLeapTime = Timestamp(time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC))

type Server struct {
	mux     *http.ServeMux
	nowFunc func() time.Time
}

func NewServer() *Server {
	s := &Server{
		mux:     http.NewServeMux(),
		nowFunc: time.Now,
	}
	s.mux.HandleFunc("HEAD /.well-known/time", s.timeOverHTTP)
	s.mux.HandleFunc("GET /json", s.jsonHandler)
	return s
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(rw, req)
}

// Time over HTTPS
// https://phk.freebsd.dk/time/20151129/
func (s *Server) timeOverHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	now := s.nowFunc()
	w.Header().Set("X-Httpstime", Timestamp(now).String())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) jsonHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	it := zeroEpochTime
	if q := r.URL.RawQuery; q != "" {
		t, err := ParseTimestamp(q)
		if err == nil {
			it = t
		}
	}

	now := s.nowFunc()
	resp := &Response{
		ID:           r.Host,
		InitiateTime: it,
		SendTime:     Timestamp(now),
		Time:         Timestamp(now),

		// Leap seconds are scheduled to be abolished,
		// but for backward compatibility, it returns information on the last inserted leap second.
		Leap: 36,
		Next: lastLeapTime,
		Step: 1,
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.ErrorContext(ctx, "failed to marshal response", slog.Any("error", err))
		return
	}
	if _, err := w.Write(buf); err != nil {
		slog.ErrorContext(ctx, "failed to write response", slog.Any("error", err))
	}
}
