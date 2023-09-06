package static

import (
	"embed"
)

//go:generate go run gen/download-fe.go

//go:embed frontend
var Static embed.FS
