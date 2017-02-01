package webntp

import (
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
