// upgrade/store/jsonstore.go
package store

import (
	"encoding/json"
	"os"
	"sync"

	upgrade "sealdice-core/utils/upgrader"
)

type JSONStore struct {
	Path   string
	mutex  sync.Mutex
	data   []upgrade.UpgradeRecord
	loaded bool
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{Path: path}
}

func (js *JSONStore) load() error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	if js.loaded {
		return nil
	}

	f, err := os.Open(js.Path)
	if os.IsNotExist(err) {
		js.data = []upgrade.UpgradeRecord{}
		js.loaded = true
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&js.data)
	if err == nil {
		js.loaded = true
	}
	return err
}

func (js *JSONStore) save() error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	f, err := os.Create(js.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(js.data)
}

func (js *JSONStore) IsApplied(id string) (bool, error) {
	if err := js.load(); err != nil {
		return false, err
	}
	for _, rec := range js.data {
		if rec.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func (js *JSONStore) SaveRecord(rec upgrade.UpgradeRecord) error {
	if err := js.load(); err != nil {
		return err
	}
	js.data = append(js.data, rec)
	return js.save()
}

func (js *JSONStore) LoadRecords() ([]upgrade.UpgradeRecord, error) {
	if err := js.load(); err != nil {
		return nil, err
	}
	return js.data, nil
}
