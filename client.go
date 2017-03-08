package webntp

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
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
	mu     sync.Mutex
	conn   *websocket.Conn
	pong   chan struct{}
	result chan Result
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

// GetMulti gets synchronization information.
// It improve accuracy by calling the Get method many times.
func (c *Client) GetMulti(uri string, samples int) (Result, error) {
	var s int64
	if err := binary.Read(crand.Reader, binary.LittleEndian, &s); err != nil {
		s = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(s))

	results := make([]Result, samples)
	minDelay := time.Duration(1<<63 - 1) // the maximum number of time.Duration
	for i := range results {
		var err error
		results[i], err = c.Get(uri)
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

	var result Result
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
	b, err := Timestamp(time.Now()).MarshalJSON()
	if err != nil {
		return Result{}, err
	}
	err = conn.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		return Result{}, err
	}

	select {
	case result, ok := <-wsConn.result:
		if ok {
			return result, nil
		}
		return Result{}, errors.New("webntp: connection is closed")
	case <-time.After(10 * time.Second):
		return Result{}, errors.New("webntp: timeout")
	}
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

	dialer := c.Dialer
	if dialer == nil {
		dialer = DefaultDialer
	}

	var err error
	for i := 0; i < 3; i++ {
		err = conn.dial(dialer, uri)
		if err != nil {
			// retry
			conn.conn.Close()
			conn.conn = nil
			continue
		}

		// success
		return conn, nil
	}

	// give up :(
	return nil, err
}

func (conn *wsConn) dial(dialer *websocket.Dialer, uri string) error {
	if conn.conn == nil {
		conn2, _, err := dialer.Dial(uri, nil)
		if err != nil {
			return err
		}
		conn.pong = make(chan struct{}, 1)
		conn.result = make(chan Result, 1)
		conn.conn = conn2
		go conn.readLoop()
	}

	// check the connection is now available.
	err := conn.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
	if err != nil {
		return err
	}
	select {
	case _, ok := <-conn.pong:
		if !ok {
			return errors.New("webntp: connection is closed")
		}
	case <-time.After(10 * time.Second):
		return errors.New("webntp: pong timeout")
	}
	return nil
}

func (conn *wsConn) readLoop() {
	conn.conn.SetPongHandler(func(string) error {
		select {
		case conn.pong <- struct{}{}:
		default:
		}
		return nil
	})
	defer close(conn.pong)
	defer close(conn.result)

	var buf [1024]byte
	conn.conn.SetReadLimit(int64(len(buf)))
	for {
		// read the response.
		_, r, err := conn.conn.NextReader()
		if err != nil {
			return
		}

		end := time.Now()
		var n int
		for n < len(buf) && err == nil {
			var nn int
			nn, err = r.Read(buf[n:])
			n += nn
		}
		if err != io.EOF {
			return
		}

		// parse the response.
		var response Response
		if err := json.Unmarshal(buf[:n], &response); err != nil {
			return
		}
		start := time.Time(response.InitiateTime)
		delay := end.Sub(start)
		offset := time.Time(response.SendTime).Sub(start) - delay/2

		conn.result <- Result{
			Delay:  delay,
			Offset: offset,
		}
	}
}
