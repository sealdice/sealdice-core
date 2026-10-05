package api //nolint:testpackage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"sealdice-core/dice"
)

const authTestToken = "auth-regression-test-token"

// This independent expectation list must not use the production allowlist.
var authTestPublicRoutes = map[string]bool{
	"GET /sd-api/preInfo":       true,
	"POST /sd-api/signin":       true,
	"GET /sd-api/signin/salt":   true,
	"GET /sd-api/utils/ga/:uid": true,
	"POST /sd-api/force_stop":   true,
}

func newAuthTestServer(t *testing.T) (*echo.Echo, *dice.Dice) {
	t.Helper()
	previousDice, previousDM := myDice, dm
	t.Cleanup(func() { myDice, dm = previousDice, previousDM })

	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: "."},
		Config:     dice.NewConfig(nil),
		Logger:     zap.NewNop().Sugar(),
	}
	manager := &dice.DiceManager{
		Dice:           []*dice.Dice{testDice},
		UIPasswordHash: "configured-test-password",
		UIPasswordSalt: "test-salt",
	}
	manager.AccessTokens.Store(authTestToken, true)
	testDice.Parent = manager
	testDice.Config.BanList = &dice.BanListInfo{
		Parent: testDice,
		Map:    &dice.SyncMap[string, *dice.BanListInfoItem]{},
	}
	e := echo.New()
	Bind(e, manager)
	return e, testDice
}

func authTestRequest(e *echo.Echo, method, target, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set("Token", token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestBanMapAddOneRequiresAuth(t *testing.T) {
	e, testDice := newAuthTestServer(t)
	const id = "QQ:auth-regression"
	for _, tc := range []struct {
		name  string
		query string
		token string
	}{
		{name: "missing token"},
		{name: "invalid header", token: "invalid"},
		{name: "invalid query", query: "?token=invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testDice.Config.BanList.Map.Store(id, &dice.BanListInfoItem{ID: id, Rank: dice.BanRankBanned})
			rec := authTestRequest(e, http.MethodPost, "/sd-api/banconfig/map_add_one"+tc.query,
				`{"ID":"QQ:auth-regression","rank":30}`, tc.token)
			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403; body = %s", rec.Code, rec.Body.String())
			}
			item, _ := testDice.Config.BanList.Map.Load(id)
			if item.Rank != dice.BanRankBanned {
				t.Errorf("unauthenticated request changed ban rank to %d", item.Rank)
			}
		})
	}
	// A valid token must still allow the intended administrative operation.
	rec := authTestRequest(e, http.MethodPost, "/sd-api/banconfig/map_add_one",
		`{"ID":"QQ:auth-regression","rank":30}`, authTestToken)
	item, _ := testDice.Config.BanList.Map.Load(id)
	if rec.Code != http.StatusOK || item.Rank != dice.BanRankTrusted {
		t.Fatalf("authorized trust update failed: status=%d rank=%d", rec.Code, item.Rank)
	}
}

func TestPackageConfigRequiresAuth(t *testing.T) {
	t.Chdir(t.TempDir())
	e, testDice := newAuthTestServer(t)
	pm := dice.NewPackageManager(testDice)
	testDice.PackageManager = pm
	if err := pm.Init(); err != nil {
		t.Fatal(err)
	}
	const pkgID = "alice/auth-test"
	archive := createReplyAPITestSealPack(t, pkgID, "1.0.0", map[string]string{
		"reply/main.yaml": "items: []\n",
	})
	if err := pm.Install(archive); err != nil {
		t.Fatal(err)
	}
	pkg, exists := pm.Get(pkgID)
	if !exists {
		t.Fatal("test package not installed")
	}
	const secret = "test-only-private-config-value"
	pkg.Config = map[string]interface{}{"apiKey": secret}

	for _, target := range []string{
		"/sd-api/package/list",
		"/sd-api/package/_?id=" + pkgID,
		"/sd-api/package/_/config?id=" + pkgID,
		"/sd-api/package/_/config-schema?id=" + pkgID,
	} {
		t.Run(target, func(t *testing.T) {
			for _, token := range []string{"", "invalid"} {
				rec := authTestRequest(e, http.MethodGet, target, "", token)
				if rec.Code != http.StatusForbidden {
					t.Errorf("status = %d, want 403", rec.Code)
				}
				if strings.Contains(rec.Body.String(), secret) {
					t.Error("unauthenticated response exposed private package config")
				}
			}
			rec := authTestRequest(e, http.MethodGet, target, "", authTestToken)
			if rec.Code != http.StatusOK {
				t.Fatalf("authorized status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
	// The package configuration write was already guarded; keep it that way.
	rec := authTestRequest(e, http.MethodPost, "/sd-api/package/_/config?id="+pkgID,
		`{"apiKey":"unauthorized-change"}`, "")
	if rec.Code != http.StatusForbidden || pkg.Config["apiKey"] != secret {
		t.Fatalf("unauthorized package write: status=%d config=%v", rec.Code, pkg.Config)
	}
}

func TestExistingConfigGuards(t *testing.T) {
	e, _ := newAuthTestServer(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/sd-api/js/get_configs"},
		{http.MethodPost, "/sd-api/js/set_configs"},
		{http.MethodPost, "/sd-api/js/delete_unused_configs"},
		{http.MethodPost, "/sd-api/js/reset_config"},
		{http.MethodPost, "/sd-api/package/_/config?id=alice/test"},
		{http.MethodPost, "/sd-api/banconfig/set"},
		{http.MethodPost, "/sd-api/banconfig/import"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			for _, body := range []string{`{}`, `{"token":"` + authTestToken + `"}`, `{invalid`} {
				rec := authTestRequest(e, tc.method, tc.path, body, "")
				if rec.Code != http.StatusForbidden {
					t.Errorf("status = %d, want 403; body = %s", rec.Code, rec.Body.String())
				}
			}
		})
	}
}

func TestManagementRoutesRequireAuth(t *testing.T) {
	for _, mode := range []struct {
		name string
		demo bool
	}{{"normal", false}, {"demo", true}} {
		t.Run(mode.name, func(t *testing.T) {
			e, testDice := newAuthTestServer(t)
			testDice.Parent.JustForTest = mode.demo
			testDice.Parent.AccessTokens.Store("", true)
			testDice.Parent.AccessTokens.Store("disabled", false)

			// The innermost sentinel proves the central guard runs before any
			// handler, including handlers without their own doAuth call. Never
			// invoke real network, profiling, upgrade or process-control actions.
			reached := false
			e.Use(func(_ echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					reached = true
					return c.NoContent(http.StatusNoContent)
				}
			})
			routes := e.Routes()
			sort.Slice(routes, func(i, j int) bool {
				return routes[i].Method+routes[i].Path < routes[j].Method+routes[j].Path
			})
			tested := 0
			for _, route := range routes {
				if !strings.HasPrefix(route.Path, "/sd-api/") && !strings.HasPrefix(route.Path, "/dice/api/store/") {
					continue
				}
				switch route.Method {
				case http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut, http.MethodPatch, http.MethodHead, http.MethodOptions:
				default: // Echo's synthetic RouteNotFound entries are not HTTP methods.
					continue
				}
				if authTestPublicRoutes[route.Method+" "+route.Path] {
					continue
				}
				tested++
				t.Run(route.Method+" "+route.Path, func(t *testing.T) {
					segments := strings.Split(route.Path, "/")
					for i, segment := range segments {
						if strings.HasPrefix(segment, ":") || segment == "*" {
							segments[i] = "test"
						}
					}
					path := strings.Join(segments, "/")
					for _, credential := range []struct {
						name, token, query, body string
						allowed                  bool
					}{
						{name: "missing", body: `{malformed`},
						{name: "invalid header", token: "invalid"},
						{name: "invalid query", query: "?token=invalid"},
						{name: "disabled", token: "disabled"},
						{name: "invalid header wins", token: "invalid", query: "?token=" + authTestToken},
						{name: "JSON is not a credential", body: `{"token":"` + authTestToken + `"}`},
						{name: "valid header", token: authTestToken, allowed: true},
						{name: "valid query", query: "?token=" + authTestToken, allowed: true},
						{name: "valid header wins", token: authTestToken, query: "?token=invalid", allowed: true},
					} {
						reached = false
						rec := authTestRequest(e, route.Method, path+credential.query, credential.body, credential.token)
						wantStatus := http.StatusForbidden
						if credential.allowed {
							wantStatus = http.StatusNoContent
						}
						if rec.Code != wantStatus || reached != credential.allowed {
							t.Errorf("%s: status=%d reached=%v, want status=%d reached=%v", credential.name, rec.Code, reached, wantStatus, credential.allowed)
						}
					}
				})
			}
			if tested < 190 {
				t.Fatalf("only checked %d routes; expected at least the 190 protected registered routes", tested)
			}
			t.Logf("checked %d protected routes with 9 credential cases", tested)
		})
	}
}

func TestManagementAuthScopeAndPublicRoutes(t *testing.T) {
	e, _ := newAuthTestServer(t)
	marker := func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }
	e.GET("/sd-api/future-route", marker)
	e.GET("/dice/api/store/future-route", marker)
	e.GET("/*", marker) // Model the UI fallback without reading any real files.
	e.Use(func(_ echo.HandlerFunc) echo.HandlerFunc { return marker })

	for _, tc := range []struct {
		method, target string
		status         int
	}{
		{http.MethodGet, "/sd-api/preInfo", http.StatusNoContent},
		{http.MethodPost, "/sd-api/signin", http.StatusNoContent},
		{http.MethodGet, "/sd-api/signin/salt", http.StatusNoContent},
		{http.MethodGet, "/sd-api/utils/ga/example", http.StatusNoContent},
		{http.MethodPost, "/sd-api/force_stop", http.StatusNoContent},
		{http.MethodGet, "/", http.StatusNoContent},
		{http.MethodGet, "/assets/app.js", http.StatusNoContent},
		{http.MethodGet, "/sd-api-unrelated", http.StatusNoContent},
		{http.MethodGet, "/dice/api/store-unrelated", http.StatusNoContent},
		{http.MethodGet, "/sd-api", http.StatusForbidden},
		{http.MethodGet, "/sd-api/unknown", http.StatusForbidden},
		{http.MethodGet, "/dice/api/store", http.StatusForbidden},
		{http.MethodGet, "/dice/api/store/unknown", http.StatusForbidden},
		{http.MethodGet, "/sd-api/future-route", http.StatusForbidden},
		{http.MethodGet, "/dice/api/store/future-route", http.StatusForbidden},
		{http.MethodPost, "/sd-api/preInfo", http.StatusForbidden},
		{http.MethodHead, "/sd-api/preInfo", http.StatusForbidden},
		{http.MethodGet, "/sd-api/preInfo/extra", http.StatusForbidden},
		{http.MethodGet, "/sd-api/signin", http.StatusForbidden},
		{http.MethodPost, "/sd-api/signin/extra", http.StatusForbidden},
		{http.MethodPost, "/sd-api/utils/ga/example", http.StatusForbidden},
		{http.MethodGet, "/sd-api/force_stop", http.StatusForbidden},
		{http.MethodGet, "/sd%2Dapi/package/list", http.StatusForbidden},
		{http.MethodGet, "/sd-api/package/%74est/config", http.StatusForbidden},
	} {
		t.Run(tc.method+tc.target, func(t *testing.T) {
			rec := authTestRequest(e, tc.method, tc.target, `{}`, "")
			if rec.Code != tc.status {
				t.Fatalf("status=%d, want %d", rec.Code, tc.status)
			}
		})
	}
}

func TestManagementAuthPreservesStaticLoginAssets(t *testing.T) {
	_, testDice := newAuthTestServer(t)
	files := fstest.MapFS{
		"index.html":       {Data: []byte(`<html><div id="app"></div><script src="/assets/login.js"></script></html>`)},
		"assets/login.js":  {Data: []byte(`/* login UI fixture */`)},
		"assets/login.css": {Data: []byte(`body { margin: 0; }`)},
		"sd-api/unknown":   {Data: []byte("must not bypass API authentication")},
	}
	e := echo.New()
	// Match main.go's registration order and use the real static handler,
	// not the sentinel used by the route-coverage test. No browser is run.
	e.StaticFS("/", files)
	Bind(e, testDice.Parent)

	for _, token := range []string{"", "stale-token"} {
		for _, tc := range []struct{ path, file string }{
			{"/", "index.html"},
			{"/assets/login.js", "assets/login.js"},
			{"/assets/login.css", "assets/login.css"},
		} {
			rec := authTestRequest(e, http.MethodGet, tc.path, "", token)
			if rec.Code != http.StatusOK || rec.Body.String() != string(files[tc.file].Data) {
				t.Fatalf("static %s with token %q: status=%d, unexpected login asset response", tc.path, token, rec.Code)
			}
		}
		for _, path := range []string{"/sd-api/package/list", "/sd-api/unknown"} {
			if rec := authTestRequest(e, http.MethodGet, path, "", token); rec.Code != http.StatusForbidden {
				t.Fatalf("static fallback exposed %s: status=%d", path, rec.Code)
			}
		}
	}
}

func TestManagementAuthPreservesLogin(t *testing.T) {
	for _, tc := range []struct{ name, token string }{
		{"first visit", ""},
		{"stale cached token", "stale-token"},
		{"disabled cached token", "disabled-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.MkdirAll("data", 0o755); err != nil {
				t.Fatal(err)
			}
			e, testDice := newAuthTestServer(t)
			testDice.Parent.AccessTokens.Store("disabled-token", false)

			// trySignIn gets the salt, probes hello and handles rejection by
			// attempting the no-password login before showing the unlock UI.
			for _, path := range []string{"/sd-api/preInfo", "/sd-api/signin/salt"} {
				if rec := authTestRequest(e, http.MethodGet, path, "", tc.token); rec.Code != http.StatusOK {
					t.Fatalf("public %s returned %d", path, rec.Code)
				}
			}
			if rec := authTestRequest(e, http.MethodGet, "/sd-api/hello", "", tc.token); rec.Code != http.StatusForbidden {
				t.Fatalf("unauthenticated hello returned %d", rec.Code)
			}
			for _, body := range []string{`{"password":"defaultSignin"}`, `{"password":"wrong"}`} {
				if rec := authTestRequest(e, http.MethodPost, "/sd-api/signin", body, tc.token); rec.Code != http.StatusBadRequest {
					t.Fatalf("incorrect password returned %d", rec.Code)
				}
			}
			rec := authTestRequest(e, http.MethodPost, "/sd-api/signin", `{"password":"configured-test-password"}`, tc.token)
			var result struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || rec.Code != http.StatusOK || result.Token == "" {
				t.Fatalf("login failed: status=%d error=%v", rec.Code, err)
			}
			if !testDice.Parent.AccessTokens.Exists(result.Token) {
				t.Fatal("login did not register the token")
			}
			if rec := authTestRequest(e, http.MethodGet, "/sd-api/hello", "", result.Token); rec.Code != http.StatusOK {
				t.Fatalf("issued header token returned %d", rec.Code)
			}
			if rec := authTestRequest(e, http.MethodGet, "/sd-api/hello?token="+result.Token, "", ""); rec.Code != http.StatusOK {
				t.Fatalf("issued query token returned %d", rec.Code)
			}
		})
	}
}

func TestManagementAuthPreservesPasswordlessSetup(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	e, testDice := newAuthTestServer(t)
	testDice.Parent.UIPasswordHash = ""
	// Even without a configured password, management access still requires
	// going through signin to obtain a token; do not exempt management APIs.
	if rec := authTestRequest(e, http.MethodGet, "/sd-api/hello", "", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("passwordless setup bypassed the token check: status=%d", rec.Code)
	}
	rec := authTestRequest(e, http.MethodPost, "/sd-api/signin", `{"password":"defaultSignin"}`, "stale-token")
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || rec.Code != http.StatusOK || result.Token == "" {
		t.Fatalf("passwordless setup login failed: status=%d error=%v", rec.Code, err)
	}
	if rec := authTestRequest(e, http.MethodGet, "/sd-api/hello", "", result.Token); rec.Code != http.StatusOK {
		t.Fatalf("setup token returned %d", rec.Code)
	}
}

func TestManagementAuthPreservesCORSPreflight(t *testing.T) {
	_, testDice := newAuthTestServer(t)
	e := echo.New()
	// Match main.go: CORS wraps the API middleware and can handle OPTIONS,
	// but must not exempt the subsequent real POST from authentication.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderContentType, "token"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
	}))
	Bind(e, testDice.Parent)
	req := httptest.NewRequest(http.MethodOptions, "/sd-api/banconfig/map_add_one", nil)
	req.Header.Set(echo.HeaderOrigin, "https://frontend.example")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
	req.Header.Set(echo.HeaderAccessControlRequestHeaders, "content-type,token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get(echo.HeaderAccessControlAllowOrigin) == "" {
		t.Fatalf("preflight rejected: status=%d headers=%v", rec.Code, rec.Header())
	}
	req = httptest.NewRequest(http.MethodPost, "/sd-api/banconfig/map_add_one", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderOrigin, "https://frontend.example")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("CORS allowed unauthenticated POST: status=%d", rec.Code)
	}
}

func TestSensitiveHandlersRequireAuthDirectly(t *testing.T) {
	_, _ = newAuthTestServer(t)
	for _, tc := range []struct {
		name    string
		handler echo.HandlerFunc
	}{
		{"ban add", banMapAddOne},
		{"ban delete", banMapDeleteOne},
		{"ban export", banExport},
		{"package list", packageList},
		{"package detail", packageGet},
		{"package config", packageGetConfig},
		{"package schema", packageGetConfigSchema},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{invalid`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if err := tc.handler(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}
			if rec.Code != http.StatusForbidden {
				t.Fatalf("direct handler returned %d, want 403 before decoding/accessing state", rec.Code)
			}
		})
	}
}

func TestBanDeleteAndExportRequireAuth(t *testing.T) {
	e, testDice := newAuthTestServer(t)
	const id = "QQ:auth-regression"
	testDice.Config.BanList.Map.Store(id, &dice.BanListInfoItem{ID: id, Rank: dice.BanRankBanned})
	for _, demo := range []bool{false, true} {
		testDice.Parent.JustForTest = demo
		for _, tc := range []struct{ method, path string }{
			{http.MethodPost, "/sd-api/banconfig/map_delete_one"},
			{http.MethodGet, "/sd-api/banconfig/export"},
		} {
			rec := authTestRequest(e, tc.method, tc.path, `{"ID":"QQ:auth-regression"}`, "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("demo=%v %s returned %d", demo, tc.path, rec.Code)
			}
			if !testDice.Config.BanList.Map.Exists(id) {
				t.Fatal("unauthenticated request removed a ban")
			}
		}
	}
}

func TestDoAuthFailsClosedWithoutManager(t *testing.T) {
	_, _ = newAuthTestServer(t)
	for _, state := range []*dice.Dice{nil, {}} {
		myDice = state
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/sd-api/hello", nil)
		req.Header.Set("Token", authTestToken)
		if doAuth(e.NewContext(req, httptest.NewRecorder())) {
			t.Fatal("uninitialized authentication state allowed access")
		}
	}
}

func TestRequireForceStopKey(t *testing.T) {
	for _, tc := range []struct {
		name, key, body string
		status          int
	}{
		{"empty configured key", "", `{}`, http.StatusForbidden},
		{"empty supplied key", "test-secret==", `{}`, http.StatusForbidden},
		{"wrong key", "test-secret==", `{"key":"invalid"}`, http.StatusForbidden},
		{"truncated key", "test-secret==", `{"key":"test-secret"}`, http.StatusForbidden},
		{"full key", "test-secret==", `{"key":"test-secret=="}`, http.StatusOK},
		{"malformed body", "test-secret==", `{invalid`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FSTOP_KEY", tc.key)
			e := echo.New()
			// Test only authorization; never execute forceStop/os.Exit.
			e.POST("/check-key", func(c echo.Context) error {
				if err := requireForceStopKey(c); err != nil {
					return err
				}
				return c.NoContent(http.StatusOK)
			})
			rec := authTestRequest(e, http.MethodPost, "/check-key", tc.body, "")
			if rec.Code != tc.status {
				t.Fatalf("status=%d, want %d", rec.Code, tc.status)
			}
		})
	}
}
