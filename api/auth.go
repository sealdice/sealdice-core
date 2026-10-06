package api

import (
	"crypto/subtle"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// managementAuthMiddleware makes both management API namespaces private by
// default, including routes added later and the legacy store aliases. Use the
// matched route rather than a raw-URL allowlist so routing/encoding differences
// cannot exempt a different handler. The URL fallback also protects unmatched
// API requests that would otherwise reach the static UI fallback.
func managementAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	authenticated := AuthMiddleware(next)
	return func(c echo.Context) error {
		if !isManagementAPIPath(c.Path()) && !isManagementAPIPath(c.Request().URL.Path) {
			return next(c)
		}

		// Keep this list method-specific and limited to intentional public
		// handlers. force_stop authenticates separately using FSTOP_KEY.
		switch c.Request().Method + " " + c.Path() {
		case "GET /sd-api/preInfo",
			"POST /sd-api/signin",
			"GET /sd-api/signin/salt",
			"GET /sd-api/utils/ga/:uid",
			"POST /sd-api/force_stop":
			return next(c)
		default:
			return authenticated(c)
		}
	}
}

func isManagementAPIPath(path string) bool {
	return path == "/sd-api" || strings.HasPrefix(path, "/sd-api/") ||
		path == "/dice/api/store" || strings.HasPrefix(path, "/dice/api/store/")
}

// requireForceStopKey is separate from UI token authentication because the
// Android host uses its own shared secret to stop the embedded process.
func requireForceStopKey(c echo.Context) error {
	key := os.Getenv("FSTOP_KEY")
	if key == "" {
		return echo.ErrForbidden
	}
	var request fStopEcho
	if err := c.Bind(&request); err != nil {
		return echo.ErrBadRequest
	}
	if subtle.ConstantTimeCompare([]byte(request.Key), []byte(key)) != 1 {
		return echo.ErrForbidden
	}
	return nil
}
