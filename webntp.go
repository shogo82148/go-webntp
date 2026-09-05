package webntp

import (
	"bytes"
	"encoding/json/jsontext"
	"strconv"
	"strings"
	"time"
)

// Timestamp is posix unix timestamp.
type Timestamp time.Time

func ParseTimestamp(s string) (Timestamp, error) {
	i, d, ok := strings.Cut(s, ".")
	if !ok {
		sec, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return Timestamp{}, err
		}
		return Timestamp(time.Unix(sec, 0)), nil
	}

	sec, err := strconv.ParseInt(i, 10, 64)
	if err != nil {
		return Timestamp{}, err
	}

	nsec := int64(0)
	digit := int64(1e8)
	j := 0
	for ; j < len(d) && j < 9; j++ {
		if d[j] < '0' || d[j] > '9' {
			return Timestamp{}, strconv.ErrSyntax
		}
		nsec += int64(d[j]-'0') * digit
		digit /= 10
	}
	for ; j < len(d); j++ {
		if d[j] < '0' || d[j] > '9' {
			return Timestamp{}, strconv.ErrSyntax
		}
	}
	return Timestamp(time.Unix(sec, nsec)), nil
}

func (t Timestamp) MarshalJSONTo(enc *jsontext.Encoder) error {
	tt := time.Time(t)
	buf := make([]byte, 0, 32)
	sec := tt.Unix()
	nsec := tt.Nanosecond()
	buf = strconv.AppendInt(buf, sec, 10)
	if nsec != 0 {
		buf = append(buf, '.')
		buf = append(buf, byte(nsec/1e8+'0'))
		nsec %= 1e8
		buf = append(buf, byte(nsec/1e7+'0'))
		nsec %= 1e7
		buf = append(buf, byte(nsec/1e6+'0'))
		nsec %= 1e6
		buf = append(buf, byte(nsec/1e5+'0'))
		nsec %= 1e5
		buf = append(buf, byte(nsec/1e4+'0'))
		nsec %= 1e4
		buf = append(buf, byte(nsec/1e3+'0'))
		nsec %= 1e3
		buf = append(buf, byte(nsec/1e2+'0'))
		nsec %= 1e2
		buf = append(buf, byte(nsec/10+'0'))
		nsec %= 10
		buf = append(buf, byte(nsec+'0'))
		buf = bytes.TrimRight(buf, "0")
	}
	return enc.WriteValue(jsontext.Value(buf))
}

var zeroEpochTime = Timestamp(time.Unix(0, 0))

// Response is a response from webntp server.
type Response struct {
	// ID is the id of the web-ntp server.
	ID string `json:"id"`

	// InitiateTime is the time that the request has been sent.
	InitiateTime Timestamp `json:"it"`

	// SendTime is the time that the response has been sent.
	SendTime Timestamp `json:"st"`

	// Time is a same as SendTime.
	// It is for htptime compatibility.
	Time Timestamp `json:"time"`

	// Leap is from TAI to UTC. (**before** Response.Next)
	Leap int `json:"leap"`

	// Next is the time of next or last leap second event.
	Next Timestamp `json:"next"`

	// Step describes next or last leap second is insertion or deletion.
	// +1 is insertion, -1 is deletion.
	Step int `json:"step"`
}
