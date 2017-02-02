package webntp

import (
	"encoding/json"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	HTTPClient *http.Client
}

type Result struct {
	Offset time.Duration
	Delay  time.Duration
}

func (c *Client) Get(uri string) (Result, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return Result{}, err
	}

	if u.Scheme == "ws" || u.Scheme == "wss" {
		return c.getWebsocket(uri)
	}
	return c.getHTTP(uri)
}

func (c *Client) getHTTP(uri string) (Result, error) {
	req, err := http.NewRequest("GET", uri, nil)
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
	ntpTime := time.Time(result.SendTime)
	delay := end.Sub(start)
	offset := start.Sub(ntpTime) + delay/2

	return Result{
		Delay:  delay,
		Offset: offset,
	}, nil
}

func (c *Client) getWebsocket(uri string) (Result, error) {
	conn, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		return Result{}, nil
	}

	// Send the request
	err = conn.WriteJSON(Timestamp(time.Now()))
	if err != nil {
		return Result{}, nil
	}

	// Parse the response
	var response Response
	err = conn.ReadJSON(&response)
	if err != nil {
		return Result{}, nil
	}
	start := time.Time(response.InitiateTime)
	end := time.Now()
	delay := end.Sub(start)
	offset := start.Sub(time.Time(response.SendTime)) + delay/2

	return Result{
		Delay:  delay,
		Offset: offset,
	}, nil
}
