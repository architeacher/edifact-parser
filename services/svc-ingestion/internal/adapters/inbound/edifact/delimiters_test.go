package edifact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultDelimiters(t *testing.T) {
	t.Parallel()

	delimiters := DefaultDelimiters()

	require.Equal(t, byte(':'), delimiters.ComponentSeparator)
	require.Equal(t, byte('+'), delimiters.DataElementSeparator)
	require.Equal(t, byte('.'), delimiters.DecimalNotation)
	require.Equal(t, byte('?'), delimiters.ReleaseCharacter)
	require.Equal(t, byte('*'), delimiters.RepetitionSeparator)
	require.Equal(t, byte('\''), delimiters.SegmentTerminator)
}

func TestParseUNA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   []byte
		want    *Delimiters
		wantErr error
	}{
		{
			name:  "standard UNA with ISO 9735 defaults",
			input: []byte("UNA:+.? '"),
			want: &Delimiters{
				ComponentSeparator:   ':',
				DataElementSeparator: '+',
				DecimalNotation:      '.',
				ReleaseCharacter:     '?',
				RepetitionSeparator:  ' ',
				SegmentTerminator:    '\'',
			},
		},
		{
			name:  "custom delimiters",
			input: []byte("UNA;-.!*~"),
			want: &Delimiters{
				ComponentSeparator:   ';',
				DataElementSeparator: '-',
				DecimalNotation:      '.',
				ReleaseCharacter:     '!',
				RepetitionSeparator:  '*',
				SegmentTerminator:    '~',
			},
		},
		{
			name:  "UNA followed by additional data is accepted",
			input: []byte("UNA:+.? 'UNB+UNOC:3+1234:14+5678:14+210101:1200+00000001'"),
			want: &Delimiters{
				ComponentSeparator:   ':',
				DataElementSeparator: '+',
				DecimalNotation:      '.',
				ReleaseCharacter:     '?',
				RepetitionSeparator:  ' ',
				SegmentTerminator:    '\'',
			},
		},
		{
			name:    "too short input",
			input:   []byte("UNA:+"),
			wantErr: ErrInvalidUNALength,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: ErrInvalidUNALength,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: ErrInvalidUNALength,
		},
		{
			name:    "wrong prefix",
			input:   []byte("UNB:+.? '"),
			wantErr: ErrInvalidUNAPrefix,
		},
		{
			name:    "lowercase prefix",
			input:   []byte("una:+.? '"),
			wantErr: ErrInvalidUNAPrefix,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseUNA(tc.input)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDelimiters_IsReleaseChar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		char byte
		want bool
	}{
		{
			name: "default release character matches",
			char: '?',
			want: true,
		},
		{
			name: "non-release character does not match",
			char: '+',
			want: false,
		},
		{
			name: "null byte does not match",
			char: 0,
			want: false,
		},
	}

	delimiters := DefaultDelimiters()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, delimiters.IsReleaseChar(tc.char))
		})
	}
}

func TestDelimiters_IsReleaseChar_CustomDelimiter(t *testing.T) {
	t.Parallel()

	delimiters := &Delimiters{ReleaseCharacter: '!'}

	require.True(t, delimiters.IsReleaseChar('!'))
	require.False(t, delimiters.IsReleaseChar('?'))
}
