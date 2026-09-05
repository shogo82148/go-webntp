package webntp

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetJSON(t *testing.T) {
	c := NewClient()
	c.startTime = func() time.Time {
		return time.Unix(1234567890, 0)
	}
	c.endTime = func() time.Time {
		return time.Unix(1234567892, 0)
	}

	s := NewServer()
	s.nowFunc = func() time.Time {
		return time.Unix(1234567895, 0)
	}
	ts := httptest.NewTestServer(t, s)
	c.HTTPClient = ts.Client()

	ctx := t.Context()
	result, err := c.Get(ctx, "http://example.com/json")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if result.Offset != 4*time.Second {
		t.Errorf("Get() returned Offset = %v; want %v", result.Offset, 4*time.Second)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("Get() returned Delay = %v; want %v", result.Delay, 2*time.Second)
	}
}

func TestGetTimeOverHTTPS(t *testing.T) {
	c := NewClient()
	c.startTime = func() time.Time {
		return time.Unix(1234567890, 0)
	}
	c.endTime = func() time.Time {
		return time.Unix(1234567892, 0)
	}

	s := NewServer()
	s.nowFunc = func() time.Time {
		return time.Unix(1234567895, 0)
	}
	ts := httptest.NewTestServer(t, s)
	c.HTTPClient = ts.Client()

	ctx := t.Context()
	result, err := c.Get(ctx, "https://example.com/.well-known/time")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if result.Offset != 4*time.Second {
		t.Errorf("Get() returned Offset = %v; want %v", result.Offset, 4*time.Second)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("Get() returned Delay = %v; want %v", result.Delay, 2*time.Second)
	}
}

func TestGetWebSocket(t *testing.T) {
	c := NewClient()
	c.startTime = func() time.Time {
		return time.Unix(1234567890, 0)
	}
	c.endTime = func() time.Time {
		return time.Unix(1234567892, 0)
	}

	s := NewServer()
	s.nowFunc = func() time.Time {
		return time.Unix(1234567895, 0)
	}
	ts := httptest.NewTestServer(t, s)
	c.HTTPClient = ts.Client()

	ctx := t.Context()
	result, err := c.Get(ctx, "ws://example.com/websocket")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if result.Offset != 4*time.Second {
		t.Errorf("Get() returned Offset = %v; want %v", result.Offset, 4*time.Second)
	}
	if result.Delay != 2*time.Second {
		t.Errorf("Get() returned Delay = %v; want %v", result.Delay, 2*time.Second)
	}
}
