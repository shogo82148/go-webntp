package webntp

import (
	"encoding/json"
	"net/http"
	"net/http/httptrace"
	"time"
)

type Client struct {
	HTTPClient *http.Client
}

type Result struct {
	Offset time.Duration
	Delay  time.Duration
}

func (c *Client) Get(url string) (Result, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Result{}, err
	}

	// Install ClientTrace
	var start, end time.Time
	trace := &httptrace.ClientTrace{
		WroteRequest:         func(info httptrace.WroteRequestInfo) { start = time.Now() },
		GotFirstResponseByte: func() { end = time.Now() },
	}
	ctx := httptrace.WithClientTrace(req.Context(), trace)
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
	ntpTime := timestampToTime(result.SendTime)
	delay := end.Sub(start)
	offset := start.Sub(ntpTime) + delay/2

	return Result{
		Delay:  delay,
		Offset: offset,
	}, nil
}
