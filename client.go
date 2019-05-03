package webntp

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// clientStartTime is used by tests.
var clientStartTime = time.Now

// clientEndTime is used by tests.
var clientEndTime = time.Now

// Client is a webntp client.
type Client struct {
	HTTPClient *http.Client
	Dialer     *websocket.Dialer
}

// DefaultDialer is a dialer for webntp.
var DefaultDialer = &websocket.Dialer{
	Proxy:        http.ProxyFromEnvironment,
	Subprotocols: []string{Subprotocol},
}

// Result is the result of synchronization.
type Result struct {
	Offset    time.Duration
	Delay     time.Duration
	NextLeap  time.Time
	TAIOffset time.Duration
	Step      int
}

// Get gets synchronization information.
func (c *Client) Get(ctx context.Context, uri string) (Result, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return Result{}, err
	}

	if u.Scheme == "ws" || u.Scheme == "wss" {
		return c.getWebsocket(ctx, uri)
	}
	return c.getHTTP(ctx, uri)
}

// GetMulti gets synchronization information.
// It improve accuracy by calling the Get method many times.
func (c *Client) GetMulti(ctx context.Context, uri string, samples int) (Result, error) {
	var s int64
	if err := binary.Read(crand.Reader, binary.LittleEndian, &s); err != nil {
		s = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(s))

	results := make([]Result, samples)
	minDelay := time.Duration(1<<63 - 1) // the maximum number of time.Duration
	for i := range results {
		var err error
		results[i], err = c.Get(ctx, uri)
		if err != nil {
			return Result{}, err
		}
		if results[i].Delay < minDelay {
			minDelay = results[i].Delay
		}
		if i < samples-1 {
			d := r.Int63n(int64(time.Second))
			time.Sleep(time.Duration(d))
		}
	}

	result := results[len(results)-1]
	var num int
	for _, r := range results {
		if r.Delay >= minDelay*2 {
			// this sample may be re-sent. ignore it.
			continue
		}
		result.Delay += r.Delay
		result.Offset += r.Offset
		num++
	}
	result.Delay /= time.Duration(num)
	result.Offset /= time.Duration(num)
	return result, nil
}

func (c *Client) getHTTP(ctx context.Context, uri string) (Result, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "webntp.shogo82148.com")

	// Install ClientTrace
	var start, end time.Time
	trace := &httptrace.ClientTrace{
		WroteRequest:         func(info httptrace.WroteRequestInfo) { start = clientStartTime() },
		GotFirstResponseByte: func() { end = clientEndTime() },
	}
	ctx = httptrace.WithClientTrace(ctx, trace)
	req = req.WithContext(ctx)

	// Send the request
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	// Parse the response
	var result Response
	dec := json.NewDecoder(resp.Body)
	if err = dec.Decode(&result); err != nil {
		return Result{}, err
	}
	ntpTime := time.Time(result.SendTime)
	if ntpTime.IsZero() {
		ntpTime = time.Time(result.Time) // fallback htptime
	}
	delay := end.Sub(start)
	offset := ntpTime.Sub(start) - delay/2

	return Result{
		Delay:     delay,
		Offset:    offset,
		NextLeap:  time.Time(result.Next),
		TAIOffset: time.Duration(result.Leap) * time.Second,
		Step:      result.Step,
	}, nil
}

func (c *Client) getWebsocket(ctx context.Context, uri string) (Result, error) {
	dialer := c.Dialer
	if dialer == nil {
		dialer = DefaultDialer
	}
	conn, _, err := dialer.DialContext(ctx, uri, nil)
	if err != nil {
		return Result{}, err
	}
	defer conn.Close()

	// Send the request
	start := clientStartTime()
	b, err := Timestamp(start).MarshalJSON()
	if err != nil {
		return Result{}, err
	}
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return Result{}, err
	}

	// Receive the response
	var result Response
	if err := conn.ReadJSON(&result); err != nil {
		return Result{}, nil
	}
	end := clientEndTime()

	ntpTime := time.Time(result.SendTime)
	if ntpTime.IsZero() {
		ntpTime = time.Time(result.Time) // fallback htptime
	}
	delay := end.Sub(start)
	offset := ntpTime.Sub(start) - delay/2
	return Result{
		Delay:     delay,
		Offset:    offset,
		NextLeap:  time.Time(result.Next),
		TAIOffset: time.Duration(result.Leap) * time.Second,
		Step:      result.Step,
	}, nil
}
