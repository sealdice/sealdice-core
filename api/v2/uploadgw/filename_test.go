package uploadgw

import "testing"

func TestUploadFilenameRejectsDirectoryNames(t *testing.T) {
	for _, name := range []string{"", " ", ".", "..", " .. "} {
		if got := sanitizeFilename(name); got != "" {
			t.Errorf("sanitizeFilename(%q) = %q", name, got)
		}
	}
	if got := sanitizeFilename("folder/回复.yaml"); got != "folder_回复.yaml" {
		t.Fatalf("separator normalization changed: %q", got)
	}
}
