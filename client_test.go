package webntp

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	defer func(f func() time.Time) { serverTime = f }(serverTime)
	serverTime = func() time.Time {
		return time.Unix(1234567895, 0)
	}

	defer func(f func() time.Time) { clientStartTime = f }(clientStartTime)
	clientStartTime = func() time.Time {
		return time.Unix(1234567890, 0)
	}
	defer func(f func() time.Time) { clientEndTime = f }(clientEndTime)
	clientEndTime = func() time.Time {
		return time.Unix(1234567892, 0)
	}

	s := &Server{}
	s.Start()
	defer s.Close()
	ts := httptest.NewServer(s)
	defer ts.Close()

	c := &Client{}
	result, err := c.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != -4*time.Second {
		t.Errorf("unexpected offset, want %s, got %s", -4*time.Second, result.Offset)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("unexpected delay, want %s, got %s", 2*time.Second, result.Delay)
	}
}

func TestGetWebSocket(t *testing.T) {
	defer func(f func() time.Time) { serverTime = f }(serverTime)
	serverTime = func() time.Time {
		return time.Unix(1234567895, 0)
	}

	defer func(f func() time.Time) { clientStartTime = f }(clientStartTime)
	clientStartTime = func() time.Time {
		return time.Unix(1234567890, 0)
	}
	defer func(f func() time.Time) { clientEndTime = f }(clientEndTime)
	clientEndTime = func() time.Time {
		return time.Unix(1234567892, 0)
	}

	s := &Server{}
	s.Start()
	defer s.Close()
	ts := httptest.NewServer(s)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"
	wsURL := u.String()
	c := &Client{}
	result, err := c.Get(wsURL)
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != -4*time.Second {
		t.Errorf("unexpected offset, want %s, got %s", -4*time.Second, result.Offset)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("unexpected delay, want %s, got %s", 2*time.Second, result.Delay)
	}
}

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
