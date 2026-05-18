package realtime

import "strings"

func TokenFromHandshake(headers map[string]any, query map[string]any, auth map[string]any) string {
	if auth != nil {
		if token, ok := auth["token"].(string); ok && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token)
		}
	}

	headerToken := firstHeaderValue(headers, "Authorization")
	if strings.HasPrefix(headerToken, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(headerToken, "Bearer "))
	}

	if token := firstHeaderValue(headers, "token"); token != "" {
		return token
	}

	return firstQueryValue(query, "token")
}

func firstHeaderValue(headers map[string]any, key string) string {
	for name, values := range headers {
		if !strings.EqualFold(name, key) {
			continue
		}
		return firstCollectionValue(values)
	}
	return ""
}

func firstQueryValue(query map[string]any, key string) string {
	values, ok := query[key]
	if !ok {
		return ""
	}
	return firstCollectionValue(values)
}

func firstCollectionValue(value any) string {
	switch vv := value.(type) {
	case string:
		return strings.TrimSpace(vv)
	case []string:
		if len(vv) == 0 {
			return ""
		}
		return strings.TrimSpace(vv[0])
	case []any:
		if len(vv) == 0 {
			return ""
		}
		if first, ok := vv[0].(string); ok {
			return strings.TrimSpace(first)
		}
	}
	return ""
}
