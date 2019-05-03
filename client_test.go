package webntp

import (
	"context"
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
	result, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != 4*time.Second {
		t.Errorf("unexpected offset, want %s, got %s", 4*time.Second, result.Offset)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("unexpected delay, want %s, got %s", 2*time.Second, result.Delay)
	}
}

func TestGetMulti(t *testing.T) {
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
	result, err := c.GetMulti(context.Background(), ts.URL, 3)
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != 4*time.Second {
		t.Errorf("unexpected offset, want %s, got %s", 4*time.Second, result.Offset)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("unexpected delay, want %s, got %s", 2*time.Second, result.Delay)
	}
}

func TestInt128Add(t *testing.T) {
	testcases := []struct {
		a    int128
		b    int128
		want int128
	}{
		{
			a:    int128{0, 0}, // 0
			b:    int128{0, 0}, // 0
			want: int128{0, 0}, // 0
		},
		{
			a:    int128{0, 1},
			b:    int128{0, 2},
			want: int128{0, 3},
		},

		// carry up
		{
			a:    int128{0, 0xFFFFFFFFFFFFFFFF},
			b:    int128{0, 1},
			want: int128{1, 0},
		},

		// negative integer
		{
			a:    int128{0, 1}.Neg(),
			b:    int128{0, 1},
			want: int128{0, 0},
		},
	}
	for _, tc := range testcases {
		got := tc.a.Add(tc.b)
		if got != tc.want {
			t.Errorf("want %v, got %v", tc.want, got)
		}
	}
}

func TestInt128Div(t *testing.T) {
	testcases := []struct {
		a   int128
		b   int64
		quo int64
		rem int64
	}{
		{
			a:   int128{0, 7},
			b:   3,
			quo: 2,
			rem: 1,
		},
		{
			a:   int128{0, 7},
			b:   -3,
			quo: -2,
			rem: 1,
		},
		{
			a:   int128{0, 7}.Neg(),
			b:   3,
			quo: -2,
			rem: -1,
		},
		{
			a:   int128{0, 7}.Neg(),
			b:   -3,
			quo: 2,
			rem: -1,
		},
	}
	for i, tc := range testcases {
		quo, rem := tc.a.Div(tc.b)
		if rem != tc.rem {
			t.Errorf("%d: unexpected rem, want %v, got %v", i, tc.rem, rem)
		}
		if quo != tc.quo {
			t.Errorf("%d: unexpected quo, want %v, got %v", i, tc.quo, quo)
		}
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
	result, err := c.Get(context.Background(), wsURL)
	if err != nil {
		t.Fatal(err)
	}
	if result.Offset != 4*time.Second {
		t.Errorf("unexpected offset, want %s, got %s", 4*time.Second, result.Offset)
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
		c.Get(context.Background(), ts.URL)
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
		c.Get(context.Background(), wsURL)
	}
}
