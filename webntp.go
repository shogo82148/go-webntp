package webntp

import (
	"bytes"
	"encoding/json/jsontext"
	"strconv"
	"strings"
	"time"
)

// Subprotocol is a subprotocol name for websocket.
const Subprotocol = "webntp.shogo82148.com"

// Timestamp is posix unix timestamp.
type Timestamp time.Time

func ParseTimestamp(s string) (Timestamp, error) {
	mantissa, exponentString, hasExponent := strings.Cut(s, "e")
	if !hasExponent {
		mantissa, exponentString, hasExponent = strings.Cut(s, "E")
	}

	exponent := int64(0)
	if hasExponent {
		var err error
		exponent, err = strconv.ParseInt(exponentString, 10, 32)
		if err != nil {
			return Timestamp{}, err
		}
	}

	integer, fraction, hasFraction := strings.Cut(mantissa, ".")
	if integer == "" || (hasFraction && fraction == "") {
		return Timestamp{}, strconv.ErrSyntax
	}
	negative := integer[0] == '-'
	start := 0
	if integer[0] == '+' || integer[0] == '-' {
		start++
	}
	if start == len(integer) {
		return Timestamp{}, strconv.ErrSyntax
	}
	for i := start; i < len(integer); i++ {
		if integer[i] < '0' || integer[i] > '9' {
			return Timestamp{}, strconv.ErrSyntax
		}
	}
	for i := range fraction {
		if fraction[i] < '0' || fraction[i] > '9' {
			return Timestamp{}, strconv.ErrSyntax
		}
	}

	digits := integer[start:] + fraction
	decimalPos := int64(len(integer)-start) + exponent
	secEnd := max(decimalPos, 0)
	secDigits := min(secEnd, int64(len(digits)))

	limit := uint64(1<<63 - 1)
	if negative {
		limit++
	}
	sec := uint64(0)
	for i := range secDigits {
		digit := uint64(digits[i] - '0')
		if sec > (limit-digit)/10 {
			return Timestamp{}, strconv.ErrRange
		}
		sec = sec*10 + digit
	}
	for i := secDigits; i < secEnd; i++ {
		if sec > limit/10 {
			return Timestamp{}, strconv.ErrRange
		}
		sec *= 10
	}

	nsec := int64(0)
	for i := range int64(9) {
		nsec *= 10
		index := decimalPos + i
		if index >= 0 && index < int64(len(digits)) {
			nsec += int64(digits[index] - '0')
		}
	}
	seconds := int64(sec)
	if negative {
		seconds = -seconds
		nsec = -nsec
	}
	return Timestamp(time.Unix(seconds, nsec)), nil
}

func (t Timestamp) String() string {
	return string(t.encode())
}

func (t Timestamp) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(t.encode()))
}

func (t Timestamp) encode() []byte {
	tt := time.Time(t)
	buf := make([]byte, 0, 32)
	sec := tt.Unix()
	nsec := tt.Nanosecond()
	if sec < 0 && nsec != 0 {
		sec++
		nsec = 1e9 - nsec
	}
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
	return buf
}

func (t *Timestamp) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	ts, err := ParseTimestamp(string(v))
	if err != nil {
		return err
	}
	*t = ts
	return nil
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
