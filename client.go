package webntp

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"math/bits"
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
	// initialize seed
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

		// sleep a little
		if i < samples-1 {
			d := r.Int63n(int64(time.Second))
			time.Sleep(time.Duration(d))
		}
	}

	result := results[0]
	var num int64
	var delay, offset int128
	for _, r := range results {
		if r.Delay >= minDelay*2 {
			// the sample of this sample may be re-sent. ignore it.
			continue
		}
		delay = delay.Add(int64ToInt128(int64(r.Delay)))
		offset = offset.Add(int64ToInt128(int64(r.Offset)))
		num++
	}
	quo, _ := delay.Div(num)
	result.Delay = time.Duration(quo)
	quo, _ = offset.Div(num)
	result.Offset = time.Duration(quo)
	return result, nil
}

type int128 [2]uint64

func int64ToInt128(a int64) int128 {
	var neg bool
	if a < 0 {
		a *= -1
		neg = true
	}
	ret := int128{0, uint64(a)}
	if neg {
		ret = ret.Neg()
	}
	return ret
}

// Add returns a + b
func (a int128) Add(b int128) int128 {
	sum1, carry := bits.Add64(a[1], b[1], 0)
	sum0, _ := bits.Add64(a[0], b[0], carry)
	return int128{sum0, sum1}
}

func (a int128) Div(b int64) (quo, rem int64) {
	quoSign := int64(1)
	remSign := int64(1)
	if a[1] >= 1<<63 {
		quoSign *= -1
		remSign *= -1
		a = a.Neg()
	}
	if b < 0 {
		quoSign *= -1
		b *= -1
	}
	q, r := bits.Div64(a[0], a[1], uint64(b))
	quo = int64(q) * quoSign
	rem = int64(r) * remSign
	return
}

func (a int128) Neg() int128 {
	a[0] = ^a[0]
	a[1] = ^a[1]
	a[1]++
	if a[1] == 0 {
		a[0]++
	}
	return a
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
