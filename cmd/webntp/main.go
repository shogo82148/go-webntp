package main

import (
	"net/http"

	"github.com/shogo82148/go-webntp"
)

func main() {
	s := webntp.NewServer()
	http.ListenAndServe(":8080", s)
}
