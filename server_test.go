package webntp

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shogo82148/websocket"
)

func TestServer_TimeOverHTTPS(t *testing.T) {
	t.Parallel()

	s := NewServer()
	s.nowFunc = func() time.Time {
		return time.Unix(1234567891, 123123000)
	}

	req := httptest.NewRequest(http.MethodHead, "http://example.com/.well-known/time", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	got := w.Header().Get("X-HTTPSTIME")
	if got != "1234567891.123123" {
		t.Errorf("want %s, got %s", "1234567891.123123", got)
	}
}

func TestServer_JSON(t *testing.T) {
	t.Parallel()

	s := NewServer()
	s.nowFunc = func() time.Time {
		return time.Unix(1234567891, 0)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com/json?%f", 1234567890.0), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	want := map[string]any{
		"id":   "example.com",
		"it":   1234567890.0,
		"st":   1234567891.0,
		"time": 1234567891.0,
		"leap": 36.0,
		"next": 1483228800.0,
		"step": 1.0,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestServer_WebSocket(t *testing.T) {
	t.Parallel()

	s := NewServer()
	s.nowFunc = func() time.Time {
		return time.Unix(1234567891, 0)
	}
	ts := httptest.NewTestServer(t, s)

	ctx := t.Context()
	conn, _, err := websocket.Dial(ctx, "ws://example.com/websocket", &websocket.DialOptions{
		HTTPClient:   ts.Client(),
		Subprotocols: []string{Subprotocol},
	})
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	if err := conn.Write(ctx, websocket.MessageText, []byte("1234567890.0")); err != nil {
		t.Fatalf("failed to write to WebSocket: %v", err)
	}

	var got map[string]any
	typ, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read from WebSocket: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("want message type %d, got %d", websocket.MessageText, typ)
	}
	if err := json.Unmarshal(msg, &got); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	want := map[string]any{
		"id":   "example.com",
		"it":   1234567890.0,
		"st":   1234567891.0,
		"time": 1234567891.0,
		"leap": 36.0,
		"next": 1483228800.0,
		"step": 1.0,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}

	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatalf("failed to close WebSocket: %v", err)
	}
}

func TestServer_Shutdown(t *testing.T) {
	t.Parallel()

	s := NewServer()
	ts := httptest.NewTestServer(t, s)

	ctx := t.Context()
	conn, _, err := websocket.Dial(ctx, "ws://example.com/websocket", &websocket.DialOptions{
		HTTPClient:   ts.Client(),
		Subprotocols: []string{Subprotocol},
	})
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	// Wait until the server has registered the connection before shutting it down.
	if err := conn.Write(ctx, websocket.MessageText, nil); err != nil {
		t.Fatalf("failed to write to WebSocket: %v", err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("failed to read from WebSocket: %v", err)
	}

	readErr := make(chan error, 1)
	go func() {
		_, _, err := conn.Read(ctx)
		readErr <- err
	}()

	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned an error: %v", err)
	}

	err = <-readErr
	var closeErr websocket.CloseError
	if !errors.As(err, &closeErr) {
		t.Fatalf("want websocket.CloseError, got %v", err)
	}
	if closeErr.Code != websocket.StatusGoingAway {
		t.Errorf("want close status %d, got %d", websocket.StatusGoingAway, closeErr.Code)
	}
	if closeErr.Reason != "server shutting down" {
		t.Errorf("want close reason %q, got %q", "server shutting down", closeErr.Reason)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.conns) != 0 {
		t.Errorf("want no active connections, got %d", len(s.conns))
	}
}

func TestServer_ShutdownCanceled(t *testing.T) {
	t.Parallel()

	s := NewServer()
	ts := httptest.NewTestServer(t, s)

	ctx := t.Context()
	conn, _, err := websocket.Dial(ctx, "ws://example.com/websocket", &websocket.DialOptions{
		HTTPClient:   ts.Client(),
		Subprotocols: []string{Subprotocol},
	})
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	// Wait until the server has registered the connection before shutting it down.
	if err := conn.Write(ctx, websocket.MessageText, nil); err != nil {
		t.Fatalf("failed to write to WebSocket: %v", err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("failed to read from WebSocket: %v", err)
	}

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.Shutdown(canceledCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.conns) != 1 {
		t.Errorf("want one active connection, got %d", len(s.conns))
	}
}

func BenchmarkServer_JSON(b *testing.B) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/json", nil)
	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

func BenchmarkTimeOverHTTPS(b *testing.B) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodHead, "http://example.com/.well-known/time", nil)
	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}
