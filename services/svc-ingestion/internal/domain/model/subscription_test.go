package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewSubscription(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		identifier string
		wantErr    string
	}{
		{
			name:       "valid subscription",
			identifier: "DE0000801043700000000000011145670",
		},
		{
			name:       "empty identifier",
			identifier: "",
			wantErr:    "identifier",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewSubscription(tc.identifier)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotEqual(t, uuid.Nil, got.ID)
			require.Equal(t, tc.identifier, got.Identifier)
			require.False(t, got.CreatedAt.IsZero())
		})
	}
}

func TestNewSubscription_UniqueIDs(t *testing.T) {
	t.Parallel()

	s1, err := NewSubscription("DE0000801043700000000000011145670")
	require.NoError(t, err)

	s2, err := NewSubscription("DE0000801043700000000000011145670")
	require.NoError(t, err)

	require.NotEqual(t, s1.ID, s2.ID)
}

func TestNewSubscription_DifferentIdentifiers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		identifier string
	}{
		{
			name:       "German metering point",
			identifier: "DE0000801043700000000000011145670",
		},
		{
			name:       "short identifier",
			identifier: "MP-001",
		},
		{
			name:       "alphanumeric identifier",
			identifier: "GRID-OP-9900000000001-LOC-42",
		},
		{
			name:       "numeric only",
			identifier: "1234567890",
		},
		{
			name:       "single character",
			identifier: "X",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewSubscription(tc.identifier)

			require.NoError(t, err)
			require.Equal(t, tc.identifier, got.Identifier)
		})
	}
}

func TestNewSubscription_WhitespaceOnlyIdentifier(t *testing.T) {
	t.Parallel()

	// Whitespace-only is not empty, so it should succeed (validation is caller's concern).
	got, err := NewSubscription("   ")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "   ", got.Identifier)
}

func TestNewSubscription_ErrorIsSentinel(t *testing.T) {
	t.Parallel()

	_, err := NewSubscription("")

	require.ErrorIs(t, err, ErrEmptyIdentifier)
}
