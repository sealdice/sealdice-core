package dice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/cespare/xxhash/v2"

	"sealdice-core/dice/service"
	engine "sealdice-core/utils/dboperator/engine"
)

const minLogAliasKeyLength = 8

type logNameAliasEntry struct {
	Name     string
	FullHash string
	Key      string
}

type logNameAliasIndex struct {
	entries []logNameAliasEntry
	byKey   map[string]string
	byName  map[string]string
}

func buildLogNameAliasIndex(operator engine.DatabaseOperator, groupID string) (*logNameAliasIndex, error) {
	names, err := service.LogGetList(operator, groupID)
	if err != nil {
		return nil, err
	}

	entries, err := buildLogNameAliasEntries(names, minLogAliasKeyLength)
	if err != nil {
		return nil, err
	}

	index := &logNameAliasIndex{
		entries: entries,
		byKey:   map[string]string{},
		byName:  map[string]string{},
	}
	for _, entry := range entries {
		index.byKey[entry.Key] = entry.Name
		index.byName[entry.Name] = entry.Name
	}
	return index, nil
}

func buildLogNameAliasEntries(names []string, minPrefixLen int) ([]logNameAliasEntry, error) {
	if minPrefixLen < 1 {
		minPrefixLen = 1
	}

	entries := make([]logNameAliasEntry, len(names))
	keyLens := make([]int, len(names))
	for i, name := range names {
		fullHash := logNameHashHex(name)
		if minPrefixLen > len(fullHash) {
			keyLens[i] = len(fullHash)
		} else {
			keyLens[i] = minPrefixLen
		}
		entries[i] = logNameAliasEntry{
			Name:     name,
			FullHash: fullHash,
		}
	}

	for {
		counts := map[string]int{}
		for i, entry := range entries {
			counts[entry.FullHash[:keyLens[i]]]++
		}

		changed := false
		for i, entry := range entries {
			key := entry.FullHash[:keyLens[i]]
			if counts[key] <= 1 {
				continue
			}
			if keyLens[i] >= len(entry.FullHash) {
				return nil, errors.New("日志 hash 键冲突，请改用原始名称")
			}
			keyLens[i]++
			changed = true
		}

		if !changed {
			break
		}
	}

	for i, entry := range entries {
		entry.Key = entry.FullHash[:keyLens[i]]
		entries[i] = entry
	}
	return entries, nil
}

func logNameHashHex(name string) string {
	return fmt.Sprintf("%016x", xxhash.Sum64String(name))
}

func (index *logNameAliasIndex) Resolve(input string) (string, bool) {
	if input == "" {
		return "", false
	}
	if name, exists := index.byName[input]; exists {
		return name, true
	}
	if name, exists := index.byKey[strings.ToLower(input)]; exists {
		return name, true
	}
	return "", false
}

func resolveLogNameForGroup(operator engine.DatabaseOperator, groupID, input string) (string, error) {
	if input == "" {
		return "", nil
	}
	index, err := buildLogNameAliasIndex(operator, groupID)
	if err != nil {
		return "", err
	}
	if resolved, ok := index.Resolve(input); ok {
		return resolved, nil
	}
	return input, nil
}

func formatLogNameForDisplay(name string) string {
	if strings.TrimSpace(name) != name {
		return strconv.Quote(name)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return strconv.Quote(name)
		}
	}
	return name
}

func formatLogNameListLine(entry logNameAliasEntry) string {
	return fmt.Sprintf("- 【%s】: %s", entry.Key, formatLogNameForDisplay(entry.Name))
}
