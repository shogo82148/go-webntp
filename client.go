package webntp

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"github.com/shogo82148/websocket"
)

// Client is a webntp client.
type Client struct {
	HTTPClient *http.Client
	startTime  func() time.Time
	endTime    func() time.Time
}

// Result is the result of synchronization.
type Result struct {
	Offset time.Duration
	Delay  time.Duration
}

func NewClient() *Client {
	return &Client{
		HTTPClient: http.DefaultClient,
		startTime:  time.Now,
		endTime:    time.Now,
	}
}

// Get gets the synchronization information.
func (c *Client) Get(ctx context.Context, uri string) (Result, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return Result{}, err
	}

	if strings.EqualFold(u.Scheme, "ws") || strings.EqualFold(u.Scheme, "wss") {
		return c.getWebSocket(ctx, u)
	}
	if u.Path == "/.well-known/time" {
		return c.getTimeOverHTTP(ctx, u)
	}
	return c.getJSON(ctx, u)
}

func (c *Client) getWebSocket(ctx context.Context, u *url.URL) (Result, error) {
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPClient:   c.HTTPClient,
		Subprotocols: []string{Subprotocol},
	})
	if err != nil {
		return Result{}, err
	}
	defer conn.CloseNow() //nolint:errcheck // cleanup

	conn.SetReadLimit(1024)

	start := c.startTime()
	if err := conn.Write(ctx, websocket.MessageText, []byte(Timestamp(start).String())); err != nil {
		return Result{}, err
	}

	typ, msg, err := conn.Read(ctx)
	end := c.endTime()
	if err != nil {
		return Result{}, err
	}
	if typ != websocket.MessageText {
		return Result{}, fmt.Errorf("webntp: unexpected message type: %d", typ)
	}

	var result Response
	dec := jsontext.NewDecoder(bytes.NewReader(msg))
	if err := json.UnmarshalDecode(dec, &result); err != nil {
		return Result{}, err
	}

	// Calculate the offset and delay
	sendTime := time.Time(result.SendTime)
	if sendTime.IsZero() {
		sendTime = time.Time(result.Time) // fall back to htptime
	}
	delay := end.Sub(start)
	offset := sendTime.Sub(end) + delay/2

	if err := conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
		return Result{}, err
	}
	return Result{
		Offset: offset,
		Delay:  delay,
	}, nil
}

func (c *Client) getTimeOverHTTP(ctx context.Context, u *url.URL) (Result, error) {
	// Install ClientTrace
	var start, end time.Time
	trace := &httptrace.ClientTrace{
		WroteRequest:         func(info httptrace.WroteRequestInfo) { start = c.startTime() },
		GotFirstResponseByte: func() { end = c.endTime() },
	}
	ctx = httptrace.WithClientTrace(ctx, trace)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		return Result{}, err
	}

	// Send the request
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close() //nolint:errcheck // cleanup

	// Parse the response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("webntp: unexpected status code: %d", resp.StatusCode)
	}

	// Calculate the offset and delay
	t, err := ParseTimestamp(resp.Header.Get("X-Httpstime"))
	if err != nil {
		return Result{}, err
	}
	sendTime := time.Time(t)
	delay := end.Sub(start)
	offset := sendTime.Sub(end) + delay/2

	return Result{
		Offset: offset,
		Delay:  delay,
	}, nil
}

func (c *Client) getJSON(ctx context.Context, u *url.URL) (Result, error) {
	// Install ClientTrace
	var start, end time.Time
	trace := &httptrace.ClientTrace{
		WroteRequest:         func(info httptrace.WroteRequestInfo) { start = c.startTime() },
		GotFirstResponseByte: func() { end = c.endTime() },
	}
	ctx = httptrace.WithClientTrace(ctx, trace)

	now := c.startTime()
	u.RawQuery = Timestamp(now).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}

	// Send the request
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close() //nolint:errcheck // cleanup

	// Parse the response
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("webntp: unexpected status code: %d", resp.StatusCode)
	}
	var result Response
	dec := jsontext.NewDecoder(resp.Body)
	if err := json.UnmarshalDecode(dec, &result); err != nil {
		return Result{}, err
	}

	// Calculate the offset and delay
	sendTime := time.Time(result.SendTime)
	if sendTime.IsZero() {
		sendTime = time.Time(result.Time) // fall back to htptime
	}
	delay := end.Sub(start)
	offset := sendTime.Sub(end) + delay/2

	return Result{
		Offset: offset,
		Delay:  delay,
	}, nil
}
