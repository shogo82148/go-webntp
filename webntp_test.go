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
		{"1234567890", Timestamp(time.Unix(1234567890, 0))},
		{"1234567890.123456789", Timestamp(time.Unix(1234567890, 123456789))},
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
