package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterchangeStatus_IsValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status InterchangeStatus
		want   bool
	}{
		{
			name:   "completed is valid",
			status: StatusCompleted,
			want:   true,
		},
		{
			name:   "failed is valid",
			status: StatusFailed,
			want:   true,
		},
		{
			name:   "empty string is invalid",
			status: InterchangeStatus(""),
			want:   false,
		},
		{
			name:   "unknown status is invalid",
			status: InterchangeStatus("pending"),
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.status.IsValid()
			require.Equal(t, tc.want, got)
		})
	}
}

func TestInterchangeStatus_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status InterchangeStatus
		want   string
	}{
		{
			name:   "completed string",
			status: StatusCompleted,
			want:   "completed",
		},
		{
			name:   "failed string",
			status: StatusFailed,
			want:   "failed",
		},
		{
			name:   "custom status string",
			status: InterchangeStatus("custom"),
			want:   "custom",
		},
		{
			name:   "empty status string",
			status: InterchangeStatus(""),
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.status.String())
		})
	}
}

func TestInterchangeStatus_IsValid_CaseSensitive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status InterchangeStatus
		want   bool
	}{
		{
			name:   "uppercase COMPLETED is invalid",
			status: InterchangeStatus("COMPLETED"),
			want:   false,
		},
		{
			name:   "uppercase FAILED is invalid",
			status: InterchangeStatus("FAILED"),
			want:   false,
		},
		{
			name:   "mixed case Completed is invalid",
			status: InterchangeStatus("Completed"),
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.status.IsValid())
		})
	}
}
