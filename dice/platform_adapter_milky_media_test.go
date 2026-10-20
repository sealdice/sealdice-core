//nolint:testpackage
package dice

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"sealdice-core/message"
)

func newMilkyLocalImage(t *testing.T) (string, *message.FileElement) {
	t.Helper()
	// Keep the fixture on the working directory's drive for relative-path tests.
	dir, err := os.MkdirTemp(".", "milky-image-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	filename := filepath.Join(dir, "本地 图片,#&.png")
	if err := os.WriteFile(filename, []byte("local image test data"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := message.FilepathToFileElement(filename)
	if err != nil {
		t.Fatal(err)
	}
	return filename, file
}

func expectedMilkyLocalImageURI(uri string) string {
	if runtime.GOOS == "windows" {
		return strings.Replace(uri, "file:///", "file://", 1)
	}
	return uri
}

func TestMilkyLocalFileURIPlatforms(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		os   string
		want string
	}{
		{name: "Windows drive", uri: "file:///E:/dice/data/image.png", os: "windows", want: "file://E:/dice/data/image.png"},
		{name: "lowercase drive", uri: "file:///e:/dice/image.png", os: "windows", want: "file://e:/dice/image.png"},
		{name: "drive root", uri: "file:///C:/", os: "windows", want: "file://C:/"},
		{name: "escaped filename", uri: "file:///E:/%E5%9B%BE%20%231%25%2C.png", os: "windows", want: "file://E:/%E5%9B%BE%20%231%25%2C.png"},
		{name: "localhost drive", uri: "file://localhost/E:/dice/image.png", os: "windows", want: "file://E:/dice/image.png"},
		{name: "already adapted", uri: "file://E:/dice/image.png", os: "windows", want: "file://E:/dice/image.png"},
		{name: "UNC authority", uri: "file://server/share/image.png", os: "windows", want: "file://server/share/image.png"},
		{name: "UNC path", uri: "file:////server/share/image.png", os: "windows", want: "file:////server/share/image.png"},
		{name: "no drive", uri: "file:///images/image.png", os: "windows", want: "file:///images/image.png"},
		{name: "not a drive letter", uri: "file:///1:/images/image.png", os: "windows", want: "file:///1:/images/image.png"},
		{name: "drive-relative path", uri: "file:///E:image.png", os: "windows", want: "file:///E:image.png"},
		{name: "Linux absolute", uri: "file:///opt/dice/image.png", os: "linux", want: "file:///opt/dice/image.png"},
		{name: "macOS absolute", uri: "file:///Users/dice/image.png", os: "darwin", want: "file:///Users/dice/image.png"},
		{name: "Linux colon in path", uri: "file:///E:/dice/image.png", os: "linux", want: "file:///E:/dice/image.png"},
		{name: "macOS colon in path", uri: "file:///E:/dice/image.png", os: "darwin", want: "file:///E:/dice/image.png"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := milkyLocalFileURI(test.uri, test.os)
			if err != nil || got != test.want {
				t.Fatalf("local URI = %q, %v; want %q", got, err, test.want)
			}
			again, err := milkyLocalFileURI(got, test.os)
			if err != nil || again != got {
				t.Fatalf("second conversion = %q, %v; want unchanged %q", again, err, got)
			}
		})
	}
}

func TestMilkyImageURILocalResources(t *testing.T) {
	filename, file := newMilkyLocalImage(t)
	absolutePath, err := filepath.Abs(filename)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		image *message.ImageElement
	}{
		{name: "parsed image", image: &message.ImageElement{URL: file.URL, File: file}},
		{name: "file URL fallback", image: &message.ImageElement{File: file}},
		{name: "URL only", image: &message.ImageElement{URL: file.URL}},
		{name: "file without stream", image: &message.ImageElement{File: &message.FileElement{URL: file.URL}}},
		{name: "relative path", image: &message.ImageElement{URL: filename}},
		{name: "absolute path", image: &message.ImageElement{URL: absolutePath}},
		{name: "file path fallback", image: &message.ImageElement{File: &message.FileElement{File: filename}}},
	}
	want := expectedMilkyLocalImageURI(file.URL)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalURL := test.image.URL
			uri, err := milkyImageURI(test.image)
			if err != nil || uri != want {
				t.Fatalf("image URI = %q, %v; want %q", uri, err, want)
			}
			if test.image.URL != originalURL {
				t.Errorf("original image URL changed from %q to %q", originalURL, test.image.URL)
			}
		})
	}
}

func TestMilkyImageURIFromCQAndSealCode(t *testing.T) {
	filename, file := newMilkyLocalImage(t)
	inputs := []string{
		"[图:" + filename + "]",
		"[img:" + filename + "]",
		"[CQ:image,file=" + message.EscapeCQParam(filename) + "]",
		"[CQ:image,file=" + message.EscapeCQParam(file.URL) + "]",
		"[CQ:image,url=" + message.EscapeCQParam(file.URL) + "]",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			elements := message.ConvertStringMessage(input)
			if len(elements) != 1 {
				t.Fatalf("parsed %d elements, want one image", len(elements))
			}
			image, ok := elements[0].(*message.ImageElement)
			if !ok {
				t.Fatalf("parsed %T, want *message.ImageElement", elements[0])
			}
			uri, err := milkyImageURI(image)
			want := expectedMilkyLocalImageURI(file.URL)
			if err != nil || uri != want {
				t.Fatalf("image URI = %q, %v; want %q", uri, err, want)
			}
		})
	}
}

type milkyImageUnreadableStream struct{}

func (milkyImageUnreadableStream) Read([]byte) (int, error) {
	return 0, errors.New("image stream must not be read during URI conversion")
}

func TestMilkyImageURIDoesNotReadImageData(t *testing.T) {
	filename, file := newMilkyLocalImage(t)
	file.Stream = milkyImageUnreadableStream{}
	stream := file.Stream
	// Only the path is converted. No file is reopened, and no content is encoded.
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	image := &message.ImageElement{URL: file.URL, File: file}
	for range 2 {
		uri, err := milkyImageURI(image)
		want := expectedMilkyLocalImageURI(file.URL)
		if err != nil || uri != want {
			t.Fatalf("image URI = %q, %v; want %q", uri, err, want)
		}
	}
	if file.Stream != stream {
		t.Fatal("image stream was replaced")
	}
}

func TestMilkyImageURIPreservesRemoteResources(t *testing.T) {
	for _, uri := range []string{
		"http://example.com/image.png",
		"https://example.com/image.png?a=1&b=2,3",
		"base64://aGVsbG8=",
	} {
		t.Run(uri, func(t *testing.T) {
			for _, image := range []*message.ImageElement{
				{URL: uri},
				{File: &message.FileElement{URL: uri}},
				{File: &message.FileElement{File: uri}},
				{URL: uri, File: &message.FileElement{URL: "file:///unused.png", Stream: milkyImageUnreadableStream{}}},
			} {
				got, err := milkyImageURI(image)
				if err != nil || got != uri {
					t.Fatalf("image URI = %q, %v; want unchanged %q", got, err, uri)
				}
			}
		})
	}
}

func TestMilkyImageURIRejectsInvalidResources(t *testing.T) {
	filename, _ := newMilkyLocalImage(t)
	tests := []struct {
		name  string
		image *message.ImageElement
	}{
		{name: "nil image"},
		{name: "empty image", image: &message.ImageElement{}},
		{name: "empty file", image: &message.ImageElement{File: &message.FileElement{}}},
		{name: "missing bare path", image: &message.ImageElement{URL: filename + ".missing"}},
		{name: "unsupported scheme", image: &message.ImageElement{URL: "files:///missing.png"}},
		{name: "invalid escape", image: &message.ImageElement{URL: "file:///image%xx.png"}},
		{name: "query", image: &message.ImageElement{URL: "file:///image.png?x=1"}},
		{name: "fragment", image: &message.ImageElement{URL: "file:///image.png#part"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if uri, err := milkyImageURI(test.image); err == nil || uri != "" {
				t.Fatalf("image URI = %q, %v; want empty URI and an error", uri, err)
			}
		})
	}
}
