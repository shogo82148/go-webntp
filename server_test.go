package webntp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestServer_ServeHTTP(t *testing.T) {
	defer func(f func() time.Time) { timeNow = f }(timeNow)
	timeNow = func() time.Time {
		return time.Unix(1234567891, 0)
	}

	s := &Server{}
	s.Start()
	defer s.Close()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo?1234567890.0", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("unexpected status code: want %d, got %d", http.StatusOK, w.Code)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"id":   "example.com",
		"it":   1234567890.0,
		"st":   1234567891.0,
		"time": 1234567891.0,
		"leap": 0.0,
		"next": 0.0,
		"step": 0.0,
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
