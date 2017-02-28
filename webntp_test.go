package webntp

import (
	"bytes"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestTimestamp_MarshalJSON(t *testing.T) {
	testCases := []struct {
		t   string
		str string
	}{
		{"1970-01-01T00:00:00Z", "0.000"},
		{"1970-01-01T00:00:00.0004Z", "0.000"},
		{"1970-01-01T00:00:00.0005Z", "0.001"},
		{"1970-01-01T00:00:00.001Z", "0.001"},
		{"1970-01-01T00:00:00.010Z", "0.010"},
		{"1970-01-01T00:00:00.100Z", "0.100"},
		{"2009-02-14T08:31:30+09:00", "1234567890.000"},
	}

	for _, tc := range testCases {
		tt, err := time.Parse(time.RFC3339Nano, tc.t)
		if err != nil {
			t.Error(err)
		}
		b, err := Timestamp(tt).MarshalJSON()
		if err != nil {
			t.Error(err)
		}
		if string(b) != tc.str {
			t.Errorf("want %s, got %s (%s)", tc.str, string(b), tc.t)
		}
	}
}

func BenchmarkTimestamp_MarshalJSON(b *testing.B) {
	t, _ := time.Parse(time.RFC3339Nano, "2009-02-14T08:31:30+09:00")
	timestamp := Timestamp(t)
	for i := 0; i < b.N; i++ {
		timestamp.MarshalJSON()
	}
}

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		t   string
		str string
	}{
		{"1970-01-01T00:00:00Z", "0.000"},
		{"1970-01-01T00:00:00.000Z", "0.000"},
		{"1970-01-01T00:00:00.001Z", "0.001"},
		{"1970-01-01T00:00:00.010Z", "0.010"},
		{"1970-01-01T00:00:00.100Z", "0.100"},
		{"2009-02-14T08:31:30+09:00", "1234567890.000"},
		{"2009-02-14T08:31:30+09:00", "1.234567890000e9"},
	}

	for _, tc := range testCases {
		tt, err := time.Parse(time.RFC3339Nano, tc.t)
		if err != nil {
			t.Error(err)
		}
		var timestamp Timestamp
		err = timestamp.UnmarshalJSON([]byte(tc.str))
		if err != nil {
			t.Error(err)
		}
		if !tt.UTC().Equal(time.Time(timestamp).UTC()) {
			t.Errorf("want %s, got %s (%s)", tt, time.Time(timestamp), tc.str)
		}
	}
}

func TestParseLeapSecondsList(t *testing.T) {
	r := bytes.NewBuffer([]byte(`
# leap-seconds.list for test

# Last Update of leap second values:   8 July 2016
#$	 3676924800

# File expires on:  28 June 2017
#@	3707596800

# This line is not a leap second.
# It is the definition of the relationship between UTC and TAI.
2272060800	10	# 1 Jan 1972

# A leap second.
2287785600	11	# 1 Jul 1972

# It's test for negative leap second.
# In fact, the leap second on 1 Jan 1973 is positive
2303683200	10	# 1 Jan 1973
`))
	l, err := ParseLeapSecondsList(r)

	if expected, _ := time.Parse(time.RFC3339, "2016-07-08T00:00:00Z"); !l.UpdateAt.Equal(expected) {
		t.Errorf("want update_at is %s, got %s", expected, l.UpdateAt)
	}
	if expected, _ := time.Parse(time.RFC3339, "2017-06-28T00:00:00Z"); !l.ExpireAt.Equal(expected) {
		t.Errorf("want expire_at is %s, got %s", expected, l.ExpireAt)
	}

	if len(l.LeapSeconds) != 2 {
		t.Errorf("want the length of leap second list is 2, got %d", len(l.LeapSeconds))
	}

	if expected, _ := time.Parse(time.RFC3339, "1972-07-01T00:00:00Z"); !l.LeapSeconds[0].At.Equal(expected) {
		t.Errorf("want list[0].At is %s, got %s", expected, l.LeapSeconds[0].At)
	}
	if l.LeapSeconds[0].Leap != 10 {
		t.Errorf("want list[0].Leap is 10, got %d", l.LeapSeconds[0].Leap)
	}
	if l.LeapSeconds[0].Step != 1 {
		t.Errorf("want list[0].Step is 1, got %d", l.LeapSeconds[0].Step)
	}

	if expected, _ := time.Parse(time.RFC3339, "1973-01-01T00:00:00Z"); !l.LeapSeconds[1].At.Equal(expected) {
		t.Errorf("want list[1].At is %s, got %s", expected, l.LeapSeconds[1].At)
	}
	if l.LeapSeconds[1].Leap != 11 {
		t.Errorf("want list[1].Leap is 11, got %d", l.LeapSeconds[1].Leap)
	}
	if l.LeapSeconds[1].Step != -1 {
		t.Errorf("want list[1].Step is -1, got %d", l.LeapSeconds[1].Step)
	}

	if err != nil {
		t.Error(err)
	}
}

func BenchmarkGet(b *testing.B) {
	s := &Server{
		LeapSecondsPath: "leap-seconds.list",
		LeapSecondsURL:  "https://www.ietf.org/timezones/data/leap-seconds.list",
	}
	ts := httptest.NewServer(s)
	defer ts.Close()

	c := &Client{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ts.URL)
	}
}

func BenchmarkGetWS(b *testing.B) {
	s := &Server{
		LeapSecondsPath: "leap-seconds.list",
		LeapSecondsURL:  "https://www.ietf.org/timezones/data/leap-seconds.list",
	}
	ts := httptest.NewServer(s)
	defer ts.Close()
	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"
	wsURL := u.String()

	c := &Client{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(wsURL)
	}
}

func BenchmarkTimestamp_UnmarshalJSON(b *testing.B) {
	bs := []byte("1234567890.000")
	for i := 0; i < b.N; i++ {
		var timestamp Timestamp
		timestamp.UnmarshalJSON(bs)
	}
}

func BenchmarkTimestamp_UnmarshalJSONexp(b *testing.B) {
	bs := []byte("1.234567890000e9")
	for i := 0; i < b.N; i++ {
		var timestamp Timestamp
		timestamp.UnmarshalJSON(bs)
	}
}
