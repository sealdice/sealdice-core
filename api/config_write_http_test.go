package api //nolint:testpackage // 对照旧处理器与 V2 的实际 HTTP 写入契约。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"

	"sealdice-core/api/v2/customreply"
	"sealdice-core/api/v2/customtext"
	"sealdice-core/dice"
)

func TestReplyWriteHTTPVersionsSharePersistenceBoundary(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		for _, fail := range []bool{false, true} {
			name := version + "/success"
			if fail {
				name = version + "/failure"
			}
			t.Run(name, func(t *testing.T) {
				d := persistenceDice(t)
				old := &dice.ReplyConfig{Filename: "reply.yaml", Desc: "before", VMVersion: dice.ReplyVMVersionV2}
				d.CustomReplyConfig = []*dice.ReplyConfig{old}
				target := d.GetExtConfigFilePath("reply", "reply.yaml")
				if fail {
					if err := os.Mkdir(target, 0o700); err != nil {
						t.Fatal(err)
					}
				} else if err := old.SaveToPath(target); err != nil {
					t.Fatal(err)
				}
				config := dice.ReplyConfig{Filename: "reply.yaml", Enable: true, Desc: "after", Author: []string{}, Items: []*dice.ReplyItem{}, Conditions: dice.ReplyConditions{}}
				body, err := json.Marshal(customreply.SaveBody{ReplyConfig: config})
				if err != nil {
					t.Fatal(err)
				}
				app := echo.New()
				app.POST("/v1", customReplySave)
				api := humaecho.New(app, huma.DefaultConfig("test", "1"))
				customreply.NewService(d.Parent).RegisterProtectedRoutes(huma.NewGroup(api, "/v2"))
				method, url := http.MethodPost, "/v1"
				if version == "v2" {
					method, url = http.MethodPut, "/v2/files/reply.yaml"
				}
				req := httptest.NewRequest(method, url, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Token", "persistence-test")
				rec := httptest.NewRecorder()
				app.ServeHTTP(rec, req)
				want := http.StatusOK
				if fail {
					want = http.StatusInternalServerError
				}
				if rec.Code != want {
					t.Fatalf("status=%d want=%d: %s", rec.Code, want, rec.Body.String())
				}
				current := d.CustomReplyConfig[0]
				if fail {
					if current != old || current.Desc != "before" {
						t.Fatal("failed save changed live rules")
					}
				} else {
					stored, readErr := dice.CustomReplyConfigRead(d, "reply.yaml")
					if readErr != nil || stored.Desc != "after" || current.Desc != "after" || current.VMVersion != dice.ReplyVMVersionV2 {
						t.Fatalf("runtime/disk mismatch: current=%+v stored=%+v err=%v", current, stored, readErr)
					}
				}
			})
		}
	}
}

func TestTextWriteHTTPVersionsPreserveMemoryOnFailure(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			d := persistenceDice(t)
			d.TextMapRaw["test"] = dice.TextTemplateWithWeight{"before": {}}
			if err := os.WriteFile(filepath.Join(d.BaseConfig.DataDir, "configs"), []byte("blocked"), 0o600); err != nil {
				t.Fatal(err)
			}
			app := echo.New()
			app.POST("/v1", customTextSave)
			api := humaecho.New(app, huma.DefaultConfig("test", "1"))
			customtext.NewService(d.Parent).RegisterProtectedRoutes(huma.NewGroup(api, "/v2"))
			method, url, payload := http.MethodPost, "/v1", `{"category":"test","data":{}}`
			if version == "v2" {
				method, url, payload = http.MethodPut, "/v2/test", `{"data":{}}`
			}
			req := httptest.NewRequest(method, url, bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Token", "persistence-test")
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d: %s", rec.Code, rec.Body.String())
			}
			if _, ok := d.TextMapRaw["test"]["before"]; !ok {
				t.Fatal("failed save published new category")
			}
		})
	}
}
