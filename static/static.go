package static

import (
	"embed"
)

//go:generate go run gen/download-fe.go

//go:embed frontend
var Frontend embed.FS

//go:embed scripts
var Scripts embed.FS
