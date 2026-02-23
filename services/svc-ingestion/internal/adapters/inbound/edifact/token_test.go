package edifact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenType_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		tokenTyp TokenType
		want     string
	}{
		{
			name:     "UNA type",
			tokenTyp: TokenUNA,
			want:     "UNA",
		},
		{
			name:     "SegmentTag type",
			tokenTyp: TokenSegmentTag,
			want:     "SegmentTag",
		},
		{
			name:     "DataElement type",
			tokenTyp: TokenDataElement,
			want:     "DataElement",
		},
		{
			name:     "Component type",
			tokenTyp: TokenComponent,
			want:     "Component",
		},
		{
			name:     "SegmentEnd type",
			tokenTyp: TokenSegmentEnd,
			want:     "SegmentEnd",
		},
		{
			name:     "EOF type",
			tokenTyp: TokenEOF,
			want:     "EOF",
		},
		{
			name:     "Error type",
			tokenTyp: TokenError,
			want:     "Error",
		},
		{
			name:     "unknown type falls back to numeric",
			tokenTyp: TokenType(99),
			want:     "Unknown(99)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.tokenTyp.String())
		})
	}
}

func TestToken_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		token Token
		want  string
	}{
		{
			name: "segment tag token",
			token: Token{
				Type:   TokenSegmentTag,
				Value:  []byte("UNB"),
				Line:   1,
				Column: 10,
			},
			want: `{Type: SegmentTag, Value: "UNB", Line: 1, Column: 10}`,
		},
		{
			name: "data element token",
			token: Token{
				Type:   TokenDataElement,
				Value:  []byte("UNOC"),
				Line:   1,
				Column: 14,
			},
			want: `{Type: DataElement, Value: "UNOC", Line: 1, Column: 14}`,
		},
		{
			name: "EOF token with empty value",
			token: Token{
				Type:   TokenEOF,
				Value:  nil,
				Line:   5,
				Column: 1,
			},
			want: `{Type: EOF, Value: "", Line: 5, Column: 1}`,
		},
		{
			name: "error token with message",
			token: Token{
				Type:   TokenError,
				Value:  []byte("unexpected character"),
				Line:   3,
				Column: 7,
			},
			want: `{Type: Error, Value: "unexpected character", Line: 3, Column: 7}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.token.String())
		})
	}
}

func TestTokenType_IotaValues(t *testing.T) {
	t.Parallel()

	// Verify iota ordering is stable — downstream code may serialize token types.
	require.Equal(t, TokenType(0), TokenUNA)
	require.Equal(t, TokenType(1), TokenSegmentTag)
	require.Equal(t, TokenType(2), TokenDataElement)
	require.Equal(t, TokenType(3), TokenComponent)
	require.Equal(t, TokenType(4), TokenSegmentEnd)
	require.Equal(t, TokenType(5), TokenEOF)
	require.Equal(t, TokenType(6), TokenError)
}
