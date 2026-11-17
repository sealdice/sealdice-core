// Package uploadgw 提供了基于 tus 协议的统一上传网关。
//
// 它替代了此前散落在 js/deck/helpdoc 三个模块里的自研分片上传实现：
// 三处共用同一个 tusd handler，具体业务差异通过 Upload-Metadata 里的
// module 字段分发到各自的 Destination。
package uploadgw

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/memorylocker"

	"sealdice-core/dice"
)

const (
	// BasePath 是上传网关挂载的 URL 前缀（tus 创建端点为 BasePath 本身）。
	BasePath = "/sd-api/v2/upload/"

	// 上传元数据字段（tus Upload-Metadata）。
	MetaModule   = "module"
	MetaFilename = "filename"
	MetaGroup    = "group"
	MetaSHA256   = "sha256"

	// maxUploadSize 限制单次上传大小。
	maxUploadSize = 512 << 20 // 512 MiB
)

// Gateway 是所有上传业务的统一入口。
type Gateway struct {
	dm      *dice.DiceManager
	store   filestore.FileStore
	handler *tusd.Handler
	dests   map[string]Destination
}

// New 构建网关。rootDir 是上传暂存目录。
func New(dm *dice.DiceManager, rootDir string) (*Gateway, error) {
	if rootDir == "" {
		rootDir = filepath.Join(dataDirOf(dm), ".uploads")
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}

	store := filestore.New(rootDir)
	locker := memorylocker.New()
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	g := &Gateway{
		dm:    dm,
		store: store,
		dests: map[string]Destination{
			ModuleJS:               &scriptDestination{dm: dm},
			ModuleDeck:             &deckDestination{dm: dm},
			ModuleHelpdoc:          &helpdocDestination{dm: dm},
			ModuleResource:         &resourceDestination{},
			ModuleCensor:           &censorDestination{},
			ModuleCustomReply:      &customReplyDestination{dm: dm},
			ModuleExtensionInstall: &extensionInstallDestination{dm: dm},
		},
	}

	cfg := tusd.Config{
		BasePath:                  BasePath,
		StoreComposer:             composer,
		MaxSize:                   maxUploadSize,
		RespectForwardedHeaders:   true,
		PreUploadCreateCallback:   g.preUploadCreate,
		PreFinishResponseCallback: g.preFinish,
	}
	h, err := tusd.NewHandler(cfg)
	if err != nil {
		return nil, err
	}
	g.handler = h
	return g, nil
}

// Handler 返回 tus 协议处理器（已含路由），主要供测试与自定义挂载使用。
func (g *Gateway) Handler() http.Handler {
	return g.handler
}

// Mount 把网关挂载到 Echo 上。auth 为鉴权中间件。
func (g *Gateway) Mount(e *echo.Echo, auth echo.MiddlewareFunc) {
	// tusd 的 NewHandler 自带路由，要求路径已被剥离到 base 之后（"" 或 "<id>"）。
	// 因此用 StripPrefix 去掉 BasePath（不带结尾斜杠，兼容 "/upload" 与 "/upload/"）。
	wrapped := echo.WrapHandler(http.StripPrefix(strings.TrimSuffix(BasePath, "/"), g.handler))
	grp := e.Group(strings.TrimSuffix(BasePath, "/"), auth)
	grp.Any("", wrapped)
	grp.Any("/*", wrapped)
}

func (g *Gateway) destination(meta map[string]string) (Destination, error) {
	module := strings.ToLower(strings.TrimSpace(meta[MetaModule]))
	dst := g.dests[module]
	if dst == nil {
		return nil, tusd.NewError("ERR_UNKNOWN_MODULE", "未知的上传模块: "+module, http.StatusBadRequest)
	}
	return dst, nil
}

func (g *Gateway) preUploadCreate(hook tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {
	if g.dm.JustForTest {
		return tusd.HTTPResponse{}, tusd.FileInfoChanges{},
			tusd.NewError("ERR_TEST_MODE", "展示模式不支持该操作", http.StatusForbidden)
	}
	meta := normalizeMeta(hook.Upload.MetaData)
	dst, err := g.destination(meta)
	if err != nil {
		return tusd.HTTPResponse{}, tusd.FileInfoChanges{}, err
	}
	if err := dst.Validate(meta, hook.Upload.Size); err != nil {
		return tusd.HTTPResponse{}, tusd.FileInfoChanges{}, err
	}
	// 只保留已知字段，避免把客户端任意元数据长期留存。
	clean := map[string]string{
		MetaModule:   strings.ToLower(strings.TrimSpace(meta[MetaModule])),
		MetaFilename: meta[MetaFilename],
		MetaSHA256:   strings.ToLower(strings.TrimSpace(meta[MetaSHA256])),
	}
	if grp := meta[MetaGroup]; grp != "" {
		clean[MetaGroup] = grp
	}
	return tusd.HTTPResponse{}, tusd.FileInfoChanges{MetaData: clean}, nil
}

func (g *Gateway) preFinish(hook tusd.HookEvent) (tusd.HTTPResponse, error) {
	meta := normalizeMeta(hook.Upload.MetaData)
	dst, err := g.destination(meta)
	if err != nil {
		return tusd.HTTPResponse{}, err
	}
	staged := hook.Upload.Storage[filestore.StorageKeyPath]
	if staged == "" {
		return tusd.HTTPResponse{}, tusd.NewError("ERR_STAGED_MISSING", "未找到暂存文件", http.StatusInternalServerError)
	}
	if err := verifySHA256(staged, meta[MetaSHA256]); err != nil {
		g.terminate(hook)
		return tusd.HTTPResponse{}, err
	}
	if err := dst.Install(hook.Context, staged, meta); err != nil {
		g.terminate(hook)
		return tusd.HTTPResponse{}, err
	}
	// 安装完成后删除 tus 暂存文件，避免残留。
	g.terminate(hook)
	return tusd.HTTPResponse{}, nil
}

func (g *Gateway) terminate(hook tusd.HookEvent) {
	up, err := g.store.GetUpload(hook.Context, hook.Upload.ID)
	if err != nil {
		return
	}
	_ = g.store.AsTerminatableUpload(up).Terminate(hook.Context)
}

func normalizeMeta(meta map[string]string) map[string]string {
	out := make(map[string]string, len(meta))
	for k, v := range meta {
		out[strings.ToLower(strings.TrimSpace(k))] = v
	}
	return out
}

func verifySHA256(path, wantHex string) error {
	wantHex = strings.ToLower(strings.TrimSpace(wantHex))
	if wantHex == "" {
		return tusd.NewError("ERR_CHECKSUM_REQUIRED", "缺少文件校验值 sha256", http.StatusBadRequest)
	}
	f, err := os.Open(path)
	if err != nil {
		return tusd.NewError("ERR_STAGED_OPEN", "读取暂存文件失败", http.StatusInternalServerError)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return tusd.NewError("ERR_STAGED_READ", "读取暂存文件失败", http.StatusInternalServerError)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != wantHex {
		return tusd.NewError("ERR_CHECKSUM_MISMATCH", "文件校验失败", http.StatusBadRequest)
	}
	return nil
}
