package customreply_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"

	"sealdice-core/api/v2/customreply"
	"sealdice-core/api/v2/middleware"
	"sealdice-core/dice"
)

func TestSaveConfigHTTPAcceptsLargeBodyAndEnforcesAuthorization(t *testing.T) {
	svc := newTestService(t)
	e := echo.New()
	api := humaecho.New(e, huma.DefaultConfig("test", "1"))
	grp := huma.NewGroup(api, "/reply")
	grp.UseMiddleware(middleware.WriteProtectedMiddleware(api, svc.Dice()))
	svc.RegisterProtectedRoutes(grp)
	svc.Dice().Parent.AccessTokens.Store("test-token", true)
	body, err := json.Marshal(customreply.SaveBody{ReplyConfig: dice.ReplyConfig{
		Enable: true, Filename: "reply.yaml", Desc: strings.Repeat("x", 2<<20),
		Author: []string{}, Items: []*dice.ReplyItem{}, Conditions: dice.ReplyConditions{},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, token string
		demo        bool
		status      int
	}{
		{"unauthorized", "", false, http.StatusUnauthorized},
		{"demo", "test-token", true, http.StatusForbidden},
		{"large-save", "test-token", false, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc.Dice().Parent.JustForTest = tc.demo
			req := httptest.NewRequest(http.MethodPut, "/reply/files/reply.yaml", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
	stored, err := dice.CustomReplyConfigRead(svc.Dice(), "reply.yaml")
	if err != nil || len(stored.Desc) != 2<<20 {
		t.Fatalf("large reply not persisted: %v", err)
	}
}
