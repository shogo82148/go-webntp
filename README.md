[![Build Status](https://travis-ci.com/shogo82148/go-webntp.svg?branch=master)](https://travis-ci.com/shogo82148/go-webntp)
[![Go Reference](https://pkg.go.dev/badge/github.com/shogo82148/go-webntp.svg)](https://pkg.go.dev/github.com/shogo82148/go-webntp)

# WebNTP

WebNTP is NTP(-like) service over HTTP/WebSocket.

## Synopsis

First, `go install` and start the WebNTP Server.

```shell
go install github.com/shogo82148/go-webntp/cmd/webntp
webntp -serve :8080
```

Sync with the server over HTTP.

```plain
$ webntp http://localhost:8080/json
{"time":"2026-09-06T21:22:11.176866+09:00","level":"INFO","msg":"got time from host","server":"http://localhost:8080/json","offset":-0.000313125,"delay":0.00216175}
{"time":"2026-09-06T21:22:11.177056+09:00","level":"INFO","msg":"best time","server":"http://localhost:8080/json","local":"2026-09-06T21:22:11.177054+09:00","remote":"2026-09-06T21:22:11.176740875+09:00","offset":-0.000313125,"delay":0.00216175}
```

Sync with the server over WebSocket.

```plain
$ webntp ws://localhost:8080/websocket
{"time":"2026-09-06T21:22:53.587352+09:00","level":"INFO","msg":"got time from host","server":"ws://localhost:8080/websocket","offset":0.000008,"delay":0.000212}
{"time":"2026-09-06T21:22:53.587536+09:00","level":"INFO","msg":"best time","server":"ws://localhost:8080/websocket","local":"2026-09-06T21:22:53.587534+09:00","remote":"2026-09-06T21:22:53.587542+09:00","offset":0.000008,"delay":0.000212}
```

## Usage

```plain
$ webntp --help
  -help
    	show help
  -serve string
    	server host name
  -version
    	show the version
```

## Protocol

### JSON over HTTP

The WebNTP clients access to `http://example.com/?<timestamp>`,
and then the WebNTP server returns the time information formatted by JSON.

- `id`: hostname of the server
- `it`: the client's timestamp of the request transmission
- `st`: the server's timestamp
- `leap`: the seconds of TAI - UTC (**before** `next`)
- `next`: the timestamp of the next or last leap second
- `step`: positive leap second: 1, negative leap second: -1

Example:

```plain
$ curl -s 'http://localhost:8080/json?1788697606.057' | jq .
{
  "id": "localhost:8080",
  "it": 1788697606.057,
  "st": 1788697671.734335,
  "time": 1788697671.734335,
  "leap": 36,
  "next": 1483228800,
  "step": 1
}
```

It is based on [the document of http/https service](https://jjy.nict.go.jp/QandA/reference/http-archive.html) by NICT (the National Institute of Information and Communications Technology).
(the content is written in Japanese)

### JSON over WebSocket

The WebNTP clients send a message including timestamp,
and then the WebNTP server returns the time information formatted by JSON.
JSON format is same as HTTP response.

Example:

```plain
$ wscat --connect localhost:8080
connected (press CTRL+C to quit)
> 1558915619.944235
< {"id":"localhost:8080","it":1558915619.944235,"st":1558916776.363423,"time":1558916776.363423,"leap":36,"next":1483228800.000000,"step":1}
```

### Time over HTTPS with Improved timekeeping response

The clients send `HEAD /.well-known/time` HTTP request,
and then the server returns the timestamp in the response header.

```plain
X-HTTPSTIME: <timestamp>
```

Example:

```plain
$ curl -I localhost:8080/.well-known/time
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: *
Access-Control-Expose-Headers: X-Httpstime
Cache-Control: no-cache
X-Httpstime: 1788697967.925993
Date: Sun, 06 Sep 2026 12:32:47 GMT
```

It is based on [Time over HTTPS specification](http://phk.freebsd.dk/time/20151129/).

## License

This software is released under the MIT License, see LICENSE.

## SEE ALSO

- [Time over HTTPS](http://phk.freebsd.dk/time/20151129/)
- [htp](http://www.vervest.org/htp/)
- [http/httpsを利用した時刻配信(アーカイブ)](https://jjy.nict.go.jp/QandA/reference/http-archive.html) (The time server using http/https (archived))
