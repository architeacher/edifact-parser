package edifact

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func testdataPath(parts ...string) string {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "testdata", "edifact")

	return filepath.Join(append([]string{base}, parts...)...)
}

func readTestFile(t *testing.T, parts ...string) []byte {
	t.Helper()

	path := testdataPath(parts...)

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading test file %s", path)

	return data
}

func TestNewParser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data returns error",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "nil data returns error",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "valid data with UNA",
			data:    []byte("UNA:+.? 'UNB+S+R'"),
			wantErr: false,
		},
		{
			name:    "valid data without UNA",
			data:    []byte("UNB+S+R'"),
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser, err := NewParser(tc.data)

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, parser)
			} else {
				require.NoError(t, err)
				require.NotNil(t, parser)
			}
		})
	}
}

func TestParser_Parse_ValidFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		file              string
		wantSenderID      string
		wantReceiverID    string
		wantReference     string
		wantMessageCount  int
		wantSubscriptions int
		wantMessageType   string
	}{
		{
			name:              "single MSCONS message",
			file:              "mscons_single.edi",
			wantSenderID:      "1234567890123",
			wantReceiverID:    "9876543210987",
			wantReference:     "00000001",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "multiple MSCONS messages",
			file:              "MSCONS_interval_readings.txt",
			wantSenderID:      "9904628000007",
			wantReceiverID:    "9985046000001",
			wantReference:     "555331652PF",
			wantMessageCount:  2,
			wantSubscriptions: 2,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "INVOIC message",
			file:              "invoic_simple.edi",
			wantSenderID:      "SENDER001",
			wantReceiverID:    "RECVR001",
			wantReference:     "INV00001",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "INVOIC",
		},
		{
			name:              "UTILMD message",
			file:              "utilmd_simple.edi",
			wantSenderID:      "UTIL001",
			wantReceiverID:    "GRID001",
			wantReference:     "UTL00001",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "UTILMD",
		},
		{
			name:              "without UNA uses default delimiters",
			file:              "no_una.edi",
			wantSenderID:      "NOUNASENDER",
			wantReceiverID:    "NOUNARECEIVER",
			wantReference:     "NOUNA001",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "message without LOC segment",
			file:              "no_loc.edi",
			wantSenderID:      "SENDER002",
			wantReceiverID:    "RECVR002",
			wantReference:     "NOLOC001",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "file with newlines between segments",
			file:              "mscons_newlines.edi",
			wantSenderID:      "1234567890123",
			wantReceiverID:    "9876543210987",
			wantReference:     "NL000001",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "three messages in one interchange",
			file:              "mscons_three_messages.edi",
			wantSenderID:      "SENDER003",
			wantReceiverID:    "RECVR003",
			wantReference:     "MULTI003",
			wantMessageCount:  3,
			wantSubscriptions: 3,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "shared subscription across messages",
			file:              "shared_subscription.edi",
			wantSenderID:      "SENDER004",
			wantReceiverID:    "RECVR004",
			wantReference:     "SHARED01",
			wantMessageCount:  2,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "minimal valid interchange",
			file:              "minimal.edi",
			wantSenderID:      "S",
			wantReceiverID:    "R",
			wantReference:     "REF001",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "MSCONS",
		},

		// Real-world production EDIFACT files.
		{
			name:              "real IFTSTA transport status",
			file:              "IFTSTA__9985046000001_9907778000000_20250107_10000000001683.txt",
			wantSenderID:      "9985046000001",
			wantReceiverID:    "9907778000000",
			wantReference:     "10000000001683",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "IFTSTA",
		},
		{
			name:              "real INVOIC grid invoice",
			file:              "INVOIC__9900080000007_9985046000001_20240828_452453099PF.txt",
			wantSenderID:      "9900080000007",
			wantReceiverID:    "9985046000001",
			wantReference:     "452453099PF",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "INVOIC",
		},
		{
			name:              "real MSCONS large multi-message interchange",
			file:              "MSCONS_TL_9907634000003_9985046000001_20241210_12526858PF.txt",
			wantSenderID:      "9907634000003",
			wantReceiverID:    "9985046000001",
			wantReference:     "12526858PF",
			wantMessageCount:  24,
			wantSubscriptions: 0,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real MSCONS interval readings two subscriptions",
			file:              "MSCONS_interval_readings.txt",
			wantSenderID:      "9904628000007",
			wantReceiverID:    "9985046000001",
			wantReference:     "555331652PF",
			wantMessageCount:  2,
			wantSubscriptions: 2,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real MSCONS multi-message shared subscription",
			file:              "MSCONS_multi_message_bug.txt",
			wantSenderID:      "9905048000007",
			wantReceiverID:    "9985046000001",
			wantReference:     "CS0000000JKO9W",
			wantMessageCount:  2,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real MSCONS with empty LOC subscription",
			file:              "MSCONS_reading_with_configuration_id.txt",
			wantSenderID:      "9978414000000",
			wantReceiverID:    "9985046000001",
			wantReference:     "003898184881",
			wantMessageCount:  2,
			wantSubscriptions: 0,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real MSCONS with meter number",
			file:              "MSCONS_reading_with_meter_number.txt",
			wantSenderID:      "9985046000001",
			wantReceiverID:    "9911728000001",
			wantReference:     "10000000015725",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real MSCONS with HT/NT tariff bands",
			file:              "MSCONS_reading_with_meter_number_ht_nt.txt",
			wantSenderID:      "9985046000001",
			wantReceiverID:    "9911728000001",
			wantReference:     "10000000015725",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real MSCONS with order reference",
			file:              "MSCONS_with_order_reference.txt",
			wantSenderID:      "9911013000005",
			wantReceiverID:    "9985046000001",
			wantReference:     "700194277400",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "MSCONS",
		},
		{
			name:              "real ORDRSP order response",
			file:              "ORDRSP.txt",
			wantSenderID:      "9906431000000",
			wantReceiverID:    "9985046000001",
			wantReference:     "000596233588",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "ORDRSP",
		},
		{
			name:              "real PARTIN party identification",
			file:              "PARTIN__4038777000004_9985046000001_20250214_10000000498915.txt",
			wantSenderID:      "4038777000004",
			wantReceiverID:    "9985046000001",
			wantReference:     "10000000498915",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "PARTIN",
		},
		{
			name:              "real UTILMD multi-profile",
			file:              "UTILMD__4041407000008_9985046000001_20240920_041644994028.txt",
			wantSenderID:      "4041407000008",
			wantReceiverID:    "9985046000001",
			wantReference:     "041644994028",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "UTILMD",
		},
		{
			name:              "real UTILMD single profile",
			file:              "UTILMD__9900080000007_9985046000001_20240912_457820384PF.txt",
			wantSenderID:      "9900080000007",
			wantReceiverID:    "9985046000001",
			wantReference:     "457820384PF",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "UTILMD",
		},
		{
			name:              "real UTILMD multiple signups",
			file:              "UTILMD_multiple_signups.txt",
			wantSenderID:      "9900396000006",
			wantReceiverID:    "9985046000001",
			wantReference:     "10000181914168",
			wantMessageCount:  1,
			wantSubscriptions: 0,
			wantMessageType:   "UTILMD",
		},
		{
			name:              "real INVOIC storno reversal",
			file:              "invoic_storno.txt",
			wantSenderID:      "4041407000008",
			wantReceiverID:    "9985046000001",
			wantReference:     "20250521000271",
			wantMessageCount:  1,
			wantSubscriptions: 1,
			wantMessageType:   "INVOIC",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := readTestFile(t, "valid", tc.file)

			parser, err := NewParser(data)
			require.NoError(t, err)

			result, err := parser.Parse()
			require.NoError(t, err, "parsing %s", tc.file)
			require.NotNil(t, result)

			require.Equal(t, tc.wantSenderID, result.SenderID, "sender ID")
			require.Equal(t, tc.wantReceiverID, result.ReceiverID, "receiver ID")
			require.Equal(t, tc.wantReference, result.Reference, "reference")
			require.Len(t, result.Messages, tc.wantMessageCount, "message count")
			require.Len(t, result.Subscriptions, tc.wantSubscriptions, "subscription count")

			// Validate first message type.
			if tc.wantMessageCount > 0 {
				require.Equal(t, tc.wantMessageType, result.Messages[0].MessageType, "message type")
				require.NotEmpty(t, result.Messages[0].MessageReference, "message reference")
				require.NotEmpty(t, result.Messages[0].Segments, "segments not empty")
			}

			// Content hash should always be present.
			require.Len(t, result.ContentHash, 64, "content hash length")
		})
	}
}

func TestParser_Parse_InvalidFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		file        string
		wantErrText string
	}{
		{
			name:        "missing UNB header",
			file:        "missing_unb.edi",
			wantErrText: "missing UNB interchange header",
		},
		{
			name:        "missing UNZ trailer",
			file:        "missing_unz.edi",
			wantErrText: "missing UNZ interchange trailer",
		},
		{
			name:        "wrong message count in UNZ",
			file:        "wrong_message_count.edi",
			wantErrText: "UNZ message count does not match",
		},
		{
			name:        "mismatched interchange reference",
			file:        "mismatched_reference.edi",
			wantErrText: "UNZ interchange reference does not match",
		},
		{
			name:        "truncated UNB segment",
			file:        "truncated_unb.edi",
			wantErrText: "UNB segment has insufficient data elements",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := readTestFile(t, "invalid", tc.file)

			parser, err := NewParser(data)
			require.NoError(t, err)

			result, err := parser.Parse()
			require.Error(t, err)
			require.Nil(t, result)
			require.Contains(t, err.Error(), tc.wantErrText)
		})
	}
}

func TestParser_Parse_ContentHash(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "minimal.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)

	// Same data should produce same hash.
	require.Len(t, result.ContentHash, 64)

	parser2, err := NewParser(data)
	require.NoError(t, err)

	result2, err := parser2.Parse()
	require.NoError(t, err)

	require.Equal(t, result.ContentHash, result2.ContentHash, "deterministic hash")
}

func TestParser_Parse_SenderQualifier(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "mscons_single.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)

	require.Equal(t, "14", result.SenderQualifier, "sender qualifier")
	require.Equal(t, "14", result.ReceiverQualifier, "receiver qualifier")
}

func TestParser_Parse_PreparedAt(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "mscons_single.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)

	require.False(t, result.PreparedAt.IsZero(), "prepared_at should not be zero")
	require.Equal(t, 2026, result.PreparedAt.Year())
	require.Equal(t, 2, int(result.PreparedAt.Month()))
	require.Equal(t, 19, result.PreparedAt.Day())
}

func TestParser_Parse_MessageSegments(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "mscons_single.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)

	msg := result.Messages[0]

	// First segment should be UNH.
	require.Equal(t, "UNH", msg.Segments[0]["tag"])

	// Last segment should be UNT.
	lastSeg := msg.Segments[len(msg.Segments)-1]
	require.Equal(t, "UNT", lastSeg["tag"])
}

func TestParser_Parse_MessageVersionAndRelease(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "invoic_simple.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)

	msg := result.Messages[0]
	require.Equal(t, "INVOIC", msg.MessageType)
	require.Equal(t, "D", msg.MessageVersion)
	require.Equal(t, "11A", msg.MessageRelease)
}

func TestParser_Parse_LOCSubscriptionExtraction(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "mscons_single.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	require.Equal(t, "DE0001234567890000000000000123456", result.Messages[0].SubscriptionID)
	require.Equal(t, []string{"DE0001234567890000000000000123456"}, result.Subscriptions)
}

func TestParser_Parse_SharedSubscriptionDedup(t *testing.T) {
	t.Parallel()

	data := readTestFile(t, "valid", "shared_subscription.edi")

	parser, err := NewParser(data)
	require.NoError(t, err)

	result, err := parser.Parse()
	require.NoError(t, err)

	require.Len(t, result.Messages, 2)
	require.Len(t, result.Subscriptions, 1, "shared subscription should be deduplicated")
	require.Equal(t, result.Messages[0].SubscriptionID, result.Messages[1].SubscriptionID)
}

func TestParser_Parse_InlineEDIFACT(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		input            string
		wantSenderID     string
		wantReceiverID   string
		wantReference    string
		wantMessageCount int
	}{
		{
			name:             "minimal inline",
			input:            "UNA:+.? 'UNB+UNOC:3+SEND:14+RECV:14+260101:0800+REF1'UNH+M1+MSCONS:D:04B:UN'UNT+2+M1'UNZ+1+REF1'",
			wantSenderID:     "SEND",
			wantReceiverID:   "RECV",
			wantReference:    "REF1",
			wantMessageCount: 1,
		},
		{
			name:             "without UNA",
			input:            "UNB+UNOC:3+ABC:14+XYZ:14+260301:1200+R2'UNH+1+INVOIC:D:11A:UN'UNT+2+1'UNZ+1+R2'",
			wantSenderID:     "ABC",
			wantReceiverID:   "XYZ",
			wantReference:    "R2",
			wantMessageCount: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser, err := NewParser([]byte(tc.input))
			require.NoError(t, err)

			result, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, result)

			require.Equal(t, tc.wantSenderID, result.SenderID)
			require.Equal(t, tc.wantReceiverID, result.ReceiverID)
			require.Equal(t, tc.wantReference, result.Reference)
			require.Len(t, result.Messages, tc.wantMessageCount)
		})
	}
}

func TestParser_Parse_ErrorMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		input            string
		wantErrContains  string
	}{
		{
			name:            "missing UNB",
			input:           "UNA:+.? 'UNH+1+MSCONS:D:04B:UN'UNT+2+1'UNZ+1+REF'",
			wantErrContains: "missing UNB",
		},
		{
			name:            "empty sender",
			input:           "UNA:+.? 'UNB+UNOC:3+:14+R:14+260101:0000+REF'UNH+1+MSCONS:D:04B:UN'UNT+2+1'UNZ+1+REF'",
			wantErrContains: "UNB sender identification is empty",
		},
		{
			name:            "empty receiver",
			input:           "UNA:+.? 'UNB+UNOC:3+S:14+:14+260101:0000+REF'UNH+1+MSCONS:D:04B:UN'UNT+2+1'UNZ+1+REF'",
			wantErrContains: "UNB receiver identification is empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser, err := NewParser([]byte(tc.input))
			require.NoError(t, err)

			result, err := parser.Parse()
			require.Error(t, err)
			require.Nil(t, result)
			require.Contains(t, err.Error(), tc.wantErrContains)
		})
	}
}

func TestComputeContentHash(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		data     []byte
		wantLen  int
	}{
		{
			name:    "non-empty data",
			data:    []byte("test data"),
			wantLen: 64,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantLen: 64,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hash := computeContentHash(tc.data)
			require.Len(t, hash, tc.wantLen)
		})
	}
}

func TestSegmentToMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		tag     string
		seg     segment
		wantTag string
	}{
		{
			name: "simple segment with single-value elements",
			tag:  "BGM",
			seg: segment{
				dataElements: [][]string{
					{"7"},
					{"MSG001"},
					{"9"},
				},
			},
			wantTag: "BGM",
		},
		{
			name: "segment with composite element",
			tag:  "UNB",
			seg: segment{
				dataElements: [][]string{
					{"UNOC", "3"},
					{"SENDER", "14"},
				},
			},
			wantTag: "UNB",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := segmentToMap(tc.tag, tc.seg)

			require.Equal(t, tc.wantTag, result["tag"])

			for index, element := range tc.seg.dataElements {
				key := "de" + string(rune('0'+index+1))

				if len(element) == 1 {
					require.Equal(t, element[0], result[key])
				} else {
					require.Equal(t, element, result[key])
				}
			}
		})
	}
}
