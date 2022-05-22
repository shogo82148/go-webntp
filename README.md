[![Build Status](https://travis-ci.com/shogo82148/go-webntp.svg?branch=master)](https://travis-ci.com/shogo82148/go-webntp)
[![Go Reference](https://pkg.go.dev/badge/github.com/shogo82148/go-webntp.svg)](https://pkg.go.dev/github.com/shogo82148/go-webntp)

# WebNTP

WebNTP is NTP(-like) service via HTTP/WebSocket.

## Synopsis

First, `go install` and start the WebNTP Server.

``` plain
$ go install github.com/shogo82148/go-webntp/cmd/webntp
$ webntp -serve :8080
```

Sync with the server via HTTP.

``` plain
$ webntp http://localhost:8080/
server http://localhost:8080/, offset -0.000066, delay 0.001453
2017-03-11 18:25:10.905049427 +0900 JST, server http://localhost:8080/, offset -0.000066
```

Sync with the server via WebSocket.

``` plain
$ webntp ws://localhost:8080/
server ws://localhost:8080/, offset -0.000013, delay 0.000288
2017-03-11 18:25:36.668531757 +0900 JST, server ws://localhost:8080/, offset -0.000013
```

I provide a public WebNTP API.

``` plain
$ webntp https://webntp.shogo82148.com/api
server https://webntp.shogo82148.com/api, offset -0.006376, delay 0.024411
2017-03-11 16:08:06.150393313 +0900 JST, server https://webntp.shogo82148.com/api, offset -0.006376
```

## Shared Memory support for ntpd

Add a new server to your `ntpd.conf`.

``` plain
server 127.127.28.2 noselect
fudge 127.127.28.2 refid PYTH stratum 10
```

Run WebNTP with `-shm 2` option.

``` plain
$ webntp -p 1 -shm 2 https://webntp.shogo82148.com/api
server https://webntp.shogo82148.com/api, offset -0.003258, delay 0.018910
```

ntpd starts syncing with WebNTP.

``` plain
$ ntpq -p
     remote           refid      st t when poll reach   delay   offset  jitter
==============================================================================
 SHM(2)          .PYTH.          10 l    2   64   17    0.000   -3.331   0.384
*webntp.shogo82. .NICT.           1 u   58   64   37   10.280    1.494   2.028
```


## Usage

``` plain
$ webntp --help
  -allow-cross-origin
    	allow cross origin request
  -help
    	show help
  -leap-second-path string
    	path for leap-seconds.list cache (default "leap-seconds.list")
  -leap-second-url string
    	url for leap-seconds.list (default "https://www.ietf.org/timezones/data/leap-seconds.list")
  -p int
    	Specify the number of samples (default 4)
  -serve string
    	server host name
  -shm uint
    	ntpd shared-memory-segment
```


## Protocol

### JSON over HTTP

The WebNTP clients access to `http://example.com/?<timestamp>`,
and then the WebNTP server returns the time information formatted by JSON.

- `id`: hostname of the server
- `it`: the client's timestamp of the request transmission
- `st`: the server's timestamp
- `leap`: the seconds of TAI - UTC (before `next`)
- `next`: the timestamp of the next or last leap second 
- `step`: positive leap second: 1, negative leap second: -1

Example:

``` plain
$ curl -s http://localhost:8080/?1489217288.328757 | jq .
{
  "id": "localhost:8080",
  "it": 1489217288.328757,
  "st": 1489224472.995564,
  "time": 1489224472.995564,
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
$ curl -I localhost:8080
HTTP/1.1 204 No Content
X-Httpstime: 1558915632.285965
Date: Mon, 27 May 2019 00:07:12 GMT
```

It is based on [Time over HTTPS specification](http://phk.freebsd.dk/time/20151129/).

## License

This software is released under the MIT License, see LICENSE.


## SEE ALSO

- [Time over HTTPS](http://phk.freebsd.dk/time/20151129/)
- [htp](http://www.vervest.org/htp/)
- [http/httpsを利用した時刻配信(アーカイブ)](https://jjy.nict.go.jp/QandA/reference/http-archive.html) (The time server using http/https (archived))
