package webntp

import (
	"bytes"
	"encoding/json/jsontext"
	"strings"
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected Timestamp
	}{
		{"0", Timestamp(time.Unix(0, 0))},
		{"1", Timestamp(time.Unix(1, 0))},
		{"-1", Timestamp(time.Unix(-1, 0))},
		{"-1.1", Timestamp(time.Unix(-1, -1e8))},
		{"1234567890", Timestamp(time.Unix(1234567890, 0))},
		{"1234567890.123456789", Timestamp(time.Unix(1234567890, 123456789))},
		{"1e9", Timestamp(time.Unix(1e9, 0))},
		{"1E+9", Timestamp(time.Unix(1e9, 0))},
		{"1.23456789e3", Timestamp(time.Unix(1234, 567890000))},
		{"1234567890123456789e-9", Timestamp(time.Unix(1234567890, 123456789))},
		{"1.234567890123456789e+9", Timestamp(time.Unix(1234567890, 123456789))},
		{"-11e-1", Timestamp(time.Unix(-1, -1e8))},
		{"1e-10", Timestamp(time.Unix(0, 0))},
		{"0e2147483647", Timestamp(time.Unix(0, 0))},
		{"-0e2147483647", Timestamp(time.Unix(0, 0))},
		{"9223372036854775807", Timestamp(time.Unix(1<<63-1, 0))},
		{"-9223372036854775808", Timestamp(time.Unix(-1<<63, 0))},
		{"922337203685477580.7e1", Timestamp(time.Unix(1<<63-1, 0))},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseTimestamp(test.input)
			if err != nil {
				t.Fatalf("ParseTimestamp(%q) returned error: %v", test.input, err)
			}
			if got != test.expected {
				t.Errorf("ParseTimestamp(%q) = %v; want %v", test.input, got, test.expected)
			}
		})
	}
}

func TestParseTimestampInvalid(t *testing.T) {
	invalidInputs := []string{
		"abc",
		"abc.0",
		"123.abc",
		"123.1234567890abc",
		"1e",
		"1e2e3",
		"1.2.3e4",
		"9223372036854775808",
		"-9223372036854775809",
		"1e19",
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ParseTimestamp(input)
			if err == nil {
				t.Errorf("ParseTimestamp(%q) did not return an error; expected an error", input)
			}
		})
	}
}

func TestTimestampMarshalJSONTo(t *testing.T) {
	tests := []struct {
		input    Timestamp
		expected string
	}{
		{Timestamp(time.Unix(0, 0)), "0"},
		{Timestamp(time.Unix(1, 0)), "1"},
		{Timestamp(time.Unix(-1, 0)), "-1"},
		{Timestamp(time.Unix(-1, -1e8)), "-1.1"},
		{Timestamp(time.Unix(1234567890, 0)), "1234567890"},
		{Timestamp(time.Unix(1234567890, 123456789)), "1234567890.123456789"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			var buf bytes.Buffer
			enc := jsontext.NewEncoder(&buf)
			err := test.input.MarshalJSONTo(enc)
			if err != nil {
				t.Fatalf("MarshalJSONTo returned error: %v", err)
			}
			got := strings.TrimSpace(buf.String())
			if got != test.expected {
				t.Errorf("MarshalJSONTo = %q; want %q", got, test.expected)
			}
		})
	}
}

func FuzzParseTimestamp(f *testing.F) {
	f.Add("0")
	f.Add("1")
	f.Add("-1")
	f.Add("-1.1")
	f.Add("1234567890")
	f.Add("1234567890.123456789")
	f.Add("1e9")
	f.Add("1E+9")
	f.Add("1.23456789e3")
	f.Add("1234567890123456789e-9")
	f.Add("1.234567890123456789e+9")
	f.Add("-11e-1")
	f.Add("1e-10")
	f.Add("0e2147483647")
	f.Add("-0e2147483647")
	f.Add("9223372036854775807")
	f.Add("-9223372036854775808")
	f.Add("922337203685477580.7e1")
	f.Add("abc")
	f.Add("abc.0")
	f.Add("123.abc")
	f.Add("123.1234567890abc")
	f.Add("1e")
	f.Add("1e2e3")
	f.Add("1.2.3e4")
	f.Add("9223372036854775808")
	f.Add("-9223372036854775809")
	f.Add("1e19")

	f.Fuzz(func(t *testing.T, input string) {
		got, err := ParseTimestamp(input)
		if err != nil {
			return
		}
		roundtrip, err := ParseTimestamp(got.String())
		if err != nil {
			t.Fatalf("ParseTimestamp(%q) returned error on roundtrip: %v", got.String(), err)
		}
		if got != roundtrip {
			t.Errorf("roundtrip mismatch: got %v; want %v", time.Time(roundtrip), time.Time(got))
		}
	})
}

func BenchmarkParseTimestamp(b *testing.B) {
	for b.Loop() {
		_, err := ParseTimestamp("1234567890.123456789")
		if err != nil {
			b.Fatalf("ParseTimestamp returned error: %v", err)
		}
	}
}

func BenchmarkParseTimestamp_exp(b *testing.B) {
	for b.Loop() {
		_, err := ParseTimestamp("1.23456789e3")
		if err != nil {
			b.Fatalf("ParseTimestamp returned error: %v", err)
		}
	}
}
