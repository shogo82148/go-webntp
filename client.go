package webntp

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
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
	if strings.EqualFold(u.Scheme, "time") {
		return c.getTimeOverHTTP(ctx, u)
	}
	return c.getJSON(ctx, u)
}

func (c *Client) getWebSocket(ctx context.Context, u *url.URL) (Result, error) {
	return Result{}, nil
}

func (c *Client) getTimeOverHTTP(ctx context.Context, u *url.URL) (Result, error) {
	return Result{}, nil
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
