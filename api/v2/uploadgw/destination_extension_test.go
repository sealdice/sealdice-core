package uploadgw //nolint:testpackage // 复用包内网关夹具验证扩展包安装边界。

import (
	"net/http"
	"testing"
)

func TestExtensionInstallAcceptsCreate(t *testing.T) {
	_, srv := newRealGateway(t, minimalDM(t))

	resp := createOnly(t, srv, 4, map[string]string{
		MetaModule:   "extension-install",
		MetaFilename: "pkg.zip",
		MetaSHA256:   sumHex([]byte("x")),
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
}

func TestExtensionInstallRejectsEmptyFilename(t *testing.T) {
	_, srv := newRealGateway(t, minimalDM(t))

	resp := createOnly(t, srv, 1, map[string]string{
		MetaModule:   "extension-install",
		MetaFilename: "",
		MetaSHA256:   sumHex([]byte("x")),
	})
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("create status = %d, want >= 400 for empty filename", resp.StatusCode)
	}
}

func TestExtensionInstallWithoutPackageManagerErrors(t *testing.T) {
	dm := minimalDM(t) // PackageManager 为 nil
	_, srv := newRealGateway(t, dm)

	content := []byte("zipbytes")
	resp := createUpload(t, srv, content, map[string]string{
		MetaModule:   "extension-install",
		MetaFilename: "pkg.zip",
		MetaSHA256:   sumHex(content),
	})
	if resp.StatusCode < 400 {
		t.Fatalf("patch status = %d, want >= 400 when package manager unavailable", resp.StatusCode)
	}
}
