package webntp

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/shogo82148/websocket"
)

var lastLeapTime = Timestamp(time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC))

const timeout = 90 * time.Second

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
	s.mux.HandleFunc("GET /websocket", s.websocketHandler)
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

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithoutCancel(r.Context())
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{Subprotocol},
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to accept websocket connection", slog.Any("error", err))
		return
	}
	defer conn.CloseNow() //nolint:errcheck // cleanup
	conn.SetReadLimit(1024)

	slog.InfoContext(
		ctx, "websocket connection accepted",
		slog.String("remote", r.RemoteAddr),
		slog.String("subprotocol", conn.Subprotocol()),
		slog.String("user-agent", r.UserAgent()),
		slog.String("origin", r.Header.Get("Origin")),
	)

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	go func() {
		<-timer.C
		slog.InfoContext(ctx, "websocket connection closed due to timeout")
		_ = conn.Close(websocket.StatusNormalClosure, "timeout")
	}()

	for {
		typ, buf, err := conn.Read(ctx)
		if err != nil {
			if _, ok := errors.AsType[websocket.CloseError](err); ok {
				slog.InfoContext(ctx, "websocket connection closed", slog.Any("error", err))
				return
			}
			slog.ErrorContext(ctx, "failed to read from websocket connection", slog.Any("error", err))
			return
		}
		if typ != websocket.MessageText {
			slog.ErrorContext(ctx, "unexpected message type", slog.Any("type", typ))
			_ = conn.Close(websocket.StatusProtocolError, "unexpected message type")
			return
		}
		timer.Reset(timeout)

		it := zeroEpochTime
		if len(buf) > 0 {
			t, err := ParseTimestamp(string(buf))
			if err != nil {
				slog.ErrorContext(ctx, "failed to parse timestamp", slog.Any("error", err))
				_ = conn.Close(websocket.StatusInvalidFramePayloadData, "failed to parse timestamp")
				return
			}
			it = t
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

		buf, err = json.Marshal(resp)
		if err != nil {
			slog.ErrorContext(ctx, "failed to marshal response", slog.Any("error", err))
			_ = conn.Close(websocket.StatusInternalError, "failed to marshal response")
			return
		}
		if err := conn.Write(ctx, websocket.MessageText, buf); err != nil {
			slog.ErrorContext(ctx, "failed to write to websocket connection", slog.Any("error", err))
			return
		}
	}
}
