package api //nolint:testpackage

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"sealdice-core/dice"
)

func TestCustomReplySavePackageReplyWritesCacheOnly(t *testing.T) {
	testDice, pm, token := newReplyAPITestPackageManager(t)
	const pkgID = "alice/reply-pack"
	archive := createReplyAPITestSealPkg(t, pkgID, "1.0.0", map[string]string{
		"reply/main.yaml": "enable: true\nname: original\nitems: []\n",
	})
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	dice.ReplyReload(testDice)

	body := `{"filename":"main.yaml","packageId":"alice/reply-pack","enable":true,"name":"edited","items":[],"conditions":[]}`
	rec := performReplyAPIRequest(t, http.MethodPost, "/sd-api/configs/custom_reply/save", body, token, customReplySave)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp customReplySaveResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}
	if !resp.Success || resp.Warning == "" {
		t.Fatalf("response = %#v, want success with warning", resp)
	}

	pkg, _ := pm.Get(pkgID)
	cacheReplyPath := filepath.Join(pkg.InstallPath, "reply", "main.yaml")
	cacheData, err := os.ReadFile(cacheReplyPath)
	if err != nil {
		t.Fatalf("ReadFile(cache reply) error = %v", err)
	}
	if !strings.Contains(string(cacheData), "name: edited") {
		t.Fatalf("cache reply was not updated: %s", string(cacheData))
	}
	userReplyPath := testDice.GetExtConfigFilePath("reply", "main.yaml")
	if _, err := os.Stat(userReplyPath); !os.IsNotExist(err) {
		t.Fatalf("package reply should not be copied to user dir, stat err = %v", err)
	}
}

func TestCustomReplyListDistinguishesLocalAndPackageSameName(t *testing.T) {
	testDice, pm, token := newReplyAPITestPackageManager(t)
	const pkgID = "alice/reply-pack"
	archive := createReplyAPITestSealPkg(t, pkgID, "1.0.0", map[string]string{
		"reply/main.yaml": "enable: true\nname: package\nitems: []\n",
	})
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	local := dice.CustomReplyConfigNew(testDice, "main.yaml")
	if local == nil {
		t.Fatal("expected local reply to be created")
	}
	dice.ReplyReload(testDice)

	rec := performReplyAPIRequest(t, http.MethodGet, "/sd-api/configs/custom_reply/file_list", "", token, customReplyFileList)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []ReplyConfigInfo `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	keys := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.Filename == "main.yaml" {
			keys = append(keys, item.PackageID)
		}
	}
	sort.Strings(keys)
	want := []string{"", pkgID}
	if strings.Join(keys, "|") != strings.Join(want, "|") {
		t.Fatalf("main.yaml package IDs = %#v, want %#v; items=%#v", keys, want, resp.Items)
	}
}

func TestCustomReplyNewAllowsLocalFileWithPackageSameName(t *testing.T) {
	testDice, pm, token := newReplyAPITestPackageManager(t)
	const pkgID = "alice/reply-pack"
	archive := createReplyAPITestSealPkg(t, pkgID, "1.0.0", map[string]string{
		"reply/main.yaml": "enable: true\nname: package\nitems: []\n",
	})
	if err := pm.Install(archive); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := pm.Enable(pkgID); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	dice.ReplyReload(testDice)

	rec := performReplyAPIRequest(t, http.MethodPost, "/sd-api/configs/custom_reply/file_new", `{"filename":"main.yaml"}`, token, customReplyFileNew)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}
	if !resp.Success {
		t.Fatalf("response = %#v, want success", resp)
	}
	if _, err := os.Stat(testDice.GetExtConfigFilePath("reply", "main.yaml")); err != nil {
		t.Fatalf("expected local main.yaml to be created: %v", err)
	}
}

func newReplyAPITestPackageManager(t *testing.T) (*dice.Dice, *dice.PackageManager, string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: "."},
		Logger:     zap.NewNop().Sugar(),
	}
	pm := dice.NewPackageManager(testDice)
	testDice.PackageManager = pm
	if err := pm.Init(); err != nil {
		t.Fatalf("PackageManager.Init() error = %v", err)
	}

	token := "test-token"
	manager := &dice.DiceManager{Dice: []*dice.Dice{testDice}}
	manager.AccessTokens.Store(token, true)
	testDice.Parent = manager
	myDice = testDice
	dm = manager
	t.Cleanup(func() {
		myDice = nil
		dm = nil
	})
	return testDice, pm, token
}

func performReplyAPIRequest(t *testing.T, method, target, body, token string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Token", token)
	rec := httptest.NewRecorder()
	if err := handler(e.NewContext(req, rec)); err != nil {
		t.Fatalf("handler error = %v", err)
	}
	return rec
}

func createReplyAPITestSealPkg(t *testing.T, pkgID, version string, files map[string]string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), strings.ReplaceAll(pkgID, "/", "-")+"-"+version+".sealpkg")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", archivePath, err)
	}
	defer archiveFile.Close()

	zipWriter := zip.NewWriter(archiveFile)
	if err := zipAddFile(zipWriter, "info.toml", buildReplyAPITestManifest(pkgID, version)); err != nil {
		t.Fatalf("Write(info.toml) error = %v", err)
	}
	for name, body := range files {
		if err := zipAddFile(zipWriter, name, body); err != nil {
			t.Fatalf("Write(%s) error = %v", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close(zip) error = %v", err)
	}
	return archivePath
}

func buildReplyAPITestManifest(pkgID, version string) string {
	return `[package]
id = "` + pkgID + `"
name = "Reply API Test"
version = "` + version + `"
authors = ["Tester"]
description = "test"

[contents]
reply = ["reply/*.yaml"]
`
}

func zipAddFile(zw *zip.Writer, name, body string) error {
	writer, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(body))
	return err
}
