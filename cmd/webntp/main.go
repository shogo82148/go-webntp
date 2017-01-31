package main

import (
	"fmt"

	"github.com/shogo82148/go-webntp"
)

func main() {
	c := &webntp.Client{}
	fmt.Println(c.Get("https://ntp-a1.nict.go.jp/cgi-bin/json"))
}
