//go:build ignore

// Generate the upgrade fixture using the exact Bluge version used before v0.8.0.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/fy0/bluge"
)

const legacyVersion = "v0.0.0-20260713113642-38c63a5ebb28"

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("missing build information")
	}
	matched := false
	for _, dep := range info.Deps {
		if dep.Path == "github.com/fy0/bluge" && dep.Version == legacyVersion && dep.Replace == nil {
			matched = true
		}
	}
	if !matched {
		return fmt.Errorf("run with github.com/fy0/bluge@%s in a separate module", legacyVersion)
	}
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: go run generate.go <new-index-directory>")
	}
	dir := os.Args[1]
	if err := os.Mkdir(dir, 0o755); err != nil {
		return err
	}
	writer, err := bluge.OpenWriter(bluge.DefaultConfig(dir))
	if err != nil {
		return err
	}
	for _, item := range []struct{ id, from, title, content string }{
		{"legacy-help", "data/helpdoc/upgrade.json", "旧缓存重建验证", "升级后帮助内容仍可检索"},
		{"legacy-removed", "data/helpdoc/removed.json", "已删除旧条目", "此条目不应在重建后保留"},
	} {
		doc := bluge.NewDocument(item.id).
			AddField(bluge.NewStoredOnlyField("group", []byte("default"))).
			AddField(bluge.NewKeywordField("_group_exact", "default")).
			AddField(bluge.NewKeywordField("from", item.from).StoreValue()).
			AddField(bluge.NewTextField("title", item.title).StoreValue().SearchTermPositions()).
			AddField(bluge.NewTextField("content", item.content).StoreValue().SearchTermPositions()).
			AddField(bluge.NewKeywordField("package", "upgrade").StoreValue())
		if err := writer.Insert(doc); err != nil {
			_ = writer.Close()
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "schema_version"), []byte("2"), 0o600); err != nil {
		return err
	}
	reader, err := bluge.OpenReader(bluge.DefaultConfig(dir))
	if err != nil {
		return err
	}
	count, countErr := reader.Count()
	closeErr := reader.Close()
	if countErr != nil {
		return countErr
	}
	if closeErr != nil {
		return closeErr
	}
	if count != 2 {
		return fmt.Errorf("persisted document count = %d, want 2", count)
	}
	fmt.Printf("Generated and reopened %d documents with Bluge %s\n", count, legacyVersion)
	return nil
}
