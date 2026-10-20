//nolint:testpackage
package storylog

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"

	"sealdice-core/model"
)

type payloadTestRow struct {
	nickname   string
	imUserID   string
	time       int64
	message    string
	isDice     bool
	commandID  int64
	command    payloadTestCommandInfo
	uniformID  string
	rawMsgID   string
	hasCommand bool
}

type payloadTestCommandInfo struct {
	Cmd    string `json:"cmd"`
	Expr   string `json:"expr"`
	Result int    `json:"result"`
	Detail string `json:"detail,omitempty"`
}

type payloadSizeResult struct {
	v1Size   int
	v105Size int
}

type payloadSizeExpectation string

const (
	payloadV105Larger  payloadSizeExpectation = "v1.5-larger"
	payloadV105Smaller payloadSizeExpectation = "v1.5-smaller"
)

func TestUploadPayloadSizeComparison(t *testing.T) {
	testCases := []struct {
		name        string
		rows        []payloadTestRow
		expectation payloadSizeExpectation
	}{
		{
			name:        "single-short-message",
			rows:        buildRepeatedPayloadRows(1, "ok"),
			expectation: payloadV105Larger,
		},
		{
			name:        "five-short-messages",
			rows:        buildRepeatedPayloadRows(5, "hello"),
			expectation: payloadV105Larger,
		},
		{
			name: "hundred-medium-repeated-messages",
			rows: buildRepeatedPayloadRows(
				100,
				"alice says the same thing again because this case is intentionally repetitive for compression",
			),
			expectation: payloadV105Larger,
		},
		{
			name:        "hundred-random-messages",
			rows:        buildRandomPayloadRows(100, 42, 80, 180),
			expectation: payloadV105Larger,
		},
		{
			name:        "thirty-thousand-mixed-messages",
			rows:        buildLargePayloadRows(30000),
			expectation: payloadV105Smaller,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := measureUploadPayloadSizes(t, tc.rows)
			ratio := float64(got.v105Size) / float64(got.v1Size)

			t.Logf(
				"rows=%d v1=%dB (%s) v1.5=%dB (%s) ratio=%.3fx",
				len(tc.rows),
				got.v1Size,
				formatPayloadBytes(got.v1Size),
				got.v105Size,
				formatPayloadBytes(got.v105Size),
				ratio,
			)

			if got.v1Size <= 0 {
				t.Fatalf("unexpected empty v1 payload size: %d", got.v1Size)
			}
			if got.v105Size <= 0 {
				t.Fatalf("unexpected empty v1.5 payload size: %d", got.v105Size)
			}
			switch tc.expectation {
			case payloadV105Larger:
				if got.v105Size <= got.v1Size {
					t.Fatalf("expected v1.5 parquet payload to be larger, got v1=%d v1.5=%d", got.v1Size, got.v105Size)
				}
			case payloadV105Smaller:
				if got.v105Size >= got.v1Size {
					t.Fatalf("expected v1.5 parquet payload to be smaller, got v1=%d v1.5=%d", got.v1Size, got.v105Size)
				}
			default:
				t.Fatalf("unknown expectation: %s", tc.expectation)
			}
		})
	}
}

func measureUploadPayloadSizes(t *testing.T, rows []payloadTestRow) payloadSizeResult {
	t.Helper()

	v1Rows, v105Rows := buildVersionedPayloadRows(t, rows)

	v1Payload, err := encodeV1UploadPayload(v1Rows)
	if err != nil {
		t.Fatalf("encode v1 payload: %v", err)
	}

	v105Payload, err := encodeV105UploadPayload(v105Rows)
	if err != nil {
		t.Fatalf("encode v1.5 payload: %v", err)
	}

	return payloadSizeResult{
		v1Size:   len(v1Payload),
		v105Size: len(v105Payload),
	}
}

func buildVersionedPayloadRows(t *testing.T, rows []payloadTestRow) ([]*model.LogOneItem, []model.LogOneItemParquet) {
	t.Helper()

	v1Rows := make([]*model.LogOneItem, 0, len(rows))
	v105Rows := make([]model.LogOneItemParquet, 0, len(rows))

	for index, row := range rows {
		v1Item := &model.LogOneItem{
			ID:        uint64(index + 1),
			Nickname:  row.nickname,
			IMUserID:  row.imUserID,
			Time:      row.time,
			Message:   row.message,
			IsDice:    row.isDice,
			CommandID: row.commandID,
			RawMsgID:  row.rawMsgID,
			UniformID: row.uniformID,
		}
		v105Item := model.LogOneItemParquet{
			ID:        uint64(index + 1),
			Nickname:  row.nickname,
			IMUserID:  row.imUserID,
			Time:      row.time,
			Message:   row.message,
			IsDice:    row.isDice,
			CommandID: row.commandID,
			UniformID: row.uniformID,
		}

		if row.hasCommand {
			commandInfoBytes, err := json.Marshal(row.command)
			if err != nil {
				t.Fatalf("marshal command info for row %d: %v", index, err)
			}
			v1Item.CommandInfo = row.command
			v105Item.CommandInfoStr = string(commandInfoBytes)
		}

		v1Rows = append(v1Rows, v1Item)
		v105Rows = append(v105Rows, v105Item)
	}

	return v1Rows, v105Rows
}

func encodeV1UploadPayload(rows []*model.LogOneItem) ([]byte, error) {
	data, err := json.Marshal(map[string]interface{}{
		"version": StoryVersionV1,
		"items":   rows,
	})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err = writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeV105UploadPayload(rows []model.LogOneItemParquet) ([]byte, error) {
	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[model.LogOneItemParquet](
		&buf,
		parquet.Compression(&zstd.Codec{}),
		parquet.MaxRowsPerRowGroup(4000),
	)
	if _, err := writer.Write(rows); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildRepeatedPayloadRows(count int, message string) []payloadTestRow {
	rows := make([]payloadTestRow, 0, count)
	names := []string{"alice", "bob", "keeper"}
	userIDs := []string{"qq:1001", "qq:1002", "qq:9001"}

	for i := range count {
		row := payloadTestRow{
			nickname:  names[i%len(names)],
			imUserID:  userIDs[i%len(userIDs)],
			time:      1_720_000_000 + int64(i),
			message:   message,
			uniformID: fmt.Sprintf("uniform-%d", i%len(names)),
			rawMsgID:  fmt.Sprintf("raw-%06d", i+1),
		}

		if i%5 == 0 {
			row.isDice = true
			row.commandID = int64(1000 + i)
			row.message = fmt.Sprintf(".r 1d100 -> %d", 1+(i*17)%100)
			row.command = payloadTestCommandInfo{
				Cmd:    "roll",
				Expr:   "1d100",
				Result: 1 + (i*17)%100,
				Detail: "skill-check",
			}
			row.hasCommand = true
		}

		rows = append(rows, row)
	}

	return rows
}

func buildRandomPayloadRows(count int, seed int64, minLen int, maxLen int) []payloadTestRow {
	rows := make([]payloadTestRow, 0, count)
	random := rand.New(rand.NewSource(seed))
	names := []string{"alice", "bob", "keeper", "eve", "mallory"}

	for i := range count {
		length := minLen
		if maxLen > minLen {
			length += random.Intn(maxLen - minLen + 1)
		}

		row := payloadTestRow{
			nickname:  names[i%len(names)],
			imUserID:  fmt.Sprintf("qq:%04d", 2000+i),
			time:      1_720_100_000 + int64(i*7),
			message:   buildRandomSentence(random, length),
			uniformID: fmt.Sprintf("uniform-random-%d", i%len(names)),
			rawMsgID:  fmt.Sprintf("random-%06d", i+1),
		}

		if i%4 == 0 {
			row.isDice = true
			row.commandID = int64(2000 + i)
			row.command = payloadTestCommandInfo{
				Cmd:    "roll",
				Expr:   fmt.Sprintf("%dd6", 1+random.Intn(3)),
				Result: 1 + random.Intn(18),
				Detail: buildRandomSentence(random, 24),
			}
			row.hasCommand = true
		}

		rows = append(rows, row)
	}

	return rows
}

func buildLargePayloadRows(count int) []payloadTestRow {
	rows := make([]payloadTestRow, 0, count)
	messagePool := []string{
		"the party enters the ruined observatory and checks the broken brass machinery for hidden clues",
		"keeper describes rain tapping on old glass while the table discusses whether to push deeper inside",
		"player repeats a scouting plan and confirms everyone is ready before the next dice check",
		"combat summary line with repeated structure to simulate long but compressible table chatter",
	}
	names := []string{"alice", "bob", "keeper", "david", "erin", "frank"}

	for i := range count {
		row := payloadTestRow{
			nickname:  names[i%len(names)],
			imUserID:  fmt.Sprintf("qq:%05d", 3000+(i%len(names))),
			time:      1_720_200_000 + int64(i),
			message:   messagePool[i%len(messagePool)],
			uniformID: fmt.Sprintf("uniform-large-%d", i%len(names)),
			rawMsgID:  fmt.Sprintf("large-%08d", i+1),
		}

		if i%6 == 0 {
			row.isDice = true
			row.commandID = int64(5000 + i)
			row.message = fmt.Sprintf(".ra stealth %d", 1+(i%100))
			row.command = payloadTestCommandInfo{
				Cmd:    "ra",
				Expr:   "stealth",
				Result: 1 + (i % 100),
				Detail: "large-sample",
			}
			row.hasCommand = true
		}

		rows = append(rows, row)
	}

	return rows
}

func buildRandomSentence(random *rand.Rand, length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz "
	var builder strings.Builder
	builder.Grow(length)
	for range length {
		builder.WriteByte(alphabet[random.Intn(len(alphabet))])
	}
	return strings.TrimSpace(builder.String())
}

func formatPayloadBytes(size int) string {
	switch {
	case size >= 1<<20:
		return fmt.Sprintf("%.2fMiB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.2fKiB", float64(size)/(1<<10))
	default:
		return fmt.Sprintf("%dB", size)
	}
}
