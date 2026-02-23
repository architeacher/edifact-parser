package edifact

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func collectTokens(lexer *Lexer) []Token {
	var tokens []Token

	for {
		tok := lexer.NextToken()
		tokens = append(tokens, tok)

		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}

	return tokens
}

func TestNewLexer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		data       []byte
		delimiters *Delimiters
	}{
		{
			name:       "creates lexer with default delimiters",
			data:       []byte("UNB+SENDER'"),
			delimiters: DefaultDelimiters(),
		},
		{
			name:       "creates lexer with empty input",
			data:       []byte{},
			delimiters: DefaultDelimiters(),
		},
		{
			name:       "creates lexer with nil data",
			data:       nil,
			delimiters: DefaultDelimiters(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer(tc.data, tc.delimiters)

			require.NotNil(t, lexer)
			require.Equal(t, tc.delimiters, lexer.delimiters)
			require.Equal(t, 1, lexer.line)
			require.Equal(t, 1, lexer.column)
		})
	}
}

func TestLexer_NextToken_SimpleSegment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantTokens []Token
	}{
		{
			name:  "single segment with two data elements",
			input: "UNB+SENDER+RECEIVER'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("SENDER")},
				{Type: TokenDataElement, Value: []byte("RECEIVER")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "segment tag only with terminator",
			input: "UNS'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNS")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "empty data elements",
			input: "TST++'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("")},
				{Type: TokenDataElement, Value: []byte("")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "empty input returns EOF",
			input: "",
			wantTokens: []Token{
				{Type: TokenEOF},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer([]byte(tc.input), DefaultDelimiters())
			tokens := collectTokens(lexer)

			require.Len(t, tokens, len(tc.wantTokens))

			for index, want := range tc.wantTokens {
				require.Equal(t, want.Type, tokens[index].Type, "token %d type", index)
				require.Equal(t, string(want.Value), string(tokens[index].Value), "token %d value", index)
			}
		})
	}
}

func TestLexer_NextToken_ComponentSeparator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantTokens []Token
	}{
		{
			name:  "component separator within data element",
			input: "UNB+A:B'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("A")},
				{Type: TokenComponent, Value: []byte("B")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "multiple components in one data element",
			input: "UNB+A:B:C:D'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("A")},
				{Type: TokenComponent, Value: []byte("B")},
				{Type: TokenComponent, Value: []byte("C")},
				{Type: TokenComponent, Value: []byte("D")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "empty components",
			input: "TST+A::C'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("A")},
				{Type: TokenComponent, Value: []byte("")},
				{Type: TokenComponent, Value: []byte("C")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "components across multiple data elements",
			input: "UNB+A:B+C:D'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("A")},
				{Type: TokenComponent, Value: []byte("B")},
				{Type: TokenDataElement, Value: []byte("C")},
				{Type: TokenComponent, Value: []byte("D")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer([]byte(tc.input), DefaultDelimiters())
			tokens := collectTokens(lexer)

			require.Len(t, tokens, len(tc.wantTokens))

			for index, want := range tc.wantTokens {
				require.Equal(t, want.Type, tokens[index].Type, "token %d type", index)
				require.Equal(t, string(want.Value), string(tokens[index].Value), "token %d value", index)
			}
		})
	}
}

func TestLexer_NextToken_ReleaseCharacter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantTokens []Token
	}{
		{
			name:  "escaped data element separator",
			input: "TST+TEXT?+MORE'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("TEXT+MORE")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "escaped component separator",
			input: "TST+A?:B'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("A:B")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "escaped segment terminator",
			input: "TST+A?'B'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("A'B")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "escaped release character itself",
			input: "TST+A??B'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("A?B")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "multiple escape sequences",
			input: "TST+A?+B?:C'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("TST")},
				{Type: TokenDataElement, Value: []byte("A+B:C")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer([]byte(tc.input), DefaultDelimiters())
			tokens := collectTokens(lexer)

			require.Len(t, tokens, len(tc.wantTokens))

			for index, want := range tc.wantTokens {
				require.Equal(t, want.Type, tokens[index].Type, "token %d type", index)
				require.Equal(t, string(want.Value), string(tokens[index].Value), "token %d value", index)
			}
		})
	}
}

func TestLexer_NextToken_MultiSegment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantTokens []Token
	}{
		{
			name:  "two segments back to back",
			input: "UNB+S+R'UNH+H1'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("S")},
				{Type: TokenDataElement, Value: []byte("R")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenSegmentTag, Value: []byte("UNH")},
				{Type: TokenDataElement, Value: []byte("H1")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "segments with whitespace between them",
			input: "UNB+S'  \n  UNH+H1'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("S")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenSegmentTag, Value: []byte("UNH")},
				{Type: TokenDataElement, Value: []byte("H1")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "three segments",
			input: "UNB+X'UNH+Y'UNT+1'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("X")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenSegmentTag, Value: []byte("UNH")},
				{Type: TokenDataElement, Value: []byte("Y")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenSegmentTag, Value: []byte("UNT")},
				{Type: TokenDataElement, Value: []byte("1")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer([]byte(tc.input), DefaultDelimiters())
			tokens := collectTokens(lexer)

			require.Len(t, tokens, len(tc.wantTokens))

			for index, want := range tc.wantTokens {
				require.Equal(t, want.Type, tokens[index].Type, "token %d type", index)
				require.Equal(t, string(want.Value), string(tokens[index].Value), "token %d value", index)
			}
		})
	}
}

func TestLexer_NextToken_PositionTracking(t *testing.T) {
	t.Parallel()

	input := "UNB+S'\nUNH+H'"
	lexer := NewLexer([]byte(input), DefaultDelimiters())
	tokens := collectTokens(lexer)

	// UNB starts at line 1, column 1
	require.Equal(t, 1, tokens[0].Line)
	require.Equal(t, 1, tokens[0].Column)

	// +S data element at line 1
	require.Equal(t, 1, tokens[1].Line)

	// Segment end at line 1
	require.Equal(t, 1, tokens[2].Line)

	// UNH starts at line 2
	require.Equal(t, 2, tokens[3].Line)
	require.Equal(t, 1, tokens[3].Column)
}

func TestLexer_NextToken_RealisticEDIFACT(t *testing.T) {
	t.Parallel()

	input := "UNB+UNOC:3+1234567890123:14+9876543210987:14+210101:1200+00000001'"
	lexer := NewLexer([]byte(input), DefaultDelimiters())
	tokens := collectTokens(lexer)

	require.Equal(t, TokenSegmentTag, tokens[0].Type)
	require.Equal(t, "UNB", string(tokens[0].Value))

	// UNOC:3 - composite with component
	require.Equal(t, TokenDataElement, tokens[1].Type)
	require.Equal(t, "UNOC", string(tokens[1].Value))
	require.Equal(t, TokenComponent, tokens[2].Type)
	require.Equal(t, "3", string(tokens[2].Value))

	// 1234567890123:14
	require.Equal(t, TokenDataElement, tokens[3].Type)
	require.Equal(t, "1234567890123", string(tokens[3].Value))
	require.Equal(t, TokenComponent, tokens[4].Type)
	require.Equal(t, "14", string(tokens[4].Value))

	// 9876543210987:14
	require.Equal(t, TokenDataElement, tokens[5].Type)
	require.Equal(t, "9876543210987", string(tokens[5].Value))
	require.Equal(t, TokenComponent, tokens[6].Type)
	require.Equal(t, "14", string(tokens[6].Value))

	// 210101:1200
	require.Equal(t, TokenDataElement, tokens[7].Type)
	require.Equal(t, "210101", string(tokens[7].Value))
	require.Equal(t, TokenComponent, tokens[8].Type)
	require.Equal(t, "1200", string(tokens[8].Value))

	// 00000001
	require.Equal(t, TokenDataElement, tokens[9].Type)
	require.Equal(t, "00000001", string(tokens[9].Value))

	require.Equal(t, TokenSegmentEnd, tokens[10].Type)
	require.Equal(t, TokenEOF, tokens[11].Type)
}

func TestLexer_NextToken_LargeInput(t *testing.T) {
	t.Parallel()

	// Build a 64+ KB input to test buffering across buffer boundaries.
	var buf bytes.Buffer

	segmentCount := 0

	for buf.Len() < 65536 {
		buf.WriteString("TST+DATA+VALUE'")
		segmentCount++
	}

	lexer := NewLexer(buf.Bytes(), DefaultDelimiters())
	tokens := collectTokens(lexer)

	// Each segment produces: SegmentTag + DataElement + DataElement + SegmentEnd = 4 tokens, plus final EOF.
	expectedTokens := segmentCount*4 + 1
	require.Len(t, tokens, expectedTokens)
	require.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)
}

func TestLexer_NextToken_WhitespaceHandling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantTokens []Token
	}{
		{
			name:  "leading whitespace before first segment",
			input: "  \t\n UNB+S'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("S")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "trailing whitespace after last segment",
			input: "UNB+S'  \n  ",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("UNB")},
				{Type: TokenDataElement, Value: []byte("S")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
		{
			name:  "whitespace within data element is preserved",
			input: "FTX+HELLO WORLD'",
			wantTokens: []Token{
				{Type: TokenSegmentTag, Value: []byte("FTX")},
				{Type: TokenDataElement, Value: []byte("HELLO WORLD")},
				{Type: TokenSegmentEnd, Value: []byte("'")},
				{Type: TokenEOF},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer([]byte(tc.input), DefaultDelimiters())
			tokens := collectTokens(lexer)

			require.Len(t, tokens, len(tc.wantTokens))

			for index, want := range tc.wantTokens {
				require.Equal(t, want.Type, tokens[index].Type, "token %d type", index)
				require.Equal(t, string(want.Value), string(tokens[index].Value), "token %d value", index)
			}
		})
	}
}

func TestLexer_NextToken_CustomDelimiters(t *testing.T) {
	t.Parallel()

	delimiters := &Delimiters{
		ComponentSeparator:   ';',
		DataElementSeparator: '-',
		DecimalNotation:      '.',
		ReleaseCharacter:     '!',
		RepetitionSeparator:  '*',
		SegmentTerminator:    '~',
	}

	input := "UNB-A;B-C~"
	lexer := NewLexer([]byte(input), delimiters)
	tokens := collectTokens(lexer)

	expected := []Token{
		{Type: TokenSegmentTag, Value: []byte("UNB")},
		{Type: TokenDataElement, Value: []byte("A")},
		{Type: TokenComponent, Value: []byte("B")},
		{Type: TokenDataElement, Value: []byte("C")},
		{Type: TokenSegmentEnd, Value: []byte("~")},
		{Type: TokenEOF},
	}

	require.Len(t, tokens, len(expected))

	for index, want := range expected {
		require.Equal(t, want.Type, tokens[index].Type, "token %d type", index)
		require.Equal(t, string(want.Value), string(tokens[index].Value), "token %d value", index)
	}
}

func TestLexer_NextToken_ErrorCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "release character at end of input",
			input: "TST+A?",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewLexer([]byte(tc.input), DefaultDelimiters())
			tokens := collectTokens(lexer)

			hasError := false
			for _, tok := range tokens {
				if tok.Type == TokenError {
					hasError = true

					break
				}
			}

			require.True(t, hasError, "expected an error token")
		})
	}
}

func TestLexer_NextToken_RepeatedEOF(t *testing.T) {
	t.Parallel()

	lexer := NewLexer([]byte("UNB'"), DefaultDelimiters())
	tokens := collectTokens(lexer)

	require.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)

	// Calling NextToken again should keep returning EOF.
	extra := lexer.NextToken()
	require.Equal(t, TokenEOF, extra.Type)

	extra = lexer.NextToken()
	require.Equal(t, TokenEOF, extra.Type)
}

func TestLexer_NextToken_NumericSegmentTag(t *testing.T) {
	t.Parallel()

	// Segment tags can contain digits (e.g., LOC, PIA, SG1 in some contexts).
	input := "SG1+DATA'S12+V'TAX+X'"
	lexer := NewLexer([]byte(input), DefaultDelimiters())
	tokens := collectTokens(lexer)

	require.Equal(t, TokenSegmentTag, tokens[0].Type)
	require.Equal(t, "SG1", string(tokens[0].Value))

	require.Equal(t, TokenSegmentTag, tokens[3].Type)
	require.Equal(t, "S12", string(tokens[3].Value))

	require.Equal(t, TokenSegmentTag, tokens[6].Type)
	require.Equal(t, "TAX", string(tokens[6].Value))
}

func TestLexer_NextToken_RepetitionSeparator(t *testing.T) {
	t.Parallel()

	// The default repetition separator is '*' (space in UNA:+.? ').
	// Using custom delimiters with '*' as repetition.
	delimiters := &Delimiters{
		ComponentSeparator:   ':',
		DataElementSeparator: '+',
		DecimalNotation:      '.',
		ReleaseCharacter:     '?',
		RepetitionSeparator:  '*',
		SegmentTerminator:    '\'',
	}

	// For now, repetition separators are treated like data element separators
	// since the lexer doesn't have a special RepetitionSeparator token.
	input := "TST+A*B'"
	lexer := NewLexer([]byte(input), delimiters)

	tokens := collectTokens(lexer)

	// The value "A*B" should be preserved as a single data element value
	// since '*' is not a structural separator at the lexer level in this design.
	require.Equal(t, TokenSegmentTag, tokens[0].Type)
	require.Equal(t, "TST", string(tokens[0].Value))
	require.Equal(t, TokenDataElement, tokens[1].Type)
	require.Equal(t, "A*B", string(tokens[1].Value))
}

func TestLexer_NextToken_OnlyWhitespace(t *testing.T) {
	t.Parallel()

	lexer := NewLexer([]byte("   \n\t  "), DefaultDelimiters())
	tokens := collectTokens(lexer)

	require.Len(t, tokens, 1)
	require.Equal(t, TokenEOF, tokens[0].Type)
}

func TestLexer_NextToken_SegmentTagWithNewline(t *testing.T) {
	t.Parallel()

	// Newlines between segments should be skipped.
	input := "UNB+X'\r\nUNH+Y'"
	lexer := NewLexer([]byte(input), DefaultDelimiters())
	tokens := collectTokens(lexer)

	require.Equal(t, TokenSegmentTag, tokens[0].Type)
	require.Equal(t, "UNB", string(tokens[0].Value))

	require.Equal(t, TokenSegmentTag, tokens[3].Type)
	require.Equal(t, "UNH", string(tokens[3].Value))
}

func TestLexer_NextToken_VeryLongValue(t *testing.T) {
	t.Parallel()

	longValue := strings.Repeat("ABCDEFGHIJ", 500) // 5000 chars
	input := "TST+" + longValue + "'"

	lexer := NewLexer([]byte(input), DefaultDelimiters())
	tokens := collectTokens(lexer)

	require.Equal(t, TokenDataElement, tokens[1].Type)
	require.Equal(t, longValue, string(tokens[1].Value))
}
