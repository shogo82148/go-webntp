# WebNTP

WebNTP is NTP(-like) service via HTTP/WebSocket.

## Synopsis

First, `go get` and start the WebNTP Server.

``` plain
$ go get github.com/shogo82148/go-webntp/cmd/webntp
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

NICT(National Institute of Information and Communications Technology) provides WebNTP-compatible API.

``` plain
$ webntp https://ntp-a1.nict.go.jp/cgi-bin/json
server https://ntp-a1.nict.go.jp/cgi-bin/json, offset -0.006376, delay 0.024411
2017-03-11 16:08:06.150393313 +0900 JST, server https://ntp-a1.nict.go.jp/cgi-bin/json, offset -0.006376
```

## Shared Memory support for ntpd

Add a new server to your `ntpd.conf`.

``` plain
server 127.127.28.2 noselect
fudge 127.127.28.2 refid PYTH stratum 10
```

Run WebNTP with `-shm 2` option.

``` plain
$ webntp -p 1 -shm 2 https://ntp-a1.nict.go.jp/cgi-bin/json https://ntp-b1.nict.go.jp/cgi-bin/json
server https://ntp-a1.nict.go.jp/cgi-bin/json, offset -0.003258, delay 0.018910
server https://ntp-b1.nict.go.jp/cgi-bin/json, offset -0.003570, delay 0.021652
```

ntpd starts syncing with WebNTP.

``` plain
$ ntpq -p
     remote           refid      st t when poll reach   delay   offset  jitter
==============================================================================
 SHM(2)          .PYTH.          10 l    2   64   17    0.000   -3.331   0.384
*ntp-a2.nict.go. .NICT.           1 u   58   64   37   10.280    1.494   2.028
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

WebNTP returns UNIX timestamp by JSON.
See [the document of http/https service](http://www.nict.go.jp/JST/http.html) by NICT
(the content is written in japanese).

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

## License

This software is released under the MIT License, see LICENSE.


## See Also

- [htptime](http://www.htptime.org/index.html)
- [htp](http://www.vervest.org/htp/)
