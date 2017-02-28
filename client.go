package webntp

import (
	"encoding/json"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a webntp client.
type Client struct {
	HTTPClient *http.Client
	Dialer     *websocket.Dialer

	mu   sync.Mutex
	pool map[string]*wsConn
}

// DefaultDialer is a dialer for webntp.
var DefaultDialer = &websocket.Dialer{
	Proxy:        http.ProxyFromEnvironment,
	Subprotocols: []string{Subprotocol},
}

// Result is the result of synchronization.
type Result struct {
	Offset time.Duration
	Delay  time.Duration
}

type wsConn struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

// Get gets synchronization information.
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
	wsConn, err := c.getConn(uri)
	if err != nil {
		return Result{}, err
	}
	wsConn.mu.Lock()
	defer wsConn.mu.Unlock()
	conn := wsConn.conn

	// Send the request
	start := time.Now()
	err = conn.WriteJSON(Timestamp(start))
	if err != nil {
		return Result{}, nil
	}

	// Parse the response
	var response Response
	err = conn.ReadJSON(&response)
	if err != nil {
		return Result{}, nil
	}
	end := time.Now()
	delay := end.Sub(start)
	offset := start.Sub(time.Time(response.SendTime)) + delay/2

	return Result{
		Delay:  delay,
		Offset: offset,
	}, nil
}

func (c *Client) getConn(uri string) (*wsConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, ok := c.pool[uri]
	if !ok || conn == nil {
		if c.pool == nil {
			c.pool = make(map[string]*wsConn, 1)
		}
		conn = &wsConn{}
		c.pool[uri] = conn
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.conn == nil {
		dialer := c.Dialer
		if dialer == nil {
			dialer = DefaultDialer
		}
		conn2, _, err := dialer.Dial(uri, nil)
		if err != nil {
			return nil, err
		}
		conn.conn = conn2
	}
	return conn, nil
}
