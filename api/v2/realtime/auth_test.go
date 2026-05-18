package realtime

import "testing"

func TestTokenFromHandshakePrefersSocketAuth(t *testing.T) {
	token := tokenFromHandshake(
		map[string]any{
			"Authorization": []string{"Bearer header-token"},
			"token":         []string{"legacy-header-token"},
		},
		map[string]any{
			"token": []string{"query-token"},
		},
		map[string]any{
			"token": "auth-token",
		},
	)
	if token != "auth-token" {
		t.Fatalf("token = %q, want auth-token", token)
	}
}

func TestTokenFromHandshakeFallsBackToHeadersAndQuery(t *testing.T) {
	token := tokenFromHandshake(
		map[string]any{
			"Authorization": []string{"Bearer header-token"},
			"token":         []string{"legacy-header-token"},
		},
		map[string]any{
			"token": []string{"query-token"},
		},
		nil,
	)
	if token != "header-token" {
		t.Fatalf("token = %q, want header-token", token)
	}

	token = tokenFromHandshake(
		map[string]any{
			"token": []string{"legacy-header-token"},
		},
		map[string]any{
			"token": []string{"query-token"},
		},
		nil,
	)
	if token != "legacy-header-token" {
		t.Fatalf("token = %q, want legacy-header-token", token)
	}

	token = tokenFromHandshake(nil, map[string]any{"token": []string{"query-token"}}, nil)
	if token != "query-token" {
		t.Fatalf("token = %q, want query-token", token)
	}
}
