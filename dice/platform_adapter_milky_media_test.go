//nolint:testpackage
package dice

import (
	"testing"

	"sealdice-core/message"
)

func TestMilkyLocalFileURI(t *testing.T) {
	tests := []struct{ uri, os, want string }{
		{"file:///E:/dice/%E5%9B%BE%20%231.png", "windows", "file://E:/dice/%E5%9B%BE%20%231.png"},
		{"file://localhost/E:/dice/a.png", "windows", "file://E:/dice/a.png"},
		{"file://E:/dice/a.png", "windows", "file://E:/dice/a.png"},
		{"file://server/share/a.png", "windows", "file:////server/share/a.png"},
		{"file:///opt/dice/a.png", "linux", "file:///opt/dice/a.png"},
		{"file://localhost/Users/dice/a.png", "darwin", "file:///Users/dice/a.png"},
		{"file:///E:/dice/a.png", "linux", "file:///E:/dice/a.png"},
		{"file://server/share/a.png", "linux", ""},
		{"file:///a.png?x=1", "linux", ""},
	}
	for _, test := range tests {
		got, err := milkyLocalFileURI(test.uri, test.os)
		if got != test.want || (err != nil) != (test.want == "") {
			t.Errorf("milkyLocalFileURI(%q, %s) = %q, %v; want %q", test.uri, test.os, got, err, test.want)
		}
	}
}

func TestMilkyImageURI(t *testing.T) {
	for _, uri := range []string{"https://example.com/a.png?a=1&b=2", "base64://aGVsbG8=", "data:image/png;base64,aGVsbG8="} {
		if got, err := milkyImageURI(&message.ImageElement{File: &message.FileElement{URL: uri}}); err != nil || got != uri {
			t.Errorf("milkyImageURI(%q) = %q, %v; want unchanged", uri, got, err)
		}
	}
	for _, image := range []*message.ImageElement{nil, {}, {File: &message.FileElement{}}} {
		if got, err := milkyImageURI(image); err == nil || got != "" {
			t.Errorf("milkyImageURI(%#v) = %q, %v; want an error", image, got, err)
		}
	}
}
