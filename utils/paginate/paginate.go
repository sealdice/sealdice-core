// Package paginate handles in-memory lists; database pagination stays in the query layer.
package paginate

// Page has the same fields as the API pagination DTO, without depending on HTTP.
type Page[T any] struct {
	List     []T
	Total    int64
	Page     int
	PageSize int
}

// Slice returns a shallow copy of the requested window. Empty/out-of-range pages
// return an empty non-nil list. Callers may impose their own maximum size: reply
// editing intentionally requests all rules, so this helper must not truncate it.
func Slice[T any](items []T, page, size int) Page[T] {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	result := Page[T]{List: []T{}, Total: int64(len(items)), Page: page, PageSize: size}
	// Check before multiplying to avoid overflow for untrusted page/size values.
	if len(items) == 0 || page-1 > (len(items)-1)/size {
		return result
	}
	start := (page - 1) * size
	end := start + min(size, len(items)-start)
	result.List = append(result.List, items[start:end]...)
	return result
}
