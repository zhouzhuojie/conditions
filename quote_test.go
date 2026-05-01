package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", `"hello"`},
		{"hello\nworld", `"hello\nworld"`},
		{`hello "world"`, `"hello \"world\""`},
		{`back\slash`, `"back\\slash"`},
		{"", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Quote(tt.input))
		})
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo.bar", "foo.bar"},
		{"foo_bar", "foo_bar"},
		{"foo-bar", `"foo-bar"`},
		{"foo bar", `"foo bar"`},
		{"123", `"123"`},
		{"", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, QuoteIdent(tt.input))
		})
	}
}
