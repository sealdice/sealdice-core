package js

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dop251/goja"

	"sealdice-core/api/v2/internal/uploadcore"
	"sealdice-core/dice"
	cmn "sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/paginate"
	"sealdice-core/utils/paginate/slicep"
)

type Service struct {
	dice          *dice.Dice
	dm            *dice.DiceManager
	uploadManager *uploadcore.Manager
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dice:          dm.GetDice(),
		dm:            dm,
		uploadManager: uploadcore.NewManager(filepath.Join(dm.GetDice().BaseConfig.DataDir, "scripts", ".uploads")),
	}
}

// ============================================================
// Route registration
// ============================================================

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/status", s.GetStatus, func(o *huma.Operation) {
		o.Description = "获取JS引擎状态"
	})
	huma.Get(grp, "/list", s.GetList, func(o *huma.Operation) {
		o.Description = "获取JS插件分页列表"
	})
	huma.Get(grp, "/record", s.GetRecord, func(o *huma.Operation) {
		o.Description = "获取JS执行输出记录"
	})
	huma.Get(grp, "/configs", s.GetConfigs, func(o *huma.Operation) {
		o.Description = "获取插件配置"
	})
	huma.Get(grp, "/dead-configs", s.GetDeadConfigs, func(o *huma.Operation) {
		o.Description = "获取死配置列表"
	})
	huma.Get(grp, "/{name}/data/list", s.DataList, func(o *huma.Operation) {
		o.Description = "获取插件KV数据分页列表"
	})
	huma.Get(grp, "/{name}/data", s.DataGet, func(o *huma.Operation) {
		o.Description = "获取插件KV数据单个值"
	})
	huma.Get(grp, "/{name}/data/info", s.DataInfo, func(o *huma.Operation) {
		o.Description = "获取插件数据库统计信息"
	})
	huma.Get(grp, "/upload/{sessionId}/{index}", s.GetUploadChunkStatus, func(o *huma.Operation) {
		o.Description = "获取JS脚本上传分块状态"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/reload", s.Reload, func(o *huma.Operation) {
		o.Description = "重载JS引擎"
	})
	huma.Post(grp, "/shutdown", s.Shutdown, func(o *huma.Operation) {
		o.Description = "关闭JS引擎"
	})
	huma.Post(grp, "/enable", s.Enable, func(o *huma.Operation) {
		o.Description = "启用JS脚本"
	})
	huma.Post(grp, "/disable", s.Disable, func(o *huma.Operation) {
		o.Description = "禁用JS脚本"
	})
	huma.Post(grp, "/delete", s.Delete, func(o *huma.Operation) {
		o.Description = "删除JS脚本"
	})
	huma.Post(grp, "/execute", s.Execute, func(o *huma.Operation) {
		o.Description = "执行JS代码"
	})
	huma.Post(grp, "/check-update", s.CheckUpdate, func(o *huma.Operation) {
		o.Description = "检查JS脚本更新"
	})
	huma.Post(grp, "/update", s.Update, func(o *huma.Operation) {
		o.Description = "更新JS脚本"
	})
	huma.Post(grp, "/upload", s.Upload, func(o *huma.Operation) {
		o.Description = "上传JS脚本文件"
	})
	huma.Post(grp, "/upload/init", s.InitUpload, func(o *huma.Operation) {
		o.Description = "初始化JS脚本上传会话"
	})
	huma.Put(grp, "/upload/{sessionId}/{index}", s.UploadChunk, func(o *huma.Operation) {
		o.Description = "上传JS脚本分块"
	})
	huma.Post(grp, "/upload/complete", s.CompleteUpload, func(o *huma.Operation) {
		o.Description = "完成JS脚本上传"
	})
	huma.Post(grp, "/configs", s.SetConfigs, func(o *huma.Operation) {
		o.Description = "保存插件配置"
	})
	huma.Post(grp, "/configs/reset", s.ResetConfig, func(o *huma.Operation) {
		o.Description = "重置插件配置项"
	})
	huma.Post(grp, "/dead-configs/delete", s.DeleteDeadConfigs, func(o *huma.Operation) {
		o.Description = "删除死配置"
	})
	huma.Post(grp, "/{name}/data", s.DataSet, func(o *huma.Operation) {
		o.Description = "设置插件KV数据"
	})
	huma.Post(grp, "/{name}/data/delete", s.DataDelete, func(o *huma.Operation) {
		o.Description = "删除插件KV数据"
	})
	huma.Post(grp, "/{name}/data/shrink", s.DataShrink, func(o *huma.Operation) {
		o.Description = "压缩插件数据库"
	})
}

// ============================================================
// Handlers – Public / Auth
// ============================================================

func (s *Service) GetStatus(_ context.Context, _ *cmn.Empty) (*StatusItemResponse, error) {
	return response.NewItemResponse(JsStatusResp{
		Status: s.dice.Config.JsEnable,
	}), nil
}

func (s *Service) GetList(_ context.Context, req *ListQuery) (*ListItemResponse, error) {
	if !s.dice.Config.JsEnable {
		return response.NewItemResponse(JsListResp{Success: true, Data: []JsInfo{}, Total: 0}), nil
	}

	items := make([]JsInfo, 0, len(s.dice.JsScriptList))
	for _, si := range s.dice.JsScriptList {
		if si == nil {
			continue
		}
		if req.Keyword != "" && !matchJSKeyword(si, req.Keyword) {
			continue
		}
		items = append(items, JsInfo{
			JsScriptInfo:   *si,
			HasConfig:      s.pluginHasConfig(si.Name),
			BuiltinUpdated: si.Builtin && !s.dice.JsBuiltinDigestSet[si.Digest],
		})
	}

	sortJS(items, req.SortBy, req.SortOrder)

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	pager := paginate.SimplePaginate(slicep.Adapter(items), int64(pageSize), int64(page))
	total, err := pager.GetTotal()
	if err != nil {
		return nil, huma.Error500InternalServerError("分页失败")
	}

	var paged []JsInfo
	if err := pager.Get(&paged); err != nil {
		return nil, huma.Error500InternalServerError("分页失败")
	}

	return response.NewItemResponse(JsListResp{
		Success:  true,
		Data:     paged,
		Total:    int(total),
		Page:     int(pager.GetCurrentPage()),
		PageSize: int(pager.GetListRows()),
	}), nil
}

func (s *Service) GetRecord(_ context.Context, _ *cmn.Empty) (*ExecuteItemResponse, error) {
	if s.dice.JsPrinter == nil {
		return response.NewItemResponse(JsExecuteResp{Outputs: []string{}}), nil
	}
	outputs := s.dice.JsPrinter.RecordEnd()
	return response.NewItemResponse(JsExecuteResp{Outputs: outputs}), nil
}

func (s *Service) GetConfigs(_ context.Context, req *struct {
	Name string `query:"name"`
}) (*ConfigMapItemResponse, error) {
	cm := s.dice.ConfigManager
	if cm == nil {
		return response.NewItemResponse(PluginConfigMap{}), nil
	}

	result := make(PluginConfigMap)
	for name, pc := range cm.Plugins {
		if req.Name != "" && name != req.Name {
			continue
		}
		configItems := make([]*dice.ConfigItem, 0, len(pc.OrderedConfigKeys))
		for _, key := range pc.OrderedConfigKeys {
			if ci, ok := pc.Configs[key]; ok {
				configItems = append(configItems, ci)
			}
		}
		result[name] = &APIPluginConfig{
			PluginName:        pc.PluginName,
			Configs:           configItems,
			OrderedConfigKeys: pc.OrderedConfigKeys,
		}
	}

	return response.NewItemResponse(result), nil
}

func (s *Service) GetDeadConfigs(_ context.Context, _ *cmn.Empty) (*DeadConfigsItemResponse, error) {
	cm := s.dice.ConfigManager
	if cm == nil {
		return response.NewItemResponse(DeadConfigsResp{Configs: []DeadConfig{}}), nil
	}

	var dead []DeadConfig
	for name := range cm.Plugins {
		if s.findPluginByName(name) != nil {
			continue
		}
		dead = append(dead, DeadConfig{Name: name})
	}
	sort.Slice(dead, func(i, j int) bool { return dead[i].Name < dead[j].Name })

	return response.NewItemResponse(DeadConfigsResp{Configs: dead}), nil
}

func (s *Service) DataList(_ context.Context, req *struct {
	Name string `path:"name"`
	DataListQuery
}) (*DataListItemResponse, error) {
	st, err := resolveStorage(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	resp, err := st.listKeys(req.Page, req.PageSize, req.Keyword)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(*resp), nil
}

func (s *Service) DataGet(_ context.Context, req *struct {
	Name string `path:"name"`
	Key  string `query:"key"`
}) (*DataValueItemResponse, error) {
	if req.Key == "" {
		return nil, huma.Error400BadRequest("key不能为空")
	}
	st, err := resolveStorage(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	kv, err := st.getValue(req.Key)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	return response.NewItemResponse(*kv), nil
}

func (s *Service) DataInfo(_ context.Context, req *NamePath) (*DataInfoItemResponse, error) {
	st, err := resolveStorage(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	info, err := st.info()
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(*info), nil
}

// ============================================================
// Handlers – Protected (write)
// ============================================================

func (s *Service) Reload(_ context.Context, _ *cmn.Empty) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if !s.dice.JsReloadLock.TryLock() {
		return nil, huma.Error400BadRequest("正在重载中，请稍后再试")
	}
	defer s.dice.JsReloadLock.Unlock()
	s.dice.JsReload()
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) Shutdown(_ context.Context, _ *cmn.Empty) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	s.dice.JsShutdown()
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) Enable(_ context.Context, req *NameReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if req.Body.Name == "" {
		return nil, huma.Error400BadRequest("name不能为空")
	}
	dice.JsEnable(s.dice, req.Body.Name)
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) Disable(_ context.Context, req *NameReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if req.Body.Name == "" {
		return nil, huma.Error400BadRequest("name不能为空")
	}
	dice.JsDisable(s.dice, req.Body.Name)
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) Delete(_ context.Context, req *FilenameReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if !s.dice.Config.JsEnable {
		return nil, huma.Error400BadRequest("js扩展支持已关闭")
	}
	if req.Body.Filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}
	for _, si := range s.dice.JsScriptList {
		if si != nil && si.Filename == req.Body.Filename {
			dice.JsDelete(s.dice, si)
			return response.NewItemResponse(JsSimpleResult{Success: true}), nil
		}
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) Execute(_ context.Context, req *ExecuteReq) (*ExecuteItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsExecuteResp{Outputs: []string{}}), nil
	}
	if !s.dice.Config.JsEnable {
		return nil, huma.Error400BadRequest("js扩展支持已关闭")
	}
	if req.Body.Value == "" {
		return nil, huma.Error400BadRequest("value不能为空")
	}

	loop := s.dice.ExtLoopManager.GetWebLoop()
	if loop == nil {
		return nil, huma.Error500InternalServerError("JS运行时未就绪")
	}

	s.dice.JsPrinter.RecordStart()

	var lastRet goja.Value
	waitRun := make(chan struct{})
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer close(waitRun)
		defer func() {
			if r := recover(); r != nil {
				s.dice.JsPrinter.Error(fmt.Sprintf("JS脚本报错: %v", r))
			}
		}()

		code := fmt.Sprintf("(function(exports, require, module) { %s })();", req.Body.Value)
		ret, err := vm.RunString(code)
		_ = err
		lastRet = ret
	})
	<-waitRun

	outputs := s.dice.JsPrinter.RecordEnd()
	resp := JsExecuteResp{
		Ret:     lastRet,
		Outputs: outputs,
	}

	return response.NewItemResponse(resp), nil
}

func (s *Service) CheckUpdate(_ context.Context, req *CheckUpdateReq) (*CheckUpdateItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsCheckUpdateResp{Success: false, Err: "展示模式不支持该操作"}), nil
	}
	if req.Body.Filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}
	for _, si := range s.dice.JsScriptList {
		if si != nil && si.Filename == req.Body.Filename {
			old, newCode, tempFileName, errUpdate := s.dice.JsCheckUpdate(si)
			if errUpdate == nil {
				return response.NewItemResponse(JsCheckUpdateResp{
					Success:      true,
					Old:          old,
					New:          newCode,
					Format:       "javascript",
					Filename:     si.Filename,
					TempFileName: tempFileName,
				}), nil
			}
			return response.NewItemResponse(JsCheckUpdateResp{Success: false, Err: errUpdate.Error()}), nil
		}
	}
	return response.NewItemResponse(JsCheckUpdateResp{Success: false, Err: "未找到脚本"}), nil
}

func (s *Service) Update(_ context.Context, req *UpdateReq) (*UpdateItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsUpdateResp{Success: false}), nil
	}
	if !s.dice.Config.JsEnable {
		return nil, huma.Error400BadRequest("js扩展支持已关闭")
	}
	if req.Body.Filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}
	for _, si := range s.dice.JsScriptList {
		if si != nil && si.Filename == req.Body.Filename {
			if err := s.dice.JsUpdate(si, req.Body.TempFileName); err == nil {
				s.dice.MarkModified()
				return response.NewItemResponse(JsUpdateResp{Success: true}), nil
			}
			return response.NewItemResponse(JsUpdateResp{Success: false}), nil
		}
	}
	return response.NewItemResponse(JsUpdateResp{Success: false}), nil
}

func (s *Service) Upload(_ context.Context, req *UploadReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}

	form := extractUploadForm(req.RawBody)
	if form == nil {
		return nil, huma.Error400BadRequest("未找到上传文件")
	}
	defer form.File.Close()

	sanitized := sanitizeFilename(form.File.Filename)
	if sanitized == "" {
		return nil, huma.Error400BadRequest("文件名非法")
	}

	dest := filepath.Join(s.dice.BaseConfig.DataDir, "scripts", sanitized)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, huma.Error500InternalServerError("创建目录失败")
	}
	out, err := os.Create(dest)
	if err != nil {
		return nil, huma.Error500InternalServerError("创建文件失败")
	}
	defer out.Close()
	if _, err := io.Copy(out, form.File); err != nil {
		return nil, huma.Error500InternalServerError("写入文件失败")
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) InitUpload(_ context.Context, req *UploadInitReq) (*response.ItemResponse[JsUploadSessionResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsUploadSessionResp{
			Success:         false,
			ResumeSupported: false,
		}), nil
	}

	body := req.Body
	filename := sanitizeFilename(body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("文件名非法")
	}
	if body.FileSize <= 0 {
		return nil, huma.Error400BadRequest("fileSize必须大于0")
	}
	if strings.TrimSpace(body.FileHash) == "" {
		return nil, huma.Error400BadRequest("fileHash不能为空")
	}

	chunkSize := body.ChunkSize
	if chunkSize <= 0 {
		chunkSize = uploadcore.DefaultChunkSize
	}

	session, err := s.uploadManager.Init(filename, body.FileSize, body.FileHash, chunkSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("创建上传目录失败")
	}

	return response.NewItemResponse(JsUploadSessionResp{
		Success:         true,
		SessionID:       session.SessionID,
		ChunkSize:       session.ChunkSize,
		UploadedChunks:  s.uploadManager.SortedUploadedChunks(session),
		UploadedBytes:   s.uploadManager.UploadedBytes(session),
		ExpectedChunks:  session.ExpectedChunks,
		ResumeSupported: true,
	}), nil
}

func (s *Service) GetUploadChunkStatus(_ context.Context, req *UploadChunkQuery) (*response.ItemResponse[JsUploadChunkResp], error) {
	session, err := s.uploadManager.Get(req.SessionID)
	if err != nil {
		return nil, huma.Error404NotFound("上传会话不存在")
	}
	return response.NewItemResponse(JsUploadChunkResp{
		Success:       session.UploadedChunks[req.Index],
		UploadedBytes: s.uploadManager.UploadedBytes(session),
		UploadedChunk: req.Index,
	}), nil
}

func (s *Service) UploadChunk(_ context.Context, req *UploadChunkReq) (*response.ItemResponse[JsUploadChunkResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsUploadChunkResp{
			Success:       false,
			UploadedChunk: req.Index,
		}), nil
	}

	session, err := s.uploadManager.SaveChunk(req.SessionID, req.Index, req.RawBody)
	if err != nil {
		switch {
		case errors.Is(err, uploadcore.ErrSessionNotFound):
			return nil, huma.Error404NotFound("上传会话不存在")
		case errors.Is(err, uploadcore.ErrChunkOutOfRange):
			return nil, huma.Error400BadRequest("chunk index超出范围")
		case errors.Is(err, uploadcore.ErrChunkEmpty):
			return nil, huma.Error400BadRequest("分块内容不能为空")
		default:
			return nil, huma.Error500InternalServerError("写入分块失败")
		}
	}

	return response.NewItemResponse(JsUploadChunkResp{
		Success:       true,
		UploadedBytes: s.uploadManager.UploadedBytes(session),
		UploadedChunk: req.Index,
	}), nil
}

func (s *Service) CompleteUpload(_ context.Context, req *UploadCompleteReq) (*response.ItemResponse[JsUploadCompleteResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsUploadCompleteResp{
			Success:  false,
			TestMode: true,
		}), nil
	}

	sessionMeta, err := s.uploadManager.Get(req.Body.SessionID)
	if err != nil {
		return nil, huma.Error404NotFound("上传会话不存在")
	}
	session, err := s.uploadManager.Complete(
		req.Body.SessionID,
		filepath.Join(s.dice.BaseConfig.DataDir, "scripts", sessionMeta.Filename),
	)
	if err != nil {
		switch {
		case errors.Is(err, uploadcore.ErrIncomplete):
			return nil, huma.Error400BadRequest("上传分块不完整")
		case errors.Is(err, uploadcore.ErrHashMismatch):
			return nil, huma.Error400BadRequest("文件校验失败")
		default:
			return nil, huma.Error500InternalServerError("完成上传失败")
		}
	}
	s.dice.MarkModified()
	return response.NewItemResponse(JsUploadCompleteResp{
		Success:  true,
		Filename: session.Filename,
	}), nil
}

func (s *Service) SetConfigs(_ context.Context, req *SetConfigsReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if !s.dice.Config.JsEnable {
		return nil, huma.Error400BadRequest("js扩展支持已关闭")
	}
	if req.Body.Name == "" {
		return nil, huma.Error400BadRequest("name不能为空")
	}

	cm := s.dice.ConfigManager
	var errs []error
	for key, val := range req.Body.Config {
		if err := cm.SetConfig(req.Body.Name, key, val); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, huma.Error500InternalServerError("保存配置失败")
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) ResetConfig(_ context.Context, req *ResetConfigReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if !s.dice.Config.JsEnable {
		return nil, huma.Error400BadRequest("js扩展支持已关闭")
	}
	if req.Body.Name == "" {
		return nil, huma.Error400BadRequest("name不能为空")
	}
	cm := s.dice.ConfigManager
	for _, key := range req.Body.Keys {
		cm.ResetConfigToDefault(req.Body.Name, key)
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) DeleteDeadConfigs(_ context.Context, req *DeleteDeadConfigsReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	cm := s.dice.ConfigManager
	for _, name := range req.Body.Names {
		if s.findPluginByName(name) == nil {
			cm.UnregisterConfig(name)
		}
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) DataSet(_ context.Context, req *struct {
	Name string `path:"name"`
	DataSetReq
}) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if req.Body.Key == "" {
		return nil, huma.Error400BadRequest("key不能为空")
	}
	st, err := resolveStorage(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	if err := st.setValue(req.Body.Key, req.Body.Value); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) DataDelete(_ context.Context, req *struct {
	Name string `path:"name"`
	DataDeleteReq
}) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	if len(req.Body.Keys) == 0 {
		return nil, huma.Error400BadRequest("keys不能为空")
	}
	st, err := resolveStorage(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	if err := st.deleteKeys(req.Body.Keys); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

func (s *Service) DataShrink(_ context.Context, req *NamePath) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(JsSimpleResult{Success: false, TestMode: true}), nil
	}
	st, err := resolveStorage(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	if err := st.shrink(); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(JsSimpleResult{Success: true}), nil
}

// ============================================================
// Helpers
// ============================================================

func (s *Service) pluginHasConfig(name string) bool {
	cm := s.dice.ConfigManager
	if cm == nil {
		return false
	}
	_, ok := cm.Plugins[name]
	return ok
}

func (s *Service) findPluginByName(name string) *dice.JsScriptInfo {
	for _, si := range s.dice.JsScriptList {
		if si != nil && si.Name == name {
			return si
		}
	}
	return nil
}

func matchJSKeyword(si *dice.JsScriptInfo, kw string) bool {
	kw = strings.ToLower(kw)
	return strings.Contains(strings.ToLower(si.Name), kw) ||
		strings.Contains(strings.ToLower(si.Desc), kw) ||
		strings.Contains(strings.ToLower(si.Author), kw)
}

func sortJS(items []JsInfo, sortBy, sortOrder string) {
	desc := sortOrder == "desc"
	switch sortBy {
	case "author":
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Author > items[j].Author
			}
			return items[i].Author < items[j].Author
		})
	case "version":
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Version > items[j].Version
			}
			return items[i].Version < items[j].Version
		})
	case "installTime":
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].InstallTime > items[j].InstallTime
			}
			return items[i].InstallTime < items[j].InstallTime
		})
	case "updateTime":
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].UpdateTime > items[j].UpdateTime
			}
			return items[i].UpdateTime < items[j].UpdateTime
		})
	default:
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return strings.ToLower(items[i].Name) > strings.ToLower(items[j].Name)
			}
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		})
	}
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return strings.TrimSpace(name)
}

func extractUploadForm(raw huma.MultipartFormFiles[UploadForm]) *UploadForm {
	data := raw.Data()
	if data != nil && data.File.IsSet {
		return data
	}
	if raw.Form == nil || len(raw.Form.File["file"]) == 0 {
		return nil
	}
	fh := raw.Form.File["file"][0]
	file, err := fh.Open()
	if err != nil {
		return nil
	}
	return &UploadForm{
		File: huma.FormFile{
			File:        file,
			ContentType: fh.Header.Get("Content-Type"),
			IsSet:       true,
			Size:        fh.Size,
			Filename:    fh.Filename,
		},
	}
}
