package webntp

import (
	"bufio"
	"io"
	"math"
	"sort"
	"strconv"
	"time"
	"unicode"
)

// Timestamp is posix unix timestamp.
type Timestamp time.Time

var zeroEpochTime Timestamp

func init() {
	t, err := time.Parse(time.RFC3339Nano, "1970-01-01T00:00:00Z")
	if err != nil {
		panic(err)
	}
	zeroEpochTime = Timestamp(t)
}

// MarshalJSON converts the timestamp to JSON number.
// The number is unix timestamp.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	// write seconds.
	tt := time.Time(t)
	b := make([]byte, 0, 20)
	b = strconv.AppendInt(b, tt.Unix(), 10)
	b = append(b, '.')

	// write microsecond
	m := (time.Duration(tt.Nanosecond()) + 500*time.Nanosecond) / time.Microsecond
	switch {
	case m < 10:
		b = append(b, '0', '0', '0', '0', '0')
	case m < 100:
		b = append(b, '0', '0', '0', '0')
	case m < 1000:
		b = append(b, '0', '0', '0')
	case m < 10000:
		b = append(b, '0', '0')
	case m < 100000:
		b = append(b, '0')
	}
	b = strconv.AppendInt(b, int64(m), 10)
	return b, nil
}

// UnmarshalJSON converts JSON number to a timestamp.
// The number is unix timestamp.
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

// Response is a response from webntp server.
type Response struct {
	ID           string    `json:"id"`
	InitiateTime Timestamp `json:"it"`
	SendTime     Timestamp `json:"st"`
	Leap         int       `json:"leap"`
	Next         Timestamp `json:"next"`
	Step         int       `json:"step"`
}

// LeapSecond is information for leap-seconds
type LeapSecond struct {
	// At is the time to insert/delete a leap second.
	At time.Time

	// Leap is offset from TAI to UTC. (before LeapSecond.Time)
	Leap int

	// Step describes next leap second is insertion or deletion.
	// +1 is insertion, -1 is deletion.
	Step int
}

// LeapSecondsList is the contents of leap-second.list.
type LeapSecondsList struct {
	LeapSeconds []LeapSecond
	UpdateAt    time.Time
	ExpireAt    time.Time
}

type leapSecondsParser struct {
	r        *bufio.Reader
	list     []LeapSecond
	updateAt time.Time
	expireAt time.Time
	err      error
}

const ntpEpochOffset = (70*365 + 17) * 86400

// ParseLeapSecondsList parses leap-second.list.
func ParseLeapSecondsList(r io.Reader) (*LeapSecondsList, error) {
	p := &leapSecondsParser{
		r: bufio.NewReaderSize(r, 1024),
	}
	var err error

LOOP:
	for {
		var r rune
		r, _, err = p.r.ReadRune()
		if err != nil {
			break
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case r == '#':
			// comment line
			p.parseComment()
		case '0' <= r && r <= '9':
			// leap second line
			err := p.r.UnreadRune()
			if err != nil {
				p.err = err
				break LOOP
			}
			p.parseLeapSecond()
		default:
			// unknown line
			p.skipLine()
		}
		if p.err != nil {
			return nil, p.err
		}
	}

	sort.Slice(p.list, func(i, j int) bool {
		return p.list[i].At.Before(p.list[j].At)
	})
	lastLeap := p.list[0].Leap
	for i := 1; i < len(p.list); i++ {
		p.list[i].Step = p.list[i].Leap - lastLeap
		lastLeap, p.list[i].Leap = p.list[i].Leap, lastLeap
	}

	return &LeapSecondsList{
		LeapSeconds: p.list[1:],
		UpdateAt:    p.updateAt,
		ExpireAt:    p.expireAt,
	}, nil
}

func (p *leapSecondsParser) parseComment() {
	var r rune
	r, _, p.err = p.r.ReadRune()
	if p.err != nil {
		return
	}
	switch r {
	case '\n':
		return
	case ' ':
		// normal comment line
	case '$':
		// last update date
		i := p.getInt(64)
		p.updateAt = time.Unix(i-ntpEpochOffset, 0)
	case '@':
		// the expiration date
		i := p.getInt(64)
		p.expireAt = time.Unix(i-ntpEpochOffset, 0)
	case 'h':
		// hash value of the data
	default:
		// unknown comment line. ignore it.
	}
	p.skipLine()
}

func (p *leapSecondsParser) parseLeapSecond() {
	at := p.getInt(64)
	p.skipSpace()
	leap := p.getInt(0)
	p.skipSpace()
	p.skipLine()
	if p.err != nil {
		return
	}
	p.list = append(p.list, LeapSecond{
		At:   time.Unix(at-ntpEpochOffset, 0),
		Leap: int(leap),
	})
}

func (p *leapSecondsParser) skipSpace() {
	if p.err != nil {
		return
	}
	for {
		r, _, err := p.r.ReadRune()
		if err != nil {
			p.err = err
			return
		}
		if !unicode.IsSpace(r) {
			p.err = p.r.UnreadRune()
			return
		}
	}
}

func (p *leapSecondsParser) skipLine() {
	if p.err != nil {
		return
	}
	for {
		r, _, err := p.r.ReadRune()
		if err != nil {
			p.err = err
			return
		}
		if r == '\n' {
			break
		}
	}
	return
}

func (p *leapSecondsParser) getInt(size int) int64 {
	p.skipSpace()
	if p.err != nil {
		return 0
	}
	buf := make([]byte, 0, 20)
	var err error
	var r rune
	for {
		r, _, err = p.r.ReadRune()
		if err != nil {
			break
		}
		if r < '0' || r > '9' {
			err = p.r.UnreadRune()
			break
		}
		buf = append(buf, byte(r))
	}
	if len(buf) == 0 && err != nil {
		p.err = err
		return 0
	}

	i, err := strconv.ParseInt(string(buf), 10, size)
	p.err = err
	return i
}
