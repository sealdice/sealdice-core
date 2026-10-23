package codec //nolint:testpackage // Tests verify internal large-value admission control.

import (
	"bytes"
	"sync"
	"testing"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
)

func deterministicNoise(size int) []byte {
	value := make([]byte, size)
	state := uint64(0x9e3779b97f4a7c15)
	for index := range value {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		value[index] = byte(state)
	}
	return value
}

func TestCompressorUsesFourKiBDefaultThreshold(t *testing.T) {
	compressor, err := New(Options{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	belowThreshold := bytes.Repeat([]byte("a"), 2*1024)
	encoded, kind := compressor.CompressWithKind(belowThreshold, store.ValueKindText)
	if kind != CodecRaw || !bytes.Equal(encoded, belowThreshold) {
		t.Fatalf("2 KiB value codec=%d encoded_bytes=%d", kind, len(encoded))
	}

	aboveThreshold := bytes.Repeat([]byte("compressible-json-value"), 512)
	encoded, kind = compressor.CompressWithKind(aboveThreshold, store.ValueKindJSON)
	if kind != CodecZstd || len(encoded) >= len(aboveThreshold) {
		t.Fatalf("compressible value codec=%d encoded_bytes=%d raw_bytes=%d", kind, len(encoded), len(aboveThreshold))
	}
	decoded, err := compressor.Decompress(encoded, kind)
	if err != nil || !bytes.Equal(decoded, aboveThreshold) {
		t.Fatalf("round trip bytes=%d err=%v", len(decoded), err)
	}
}

func TestCompressorLargeValueSampleGate(t *testing.T) {
	compressor, err := New(Options{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	compressible := bytes.Repeat([]byte("repeated-json-field:"), 128*1024)
	if !compressor.sampleWorthCompressing(compressible) {
		t.Fatal("compressible sample was rejected")
	}
	incompressible := deterministicNoise(2 * 1024 * 1024)
	if compressor.sampleWorthCompressing(incompressible) {
		t.Fatal("incompressible sample was accepted")
	}
	encoded, kind := compressor.CompressWithKind(incompressible, store.ValueKindText)
	if kind != CodecRaw || len(encoded) != len(incompressible) {
		t.Fatalf("incompressible value codec=%d encoded_bytes=%d", kind, len(encoded))
	}
}

func TestCompressorConcurrentRoundTrips(t *testing.T) {
	compressor, err := New(Options{Enabled: true, MinSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	value := bytes.Repeat([]byte("concurrent-value"), 16*1024)

	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			encoded, kind := compressor.CompressWithKind(value, store.ValueKindText)
			decoded, decodeErr := compressor.Decompress(encoded, kind)
			if decodeErr != nil || !bytes.Equal(decoded, value) {
				t.Errorf("concurrent round trip codec=%d bytes=%d err=%v", kind, len(decoded), decodeErr)
			}
		}()
	}
	wait.Wait()
}

func TestCompressorRecognizesLargeCompressedFrame(t *testing.T) {
	compressor, err := New(Options{Enabled: true, MinSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	value := bytes.Repeat([]byte("large-frame"), 256*1024)
	encoded, kind := compressor.CompressWithKind(value, store.ValueKindText)
	if kind != CodecZstd {
		t.Fatalf("codec = %d", kind)
	}
	if !compressor.requiresLargeValueSlot(encoded, kind) {
		t.Fatal("large compressed frame was not recognized")
	}
}
