package webntp

import (
	"math"
	"strconv"
	"time"
)

type Timestamp time.Time

var zeroEpochTime Timestamp

func init() {
	t, err := time.Parse(time.RFC3339Nano, "1970-01-01T00:00:00Z")
	if err != nil {
		panic(err)
	}
	zeroEpochTime = Timestamp(t)
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	// write seconds.
	tt := time.Time(t)
	b := make([]byte, 0, 20)
	b = strconv.AppendInt(b, tt.Unix(), 10)
	b = append(b, '.')

	// write milliseconds
	milli := (time.Duration(tt.Nanosecond()) + 500*time.Microsecond) / time.Millisecond
	switch {
	case milli < 10:
		b = append(b, '0', '0')
	case milli < 100:
		b = append(b, '0')
	}
	b = strconv.AppendInt(b, int64(milli), 10)
	return b, nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	intSec, nanoSec := int64(0), int64(0)
	nanoSecPos := int64(1e9)
	seenDot := false
	seenNumber := false
	seenSign := false
	sign := int64(1)
	for _, c := range b {
		switch c {
		case '.':
			seenDot = true
		case '-':
			if seenDot || seenNumber || seenSign {
				goto FALLBACK
			}
			sign = -1
			seenSign = true
		case '+':
			if seenDot || seenNumber || seenSign {
				goto FALLBACK
			}
			seenSign = true
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			seenNumber = true
			if seenDot {
				nanoSecPos /= 10
				nanoSec += nanoSecPos * int64(c-'0')
			} else {
				intSec = intSec*10 + int64(c-'0')
			}
		default:
			goto FALLBACK
		}
	}
	*t = Timestamp(time.Unix(sign*intSec, nanoSec))
	return nil

FALLBACK:
	timestamp, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	fintSec, fracSec := math.Modf(timestamp)
	*t = Timestamp(time.Unix(int64(fintSec), int64(fracSec*1e9)))
	return nil
}

type Response struct {
	ID           string    `json:"id"`
	InitiateTime Timestamp `json:"it"`
	SendTime     Timestamp `json:"st"`
	Leap         int       `json:"leap"`
	Next         Timestamp `json:"next"`
	Step         int       `json:"step"`
}
