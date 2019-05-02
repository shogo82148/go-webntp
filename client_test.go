package webntp

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func BenchmarkGet(b *testing.B) {
	s := &Server{
		LeapSecondsPath: "leap-seconds.list",
		LeapSecondsURL:  "https://www.ietf.org/timezones/data/leap-seconds.list",
	}
	s.Start()
	defer s.Close()
	ts := httptest.NewServer(s)
	defer ts.Close()

	c := &Client{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ts.URL)
	}
}

func BenchmarkGetWS(b *testing.B) {
	s := &Server{
		LeapSecondsPath: "leap-seconds.list",
		LeapSecondsURL:  "https://www.ietf.org/timezones/data/leap-seconds.list",
	}
	s.Start()
	defer s.Close()
	ts := httptest.NewServer(s)
	defer ts.Close()
	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"
	wsURL := u.String()

	c := &Client{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(wsURL)
	}
}
