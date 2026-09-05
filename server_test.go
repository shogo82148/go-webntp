package webntp

import (
	"encoding/json/v2"
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
