package webntp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
)

func TestServer_ServeHTTP(t *testing.T) {
	defer func(f func() time.Time) { timeNow = f }(timeNow)
	timeNow = func() time.Time {
		return time.Unix(1234567891, 0)
	}

	s := &Server{}
	s.Start()
	defer s.Close()

	want := map[string]interface{}{
		"id":   "example.com",
		"it":   1234567890.0,
		"st":   1234567891.0,
		"time": 1234567891.0,
		"leap": 0.0,
		"next": 0.0,
		"step": 0.0,
	}
	testServeHTTP(t, s, 1234567890.0, want)
	testServeWebSocket(t, s, 1234567890.0, want)
}

func testServeHTTP(t *testing.T, s *Server, it float64, want map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com/foo?%f", it), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("unexpected status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func testServeWebSocket(t *testing.T, s *Server, it float64, want map[string]interface{}) {
	t.Helper()
	ts := httptest.NewServer(s)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"
	wsURL := u.String()

	dialer := &websocket.Dialer{
		Proxy:        http.ProxyFromEnvironment,
		Subprotocols: []string{Subprotocol},
	}
	h := http.Header{}
	h.Set("Host", "example.com")
	conn, _, err := dialer.Dial(wsURL, h)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(it); err != nil {
		t.Fatal(err)
	}

	var got map[string]interface{}
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func BenchmarkServeHTTP(b *testing.B) {
	s := &Server{
		LeapSecondsPath: "leap-seconds.list",
		LeapSecondsURL:  "https://www.ietf.org/timezones/data/leap-seconds.list",
	}
	s.Start()
	defer s.Close()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}
