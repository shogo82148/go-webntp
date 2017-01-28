package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptrace"
	"time"
)

type Response struct {
	ID   string  `json:"id"`
	It   float64 `json:"it"`
	St   float64 `json:"st"`
	Leap int     `json:"leap"`
	Next float64 `json:"next"`
	Step int     `json:"step"`
}

func main() {
	url := "https://ntp-a1.nict.go.jp/cgi-bin/json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	var start, end time.Time
	trace := &httptrace.ClientTrace{
		WroteRequest:         func(info httptrace.WroteRequestInfo) { start = time.Now() },
		GotFirstResponseByte: func() { end = time.Now() },
	}
	ctx := httptrace.WithClientTrace(req.Context(), trace)
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var result Response
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&result)

	intsec, fracsec := math.Modf(result.St)
	ntpTime := time.Unix(int64(intsec), int64(fracsec*1000000000))
	delay := end.Sub(start)
	myTime := start.Add(delay / 2)
	offset := myTime.Sub(ntpTime)

	fmt.Println("offset: ", offset)
	fmt.Println("delay: ", delay)
	fmt.Println("time: ", time.Now().Add(offset))
}
