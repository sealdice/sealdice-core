package dice

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"

	"github.com/klauspost/compress/zstd"
	"go.etcd.io/bbolt"
)

type HelpDocMap[K comparable, V any] struct {
	db *bbolt.DB
}

// TODO: 其实也没什么通用的必要……
func NewHelpDocMap[K comparable, V any](dbPath string) (*HelpDocMap[K, V], error) {
	// 使用反射获取具体类型
	kType := reflect.TypeOf((*K)(nil)).Elem()
	vType := reflect.TypeOf((*V)(nil)).Elem()

	// 尝试注册类型
	gob.Register(reflect.New(kType).Elem().Interface())
	gob.Register(reflect.New(vType).Elem().Interface())

	options := &bbolt.Options{
		NoGrowSync:      false,
		NoFreelistSync:  true,                  // 禁用自由列表同步
		PreLoadFreelist: true,                  // 预加载自由页
		FreelistType:    bbolt.FreelistMapType, // 使用 HashMap 类型的自由列表
		InitialMmapSize: 10 * 1024 * 1024,      // 10M 初始内存应该足够了，都够一本斗破苍穹了……
		NoSync:          true,                  // 禁用同步操作
	}

	// 打开或创建数据库
	db, err := bbolt.Open(dbPath, 0666, options)
	if err != nil {
		return nil, err
	}

	// 确保创建默认桶
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("default"))
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &HelpDocMap[K, V]{db: db}, nil
}

func (h *HelpDocMap[K, V]) encode(value V) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}

	// 使用 zstd 编码数据
	zEnc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	defer zEnc.Close()

	compressed := zEnc.EncodeAll(buf.Bytes(), make([]byte, 0, len(buf.Bytes())))

	return compressed, nil
}

func (h *HelpDocMap[K, V]) decode(data []byte) (V, error) {
	var value V

	// 使用 zstd 解码数据
	// 解包的时候多多的用内存，反正用完就抛了
	zDec, err := zstd.NewReader(nil, zstd.WithDecoderLowmem(false))
	if err != nil {
		return value, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer zDec.Close()

	decompressed, err := zDec.DecodeAll(data, nil)
	if err != nil {
		return value, fmt.Errorf("failed to decompress data: %w", err)
	}

	dec := gob.NewDecoder(bytes.NewReader(decompressed))
	if err := dec.Decode(&value); err != nil {
		return value, fmt.Errorf("failed to decode value: %w", err)
	}

	return value, nil
}

func (h *HelpDocMap[K, V]) Store(key K, value V) error {
	return h.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("default"))
		if bucket == nil {
			return errors.New("bucket not found")
		}

		// 编码键和值
		encodedValue, err := h.encode(value)
		if err != nil {
			return err
		}

		// 存储键值对
		keyBytes, err := h.encodeKey(key)
		if err != nil {
			return err
		}

		return bucket.Put(keyBytes, encodedValue)
	})
}

func (h *HelpDocMap[K, V]) encodeKey(key K) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	wrapper := struct {
		Key K
	}{
		Key: key,
	}

	if err := enc.Encode(wrapper); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (h *HelpDocMap[K, V]) decodeKey(data []byte) (K, error) {
	var zeroKey K
	wrapper := struct {
		Key K
	}{
		Key: zeroKey,
	}

	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&wrapper); err != nil {
		return zeroKey, err
	}

	return wrapper.Key, nil
}

func (h *HelpDocMap[K, V]) Load(key K) (V, bool) {
	var zeroValue V

	var result V
	err := h.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("default"))
		if bucket == nil {
			return errors.New("bucket not found")
		}

		// 编码键
		keyBytes, err := h.encodeKey(key)
		if err != nil {
			return err
		}

		valueData := bucket.Get(keyBytes)
		if valueData == nil {
			return nil // Not found
		}

		// 解码值
		decodedValue, err := h.decode(valueData)
		if err != nil {
			return err
		}
		result = decodedValue
		return nil
	})

	if err != nil {
		return zeroValue, false
	}

	return result, true
}

func (h *HelpDocMap[K, V]) Range(f func(key K, value V) bool) error {
	return h.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("default"))
		if bucket == nil {
			return errors.New("bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			// 解码键
			key, err := h.decodeKey(k)
			if err != nil {
				return err
			}

			// 解码值
			value, err := h.decode(v)
			if err != nil {
				return err
			}

			if !f(key, value) {
				return nil
			}
			return nil
		})
	})
}

func (h *HelpDocMap[K, V]) Exists(key K) bool {
	_, exists := h.Load(key)
	return exists
}

func (h *HelpDocMap[K, V]) Len() int {
	count := 0
	err := h.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("default"))
		if bucket == nil {
			return errors.New("bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})
	if err != nil {
		return 0
	}
	return count
}

func (h *HelpDocMap[K, V]) Delete(key K) error {
	return h.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("default"))
		if bucket == nil {
			return errors.New("bucket not found")
		}

		return bucket.Delete([]byte(fmt.Sprintf("%v", key)))
	})
}

func (h *HelpDocMap[K, V]) Close() error {
	return h.db.Close()
}
