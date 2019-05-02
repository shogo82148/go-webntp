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

func TestServer_ServeHTTP_with_leap(t *testing.T) {
	now := time.Now()
	defer func(f func() time.Time) { timeNow = f }(timeNow)
	timeNow = func() time.Time {
		return now
	}

	s := &Server{
		LeapSecondsPath: "testdata/leap-seconds-2019-05-02.list",
	}
	s.Start()
	defer s.Close()

	t.Run("before leap second", func(t *testing.T) {
		now, _ = time.Parse(time.RFC3339, "2015-06-30T23:59:59Z")
		want := map[string]interface{}{
			"id":   "example.com",
			"it":   1234567890.0,
			"st":   1435708799.0, // 2015-06-30T23:59:59Z
			"time": 1435708799.0, // 2015-06-30T23:59:59Z
			"leap": 35.0,
			"next": 1435708800.0, // next leap second is on 2015-07-01
			"step": 1.0,
		}
		testServeHTTP(t, s, 1234567890.0, want)
		testServeWebSocket(t, s, 1234567890.0, want)
	})

	t.Run("after leap second", func(t *testing.T) {
		now, _ = time.Parse(time.RFC3339, "2015-07-01T00:00:00Z")
		want := map[string]interface{}{
			"id":   "example.com",
			"it":   1234567890.0,
			"st":   1435708800.0, // 2015-01-01T00:00:00Z
			"time": 1435708800.0, // 2015-01-01T00:00:00Z
			"leap": 36.0,
			"next": 1483228800.0, // next leap second is on 2017-01-01
			"step": 1.0,
		}
		testServeHTTP(t, s, 1234567890.0, want)
		testServeWebSocket(t, s, 1234567890.0, want)
	})

	t.Run("before leap second", func(t *testing.T) {
		now, _ = time.Parse(time.RFC3339, "2016-12-31T23:59:59Z")
		want := map[string]interface{}{
			"id":   "example.com",
			"it":   1234567890.0,
			"st":   1483228799.0, // 2016-12-31T23:59:59Z
			"time": 1483228799.0, // 2016-12-31T23:59:59Z
			"leap": 36.0,
			"next": 1483228800.0, // next leap second is on 2017-01-01
			"step": 1.0,
		}
		testServeHTTP(t, s, 1234567890.0, want)
		testServeWebSocket(t, s, 1234567890.0, want)
	})

	t.Run("next leap second is not scheduled", func(t *testing.T) {
		now, _ = time.Parse(time.RFC3339, "2017-01-01T00:00:00Z")
		want := map[string]interface{}{
			"id":   "example.com",
			"it":   1234567890.0,
			"st":   1483228800.0, // 2017-01-01T00:00:00Z
			"time": 1483228800.0, // 2015-01-01T00:00:00Z
			"leap": 36.0,
			"next": 1483228800.0, // next leap second is not scheduled, so return last one. It is on 2017-01-01
			"step": 1.0,
		}
		testServeHTTP(t, s, 1234567890.0, want)
		testServeWebSocket(t, s, 1234567890.0, want)
	})
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
