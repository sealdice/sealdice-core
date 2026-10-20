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

	if scheme, _, hasScheme := strings.Cut(uri, "://"); hasScheme {
		switch strings.ToLower(scheme) {
		case "http", "https", "base64":
			return uri, nil
		case "file":
			return milkyLocalFileURI(uri, runtime.GOOS)
		default:
			return "", errors.New("unsupported image URI scheme")
		}
	}

	// For bare paths, reuse the shared path checks without reading image data.
	normalized, err := message.NormalizeLocalResourcePath(uri)
	if err != nil {
		return "", fmt.Errorf("normalize local image path: %w", err)
	}
	if !strings.HasPrefix(strings.ToLower(normalized), "file://") {
		return "", errors.New("image path does not reference an accessible local file")
	}
	return milkyLocalFileURI(normalized, runtime.GOOS)
}

// milkyLocalFileURI adapts Windows drive-letter URIs at the Milky send boundary.
// Yogurt uses the text after file:// directly as a native path, so /C:/... must
// become C:/... on Windows. Keep shared/OneBot URIs standard, preserve Unix and
// UNC paths, and leave percent escapes intact for the protocol server to decode.
func milkyLocalFileURI(uri, goos string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse local image URI: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "file") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid local image URI")
	}
	if goos == "windows" && (parsed.Host == "" || strings.EqualFold(parsed.Host, "localhost")) {
		path := parsed.Path
		if len(path) >= 4 && path[0] == '/' && path[2] == ':' && path[3] == '/' &&
			(path[1] >= 'A' && path[1] <= 'Z' || path[1] >= 'a' && path[1] <= 'z') {
			return "file://" + strings.TrimPrefix(parsed.EscapedPath(), "/"), nil
		}
	}
	return uri, nil
}
