package webntp

import (
	"math"
	"time"
)

type Response struct {
	ID           string  `json:"id"`
	InitiateTime float64 `json:"it"`
	SendTime     float64 `json:"st"`
	Leap         int     `json:"leap"`
	Next         float64 `json:"next"`
	Step         int     `json:"step"`
}

func timestampToTime(timestamp float64) time.Time {
	intsec, fracsec := math.Modf(timestamp)
	return time.Unix(int64(intsec), int64(fracsec*1e9))
}

func timeToTimestamp(t time.Time) float64 {
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}
