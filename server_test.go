package webntp

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkServeHTTP(b *testing.B) {
	s := &Server{
		LeapSecondsPath: "leap-seconds.list",
		LeapSecondsURL:  "https://www.ietf.org/timezones/data/leap-seconds.list",
	}
	s.Start()
	defer s.Close()
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}
