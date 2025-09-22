package static

import (
	"embed"
)

// 依赖 ui 的构建产物作为静态资源，缺失时请参考 README 进行一次构建，或者手动前往 github action 下载最新 artifacts。

//go:embed frontend
var Frontend embed.FS

//go:embed scripts
var Scripts embed.FS
