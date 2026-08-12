package static_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	staticfs "sealdice-core/static"
)

func TestV2UIEmbedIncludesLeadingUnderscoreAssets(t *testing.T) {
	const diskAssetsDir = "v2ui/dist/assets"

	entries, err := os.ReadDir(filepath.FromSlash(diskAssetsDir))
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v", diskAssetsDir, err)
	}

	var checked int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "_") {
			continue
		}

		checked++
		embeddedPath := pathJoinSlash(diskAssetsDir, entry.Name())
		if _, err := fs.Stat(staticfs.V2UI, embeddedPath); err != nil {
			t.Fatalf("fs.Stat(V2UI, %q) error = %v", embeddedPath, err)
		}
	}

	if checked == 0 {
		t.Skip("no leading-underscore assets in v2ui dist")
	}
}

func pathJoinSlash(elem ...string) string {
	return strings.Join(elem, "/")
}
