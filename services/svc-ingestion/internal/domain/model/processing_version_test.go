package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewProcessingVersion(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name          string
		interchangeID uuid.UUID
		versionNumber int
		parserVersion string
		formatVersion string
		wantErr       string
	}{
		{
			name:          "valid processing version",
			interchangeID: interchangeID,
			versionNumber: 1,
			parserVersion: "1.0.0",
			formatVersion: "FV2310",
		},
		{
			name:          "version number zero",
			interchangeID: interchangeID,
			versionNumber: 0,
			parserVersion: "1.0.0",
			formatVersion: "FV2310",
			wantErr:       "version number",
		},
		{
			name:          "negative version number",
			interchangeID: interchangeID,
			versionNumber: -1,
			parserVersion: "1.0.0",
			formatVersion: "FV2310",
			wantErr:       "version number",
		},
		{
			name:          "empty parser version",
			interchangeID: interchangeID,
			versionNumber: 1,
			parserVersion: "",
			formatVersion: "FV2310",
			wantErr:       "parser version",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewProcessingVersion(
				tc.interchangeID,
				tc.versionNumber,
				tc.parserVersion,
				tc.formatVersion,
			)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotEqual(t, uuid.Nil, got.ID)
			require.Equal(t, tc.interchangeID, got.InterchangeID)
			require.Equal(t, tc.versionNumber, got.VersionNumber)
			require.Equal(t, tc.parserVersion, got.ParserVersion)
			require.Equal(t, tc.formatVersion, got.FormatVersion)
			require.True(t, got.IsActive)
			require.False(t, got.CreatedAt.IsZero())
		})
	}
}

func TestNewProcessingVersion_NilInterchangeID(t *testing.T) {
	t.Parallel()

	got, err := NewProcessingVersion(uuid.Nil, 1, "1.0.0", "FV2310")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, uuid.Nil, got.InterchangeID)
}

func TestNewProcessingVersion_EmptyFormatVersion(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewProcessingVersion(interchangeID, 1, "1.0.0", "")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got.FormatVersion)
}

func TestNewProcessingVersion_HighVersionNumber(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewProcessingVersion(interchangeID, 999, "2.0.0", "FV2410")

	require.NoError(t, err)
	require.Equal(t, 999, got.VersionNumber)
}

func TestNewProcessingVersion_IsActiveByDefault(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewProcessingVersion(interchangeID, 1, "1.0.0", "FV2310")

	require.NoError(t, err)
	require.True(t, got.IsActive)
}

func TestNewProcessingVersion_UniqueIDs(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	v1, err := NewProcessingVersion(interchangeID, 1, "1.0.0", "FV2310")
	require.NoError(t, err)

	v2, err := NewProcessingVersion(interchangeID, 2, "1.0.0", "FV2310")
	require.NoError(t, err)

	require.NotEqual(t, v1.ID, v2.ID)
}

func TestProcessingVersion_DeactivateAndReactivate(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	version, err := NewProcessingVersion(interchangeID, 1, "1.0.0", "FV2310")
	require.NoError(t, err)

	version.Deactivate()
	require.False(t, version.IsActive)

	version.Activate()
	require.True(t, version.IsActive)

	version.Deactivate()
	require.False(t, version.IsActive)
}

func TestProcessingVersion_Deactivate(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	version, err := NewProcessingVersion(interchangeID, 1, "1.0.0", "FV2310")
	require.NoError(t, err)
	require.True(t, version.IsActive)

	version.Deactivate()
	require.False(t, version.IsActive)
}

func TestProcessingVersion_Activate(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	version, err := NewProcessingVersion(interchangeID, 1, "1.0.0", "FV2310")
	require.NoError(t, err)

	version.Deactivate()
	require.False(t, version.IsActive)

	version.Activate()
	require.True(t, version.IsActive)
}
