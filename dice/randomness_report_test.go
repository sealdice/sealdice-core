//nolint:testpackage
package dice

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	ds "github.com/sealdice/dicescript"
)

func TestGenerateRandomnessSamples(t *testing.T) {
	if os.Getenv("SEALDICE_RANDOMNESS_GENERATE") == "" {
		t.Skip("set SEALDICE_RANDOMNESS_GENERATE=1 to generate randomness samples")
	}

	outDir := getenvOr("SEALDICE_RANDOMNESS_OUT_DIR", filepath.Join("temp", "randomness", "samples"))
	modesRaw := getenvOr("SEALDICE_RANDOMNESS_MODES", "pcg,crypto,nist,gm")
	sampleCount := mustParsePositiveInt(t, "SEALDICE_RANDOMNESS_SAMPLES", 20)
	bitCount := mustParsePositiveInt(t, "SEALDICE_RANDOMNESS_BITS", 1000000)
	if bitCount%8 != 0 {
		t.Fatalf("SEALDICE_RANDOMNESS_BITS must be divisible by 8, got %d", bitCount)
	}
	byteCount := bitCount / 8

	modes := parseRandomnessModes(t, modesRaw)
	for _, mode := range modes {
		modeDir := filepath.Join(outDir, string(mode))
		if err := os.MkdirAll(modeDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", modeDir, err)
		}
		for i := range sampleCount {
			src, err := newDiceSourceForMode(mode)
			if err != nil {
				t.Fatalf("newDiceSourceForMode(%s): %v", mode, err)
			}
			buf := make([]byte, byteCount)
			fillRandomnessBuffer(src, buf)

			name := filepath.Join(modeDir, fmt.Sprintf("sample_%03d.bin", i+1))
			if err := os.WriteFile(name, buf, 0o644); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
		}
	}
}

func fillRandomnessBuffer(src ds.DiceSource, buf []byte) {
	for offset := 0; offset < len(buf); offset += 8 {
		v := src.Uint64()
		remain := len(buf) - offset
		if remain >= 8 {
			binary.BigEndian.PutUint64(buf[offset:offset+8], v)
			continue
		}

		var tail [8]byte
		binary.BigEndian.PutUint64(tail[:], v)
		copy(buf[offset:], tail[:remain])
	}
}

func parseRandomnessModes(t *testing.T, raw string) []DiceRandomMode {
	t.Helper()

	parts := strings.Split(raw, ",")
	modes := make([]DiceRandomMode, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mode, ok := parseDiceRandomModeStrict(part)
		if !ok {
			t.Fatalf("unsupported mode %q", part)
		}
		modes = append(modes, mode)
	}
	if len(modes) == 0 {
		t.Fatal("no randomness modes selected")
	}
	return modes
}

func mustParsePositiveInt(t *testing.T, env string, fallback int) int {
	t.Helper()

	raw := os.Getenv(env)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		t.Fatalf("%s must be a positive integer, got %q", env, raw)
	}
	return n
}

func getenvOr(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
