//nolint:testpackage
package dice

import (
	"path/filepath"
	"testing"
)

func TestIsResourceCodeRecognizesAllAliases(t *testing.T) {
	resourceCodes := []string{
		"[图:./card.png]",
		"[img:./card.png]",
		"[文本:./card.txt]",
		"[text:./card.txt]",
		"[语音:./voice.wav]",
		"[voice:./voice.wav]",
		"[视频:./video.mp4]",
		"[video:./video.mp4]",
	}
	for _, code := range resourceCodes {
		if !isResourceCode(code) {
			t.Fatalf("resource code %q was treated as a deck expression", code)
		}
	}
	if isResourceCode("[1d20]") {
		t.Fatal("dice expression was treated as a resource code")
	}
}

func TestRewriteRelativeResourcePathsUsesDeckSourceDirectory(t *testing.T) {
	packageRoot := filepath.Join(t.TempDir(), "cache", "packages", "author", "package")
	deckFilename := filepath.Join(packageRoot, "decks", "main.json")
	tests := []struct {
		name     string
		alias    string
		resource string
		want     string
	}{
		{name: "image", alias: "图", resource: "./../assets/card.png", want: filepath.Join(packageRoot, "assets", "card.png")},
		{name: "text", alias: "text", resource: "./../assets/card.txt", want: filepath.Join(packageRoot, "assets", "card.txt")},
		{name: "voice", alias: "语音", resource: `.\..\assets\voice.wav`, want: filepath.Join(packageRoot, "assets", "voice.wav")},
		{name: "video", alias: "video", resource: "./../assets/video.mp4", want: filepath.Join(packageRoot, "assets", "video.mp4")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := rewriteRelativeResourcePaths(deckFilename, "["+test.alias+":"+test.resource+"]")
			wantPath, err := filepath.Abs(test.want)
			if err != nil {
				t.Fatal(err)
			}
			want := "[" + test.alias + ":" + filepath.ToSlash(wantPath) + "]"
			if got != want {
				t.Fatalf("rewriteRelativeResourcePaths() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteRelativeResourcePathsHandlesCQAndCrossPackageReferences(t *testing.T) {
	packagesRoot := filepath.Join(t.TempDir(), "cache", "packages")
	deckFilename := filepath.Join(packagesRoot, "author", "source", "decks", "main.json")
	rewritten := rewriteRelativeResourcePaths(
		deckFilename,
		"[CQ:video,file=./../../../other/target/assets/video.mp4,cache=0]",
	)
	cq := CQParse(rewritten)
	if cq.Type != "video" || cq.Args["cache"] != "0" {
		t.Fatalf("CQ parameters were not preserved: %#v", cq)
	}
	want, err := filepath.Abs(filepath.Join(packagesRoot, "other", "target", "assets", "video.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if cq.Args["file"] != filepath.ToSlash(want) {
		t.Fatalf("CQ file path = %q, want %q", cq.Args["file"], filepath.ToSlash(want))
	}
}

func TestResolveRelativeResourcePathLeavesNonRelativeReferencesUnchanged(t *testing.T) {
	deckFilename := filepath.Join(t.TempDir(), "decks", "main.json")
	for _, resource := range []string{
		"assets/card.png",
		"https://example.com/card.png",
		"base64://Y2FyZA==",
		"opaque-resource-id",
	} {
		if got := resolveRelativeResourcePath(deckFilename, resource); got != resource {
			t.Fatalf("resolveRelativeResourcePath(%q) = %q, want unchanged", resource, got)
		}
	}
}
