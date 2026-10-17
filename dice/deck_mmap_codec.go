package dice

import "github.com/klauspost/compress/zstd"

// Called under deckBlockReader. One decoder is shared across all mapped decks.
func (s *deckBlockStore) decodeFrame(compressed []byte, rawSize int) ([]byte, error) {
	if deckBlockReader.decoder == nil {
		var err error
		deckBlockReader.decoder, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true), zstd.WithDecoderMaxMemory(1<<63))
		if err != nil {
			return nil, err
		}
	}
	return deckBlockReader.decoder.DecodeAll(compressed, make([]byte, 0, rawSize))
}

// The catalog lease keeps these immutable mapped bytes alive until the draw ends.
func (s *deckBlockStore) blockBytes(b deckBlock) ([]byte, error) {
	return s.data[b.offset : b.offset+b.length], nil
}

func (s *deckBlockStore) weightBytes(table, offset, length int) ([]byte, error) {
	return s.data[table+offset : table+offset+length], nil
}

// Caller holds deckBlockReader. Oversized single entries are decoded normally
// but are not retained by the bounded cache.
func cacheDeckBlock(key deckCacheKey, data []byte) {
	if cap(data) > deckCacheBytes {
		return
	}
	for deckBlockReader.bytes+cap(data) > deckCacheBytes {
		last := deckBlockReader.lru.Back()
		entry := last.Value.(deckCacheEntry)
		deckBlockReader.bytes -= cap(entry.data)
		delete(deckBlockReader.entries, entry.key)
		deckBlockReader.lru.Remove(last)
	}
	deckBlockReader.entries[key] = deckBlockReader.lru.PushFront(deckCacheEntry{key, data})
	deckBlockReader.bytes += cap(data)
}
