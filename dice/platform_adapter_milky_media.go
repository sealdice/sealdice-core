package dice

import (
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"sealdice-core/message"
)

func milkyImageURI(image *message.ImageElement) (string, error) {
	if image == nil {
		return "", errors.New("nil image element")
	}
	uri := image.URL
	if uri == "" && image.File != nil {
		uri = image.File.URL
		if uri == "" {
			uri = image.File.File
		}
	}
	if uri == "" {
		return "", errors.New("image has no resource URI")
	}

	// Only file resources need adapter-specific rewriting. Preserve other URI
	// schemes (including opaque data: URIs) for the protocol server to handle.
	if resource, err := url.Parse(uri); err == nil && resource.Scheme != "" &&
		(runtime.GOOS != "windows" || len(resource.Scheme) != 1) {
		if strings.EqualFold(resource.Scheme, "file") {
			return milkyLocalFileURI(uri, runtime.GOOS)
		}
		return uri, nil
	}

	// For bare paths, reuse the shared path checks without reading image data.
	normalized, err := message.NormalizeLocalResourcePath(uri)
	if err != nil {
		return "", fmt.Errorf("normalize local image path: %w", err)
	}
	if strings.HasPrefix(strings.ToLower(normalized), "file://") {
		return milkyLocalFileURI(normalized, runtime.GOOS)
	}
	// Opaque references that are not local files were passed through before
	// local-path compatibility was added; do not silently discard them here.
	return uri, nil
}

// milkyLocalFileURI adapts file URIs at the Milky send boundary. Yogurt uses
// the text after file:// directly as a native path: remove localhost authority,
// adapt Windows drive letters and preserve UNC's leading //. Keep shared/OneBot
// URIs standard and percent escapes intact for the protocol server to decode.
func milkyLocalFileURI(uri, goos string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse local image URI: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "file") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid local image URI")
	}
	path := parsed.EscapedPath()
	localhost := strings.EqualFold(parsed.Host, "localhost")
	if goos == "windows" {
		if parsed.Host != "" && !localhost {
			// Already adapted drive-letter URIs parse the drive as authority.
			if len(parsed.Host) == 2 && parsed.Host[1] == ':' && isMilkyDriveLetter(parsed.Host[0]) {
				return uri, nil
			}
			return "file:////" + parsed.Host + path, nil
		}
		if len(parsed.Path) >= 4 && parsed.Path[0] == '/' && parsed.Path[2] == ':' &&
			parsed.Path[3] == '/' && isMilkyDriveLetter(parsed.Path[1]) {
			return "file://" + strings.TrimPrefix(path, "/"), nil
		}
	}
	if localhost {
		return "file://" + path, nil
	}
	if parsed.Host != "" {
		// Yogurt would resolve host/path as a relative path; Unix has no UNC form.
		return "", errors.New("file URI with a remote host is not supported")
	}
	return uri, nil
}

func isMilkyDriveLetter(letter byte) bool {
	return letter >= 'A' && letter <= 'Z' || letter >= 'a' && letter <= 'z'
}
