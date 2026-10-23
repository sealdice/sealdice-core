// Package codec provides zstd compression for value blobs.
//
// It is safe for concurrent use via EncodeAll / DecodeAll.
package codec

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"github.com/klauspost/compress/zstd"
)

const (
	CodecRaw  uint8 = 0
	CodecZstd uint8 = 1
)

// Defaults applied when zero-value fields are detected.
const (
	DefaultZstdLevel = 1 // SpeedFastest
	DefaultMinSize   = 4 * 1024

	base64DetectionMinLen  = 2048
	base64DetectionSample  = 4096
	minPrintableTextRatio  = 0.95
	minBase64DistinctChars = 16

	largeValueSampleThreshold = 1 << 20
	compressionSampleSize     = 64 << 10
	minimumSampleSavings      = 5
	encoderWindowSize         = 1 << 20
)

// Options configures compression behaviour.
type Options struct {
	// Enabled controls whether newly-written values are compressed.
	// Values already stored in compressed form are always decompressed on
	// read regardless of this flag.
	Enabled bool
	// Level sets the zstd compression level (see zstd.SpeedDefault etc.).
	// Zero means the default (1 / SpeedFastest).
	Level int
	// MinSize is the minimum uncompressed blob length in bytes for which
	// compression is attempted. Zero means the default (4 KiB).
	MinSize int
}

func (o Options) effectiveLevel() int {
	if o.Level <= 0 {
		return DefaultZstdLevel
	}
	return o.Level
}

func (o Options) effectiveMinSize() int {
	if o.MinSize <= 0 {
		return DefaultMinSize
	}
	return o.MinSize
}

// Compressor wraps a zstd encoder and decoder.
// *Encoder is nil when Options.Enabled is false.
// *Decoder is always initialised so that previously-compressed rows remain
// readable even after the feature is toggled off.
type Compressor struct {
	enc             *zstd.Encoder
	dec             *zstd.Decoder
	minSize         int
	enabled         bool
	largeValueSlots chan struct{}
}

// New creates a Compressor from o.
func New(o Options) (*Compressor, error) {
	dec, err := zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
	)
	if err != nil {
		return nil, fmt.Errorf("codec decoder: %w", err)
	}

	c := &Compressor{
		dec:             dec,
		enabled:         o.Enabled,
		minSize:         o.effectiveMinSize(),
		largeValueSlots: make(chan struct{}, 1),
	}

	if o.Enabled {
		enc, err := zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(o.effectiveLevel())),
			zstd.WithEncoderConcurrency(1),
			zstd.WithLowerEncoderMem(true),
			zstd.WithWindowSize(encoderWindowSize),
		)
		if err != nil {
			return nil, fmt.Errorf("codec encoder: %w", err)
		}
		c.enc = enc
	}

	return c, nil
}

// CompressWithKind attempts compression using an explicit or inferred value kind.
func (c *Compressor) CompressWithKind(value []byte, kind store.ValueKind) ([]byte, uint8) {
	if !c.shouldCompress(value, kind) {
		return value, CodecRaw
	}
	if !c.sampleWorthCompressing(value) {
		return value, CodecRaw
	}
	release := func() {}
	if len(value) > largeValueSampleThreshold {
		release = c.acquireLargeValueSlot()
	}
	defer release()

	compressed := c.enc.EncodeAll(value, nil)
	if len(compressed) >= len(value) {
		return value, CodecRaw
	}
	return compressed, CodecZstd
}

func (c *Compressor) acquireLargeValueSlot() func() {
	c.largeValueSlots <- struct{}{}
	return func() { <-c.largeValueSlots }
}

func (c *Compressor) requiresLargeValueSlot(value []byte, codec uint8) bool {
	if codec != CodecZstd {
		return false
	}
	if len(value) > largeValueSampleThreshold {
		return true
	}
	var header zstd.Header
	if err := header.Decode(value); err != nil {
		return false
	}
	return header.HasFCS && header.FrameContentSize > largeValueSampleThreshold
}

func (c *Compressor) sampleWorthCompressing(value []byte) bool {
	if len(value) <= largeValueSampleThreshold {
		return true
	}
	sample := sampleBytes(value, compressionSampleSize)
	compressed := c.enc.EncodeAll(sample, nil)
	return len(compressed)*100 <= len(sample)*(100-minimumSampleSavings)
}

func (c *Compressor) shouldCompress(value []byte, kind store.ValueKind) bool {
	if !c.enabled || len(value) < c.minSize {
		return false
	}

	if kind == store.ValueKindUnknown {
		kind = DetectValueKind(value)
	}

	switch kind {
	case store.ValueKindJSON:
		return true
	case store.ValueKindText:
		return !isLikelyBase64Text(value)
	default:
		return false
	}
}

// Decompress reverses the compression indicated by codec.
func (c *Compressor) Decompress(value []byte, codec uint8) ([]byte, error) {
	switch codec {
	case CodecRaw:
		return value, nil
	case CodecZstd:
		release := func() {}
		if c.requiresLargeValueSlot(value, codec) {
			release = c.acquireLargeValueSlot()
		}
		defer release()
		decompressed, err := c.dec.DecodeAll(value, nil)
		if err != nil {
			return nil, fmt.Errorf("zstd decompress: %w", err)
		}
		return decompressed, nil
	default:
		return nil, fmt.Errorf("codec: unknown codec %d", codec)
	}
}

// DetectValueKind infers whether value looks like JSON/text or binary data.
func DetectValueKind(value []byte) store.ValueKind {
	if len(value) == 0 {
		return store.ValueKindText
	}

	sample := sampleBytes(value, base64DetectionSample)
	if bytes.IndexByte(sample, 0x00) >= 0 || !utf8.Valid(sample) {
		return store.ValueKindBinary
	}

	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return store.ValueKindText
	}
	if isLikelyJSON(trimmed) {
		return store.ValueKindJSON
	}
	if isLikelyBase64Text(value) {
		return store.ValueKindBinary
	}
	if isLikelyPrintableText(sample) {
		return store.ValueKindText
	}
	return store.ValueKindBinary
}

func isLikelyJSON(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	switch value[0] {
	case '{', '[':
		return true
	default:
		return false
	}
}

func isLikelyPrintableText(sample []byte) bool {
	total := 0
	printable := 0
	for len(sample) > 0 {
		r, size := utf8.DecodeRune(sample)
		if r == utf8.RuneError && size == 1 {
			return false
		}
		total++
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
		sample = sample[size:]
	}
	if total == 0 {
		return true
	}
	return float64(printable)/float64(total) >= minPrintableTextRatio
}

func isLikelyBase64Text(value []byte) bool {
	if len(value) < base64DetectionMinLen || len(value)%4 != 0 {
		return false
	}

	sample := sampleBytes(value, base64DetectionSample)
	var seen [256]bool
	distinct := 0
	for _, b := range sample {
		switch {
		case 'A' <= b && b <= 'Z':
		case 'a' <= b && b <= 'z':
		case '0' <= b && b <= '9':
		case b == '+' || b == '/' || b == '=' || b == '-' || b == '_':
		default:
			return false
		}
		if !seen[b] {
			seen[b] = true
			distinct++
		}
	}

	return distinct >= minBase64DistinctChars
}

func sampleBytes(value []byte, limit int) []byte {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
