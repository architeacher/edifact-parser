package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReprocessFilter_IsEmpty(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	sender := "9900000000001"
	msgType := "MSCONS"
	fromDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		filter ReprocessFilter
		want   bool
	}{
		{
			name:   "empty filter",
			filter: ReprocessFilter{},
			want:   true,
		},
		{
			name:   "interchange ID set",
			filter: ReprocessFilter{InterchangeID: &interchangeID},
			want:   false,
		},
		{
			name:   "sender ID set",
			filter: ReprocessFilter{SenderID: &sender},
			want:   false,
		},
		{
			name:   "message type set",
			filter: ReprocessFilter{MessageType: &msgType},
			want:   false,
		},
		{
			name:   "from date set",
			filter: ReprocessFilter{FromDate: &fromDate},
			want:   false,
		},
		{
			name:   "to date set",
			filter: ReprocessFilter{ToDate: &toDate},
			want:   false,
		},
		{
			name: "all fields set",
			filter: ReprocessFilter{
				InterchangeID: &interchangeID,
				SenderID:      &sender,
				MessageType:   &msgType,
				FromDate:      &fromDate,
				ToDate:        &toDate,
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.filter.IsEmpty())
		})
	}
}

func TestMessageFilter_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		limit     int
		wantLimit int
	}{
		{
			name:      "zero limit defaults to 20",
			limit:     0,
			wantLimit: DefaultPageLimit,
		},
		{
			name:      "negative limit defaults to 20",
			limit:     -5,
			wantLimit: DefaultPageLimit,
		},
		{
			name:      "valid limit preserved",
			limit:     50,
			wantLimit: 50,
		},
		{
			name:      "over max capped to 100",
			limit:     200,
			wantLimit: MaxPageLimit,
		},
		{
			name:      "exact max preserved",
			limit:     100,
			wantLimit: MaxPageLimit,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			filter := &MessageFilter{Limit: tc.limit}
			err := filter.Validate()

			require.NoError(t, err)
			require.Equal(t, tc.wantLimit, filter.Limit)
		})
	}
}
