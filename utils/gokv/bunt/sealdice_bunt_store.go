package bunt

import (
	"errors"

	"github.com/philippgille/gokv/encoding"
	"github.com/philippgille/gokv/util"
	"github.com/tidwall/buntdb"
)

// Store is a gokv.Store implementation for bbolt (formerly known as Bolt / Bolt DB).
type Store struct {
	db         *buntdb.DB
	bucketName string
	codec      encoding.Codec
}

// Set stores the given value for the given key.
// Values are automatically marshalled to JSON or gob (depending on the configuration).
// The key must not be "" and the value must not be nil.
func (s Store) Set(k string, v any) error {
	if err := util.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	// First turn the passed object into something that bbolt can handle
	data, err := s.codec.Marshal(v)
	if err != nil {
		return err
	}

	err = s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err2 := tx.Set(k, string(data), nil)
		return err2
	})
	return err
}

// Get retrieves the stored value for the given key.
// You need to pass a pointer to the value, so in case of a struct
// the automatic unmarshalling can populate the fields of the object
// that v points to with the values of the retrieved object's values.
// If no value is found it returns (false, nil).
// The key must not be "" and the pointer must not be nil.
func (s Store) Get(k string, v any) (found bool, err error) {
	if err = util.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}

	var rawData string //[]byte
	err = s.db.View(func(tx *buntdb.Tx) error {
		rawData, err = tx.Get(k)
		return err
	})
	// 特判找不到的情况
	if err != nil && errors.Is(err, buntdb.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, s.codec.Unmarshal([]byte(rawData), v)
}

// Delete deletes the stored value for the given key.
// Deleting a non-existing key-value pair does NOT lead to an error.
// The key must not be "".
func (s Store) Delete(k string) error {
	if err := util.CheckKey(k); err != nil {
		return err
	}

	// 使用 BuntDB 删除键
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(k)
		return err
	})
}

// Close closes the store.
// It must be called to make sure that all open transactions finish and to release all DB resources.
func (s Store) Close() error {
	return s.db.Close()
}

// Clear 清理插件存储
func (s Store) Clear() error {
	// 清理掉所有某个插件的数据
	return ClearAllDataWithCompact(s.db)
}

func ClearAllDataWithCompact(db *buntdb.DB) error {
	// 清空所有数据
	err := db.Update(func(tx *buntdb.Tx) error {
		var keysToDelete []string

		// 遍历所有键
		err := tx.AscendKeys("*", func(key, value string) bool {
			keysToDelete = append(keysToDelete, key)
			return true // 继续迭代
		})
		if err != nil {
			return err
		}

		// 删除所有键
		for _, key := range keysToDelete {
			_, err = tx.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Options are the options for the bbolt store.
type Options struct {
	// Path of the DB file. Must have
	// Optional ("bbolt.db" by default).
	Path string
	// Encoding format.
	// Optional (encoding.JSON by default).
	Codec encoding.Codec
}

// DefaultOptions is an Options object with default values.
// BucketName: "default", Path: "bbolt.db", Codec: encoding.JSON
var DefaultOptions = Options{
	Codec: encoding.JSON,
}

// NewStore creates a new bbolt store.
// Note: bbolt uses an exclusive write lock on the database file so it cannot be shared by multiple processes.
// So when creating multiple clients you should always use a new database file (by setting a different Path in the options).
//
// You must call the Close() method on the store when you're done working with it.
func NewStore(options Options) (Store, error) {
	result := Store{}

	if options.Path == "" {
		return result, errors.New("path is nil, you must write it before")
	}
	if options.Codec == nil {
		options.Codec = DefaultOptions.Codec
	}

	// Open DB
	db, err := buntdb.Open(options.Path)
	if err != nil {
		return result, err
	}

	result.db = db
	result.codec = options.Codec

	return result, nil
}
