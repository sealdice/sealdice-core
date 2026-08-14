package static_test

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	staticfs "sealdice-core/static"
)

func TestV2UIEmbedIncludesLeadingUnderscoreAssets(t *testing.T) {
	const diskAssetsDir = "v2ui/dist/assets"

	entries, err := os.ReadDir(filepath.FromSlash(diskAssetsDir))
	if errors.Is(err, fs.ErrNotExist) {
		// dist 未纳入版本管理，只有本地构建过新 UI 才存在。
		t.Skipf("v2ui dist not built: %s", diskAssetsDir)
	}
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v", diskAssetsDir, err)
	}

	var checked int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "_") {
			continue
		}

		checked++
		embeddedPath := path.Join(diskAssetsDir, entry.Name())
		if _, err := fs.Stat(staticfs.V2UI, embeddedPath); err != nil {
			t.Fatalf("fs.Stat(V2UI, %q) error = %v", embeddedPath, err)
		}
	}

	if checked == 0 {
		t.Skip("no leading-underscore assets in v2ui dist")
	}
}
