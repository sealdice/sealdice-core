package censor

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/pelletier/go-toml/v2"

	censorm "sealdice-core/api/v2/model/censor"
	censorcore "sealdice-core/dice/censor"
	"sealdice-core/dice"
	"sealdice-core/dice/service"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/status", s.GetStatus, func(o *huma.Operation) {
		o.Description = "获取拦截管理状态"
	})
	huma.Get(grp, "/config", s.GetConfig, func(o *huma.Operation) {
		o.Description = "获取拦截配置"
	})
	huma.Get(grp, "/words", s.GetWords, func(o *huma.Operation) {
		o.Description = "获取敏感词列表"
	})
	huma.Get(grp, "/files", s.GetFiles, func(o *huma.Operation) {
		o.Description = "获取词库文件列表"
	})
	huma.Get(grp, "/files/template/toml", s.DownloadTomlTemplate, func(o *huma.Operation) {
		o.Description = "下载 TOML 词库模板"
	})
	huma.Get(grp, "/files/template/txt", s.DownloadTxtTemplate, func(o *huma.Operation) {
		o.Description = "下载 TXT 词库模板"
	})
	huma.Get(grp, "/logs/page", s.GetLogsPage, func(o *huma.Operation) {
		o.Description = "分页获取拦截日志"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/restart", s.Restart, func(o *huma.Operation) {
		o.Description = "启用并重载拦截引擎"
	})
	huma.Post(grp, "/stop", s.Stop, func(o *huma.Operation) {
		o.Description = "关闭拦截引擎"
	})
	huma.Post(grp, "/config", s.SetConfig, func(o *huma.Operation) {
		o.Description = "保存拦截配置"
	})
	huma.Post(grp, "/files/upload", s.UploadFile, func(o *huma.Operation) {
		o.Description = "上传词库文件"
	})
	huma.Delete(grp, "/files", s.DeleteFiles, func(o *huma.Operation) {
		o.Description = "删除词库文件"
	})
}

func (s *Service) GetStatus(_ context.Context, _ *request.Empty) (*response.ItemResponse[censorm.CensorStatusResp], error) {
	return response.NewItemResponse(s.buildStatusResp(false)), nil
}

func (s *Service) Restart(_ context.Context, _ *request.Empty) (*response.ItemResponse[censorm.CensorStatusResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(s.buildStatusResp(true)), nil
	}
	s.dice.NewCensorManager()
	s.dice.Config.EnableCensor = true
	s.dice.MarkModified()
	s.dm.Save()
	return response.NewItemResponse(s.buildStatusResp(false)), nil
}

func (s *Service) Stop(_ context.Context, _ *request.Empty) (*response.ItemResponse[censorm.CensorStatusResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(s.buildStatusResp(true)), nil
	}
	s.dice.Config.EnableCensor = false
	s.dice.CensorManager = nil
	s.dice.MarkModified()
	s.dm.Save()
	return response.NewItemResponse(s.buildStatusResp(false)), nil
}

func (s *Service) GetConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[censorm.CensorConfigResp], error) {
	config := s.dice.Config
	return response.NewItemResponse(censorm.CensorConfigResp{
		Mode:          int(config.CensorMode),
		CaseSensitive: config.CensorCaseSensitive,
		MatchPinyin:   config.CensorMatchPinyin,
		FilterRegex:   config.CensorFilterRegexStr,
		LevelConfig: censorm.CensorLevelConfigs{
			Notice:  buildLevelConfig(censorcore.Notice, config.CensorThresholds, config.CensorHandlers, config.CensorScores),
			Caution: buildLevelConfig(censorcore.Caution, config.CensorThresholds, config.CensorHandlers, config.CensorScores),
			Warning: buildLevelConfig(censorcore.Warning, config.CensorThresholds, config.CensorHandlers, config.CensorScores),
			Danger:  buildLevelConfig(censorcore.Danger, config.CensorThresholds, config.CensorHandlers, config.CensorScores),
		},
	}), nil
}

func (s *Service) SetConfig(_ context.Context, req *censorm.CensorConfigReq) (*response.ItemResponse[censorm.CensorConfigResp], error) {
	if s.dice.Config.CensorThresholds == nil {
		s.dice.Config.CensorThresholds = make(map[censorcore.Level]int)
	}
	if s.dice.Config.CensorHandlers == nil {
		s.dice.Config.CensorHandlers = make(map[censorcore.Level]uint8)
	}
	if s.dice.Config.CensorScores == nil {
		s.dice.Config.CensorScores = make(map[censorcore.Level]int)
	}
	body := req.Body.Body
	if _, err := regexp.Compile(body.FilterRegex); err != nil {
		return nil, huma.Error400BadRequest("过滤字符正则不是合法的正则表达式")
	}
	s.dice.Config.CensorMode = dice.CensorMode(body.Mode)
	s.dice.Config.CensorCaseSensitive = body.CaseSensitive
	s.dice.Config.CensorMatchPinyin = body.MatchPinyin
	s.dice.Config.CensorFilterRegexStr = body.FilterRegex

	applyLevelConfig(&s.dice.Config, censorcore.Notice, body.LevelConfig.Notice)
	applyLevelConfig(&s.dice.Config, censorcore.Caution, body.LevelConfig.Caution)
	applyLevelConfig(&s.dice.Config, censorcore.Warning, body.LevelConfig.Warning)
	applyLevelConfig(&s.dice.Config, censorcore.Danger, body.LevelConfig.Danger)

	s.dice.MarkModified()
	s.dm.Save()
	return s.GetConfig(context.Background(), &request.Empty{})
}

func (s *Service) GetWords(_ context.Context, _ *request.Empty) (*response.ItemResponse[censorm.CensorWordsResp], error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	temp := map[string]*censorm.CensorWordItem{}
	for word, info := range s.dice.CensorManager.Censor.SensitiveKeys {
		switch info.Reason {
		case censorcore.Origin:
			if _, ok := temp[word]; !ok {
				temp[word] = &censorm.CensorWordItem{
					Main:  word,
					Level: int(info.Level),
				}
			}
		case censorcore.IgnoreCase, censorcore.PinYin:
			sensitiveWord, ok := temp[info.Origin]
			if !ok {
				temp[info.Origin] = &censorm.CensorWordItem{
					Main:  info.Origin,
					Level: int(info.Level),
				}
				sensitiveWord = temp[info.Origin]
			}
			sensitiveWord.Related = append(sensitiveWord.Related, censorm.CensorRelatedWord{
				Word:   word,
				Reason: int(info.Reason),
			})
		}
	}
	data := make([]*censorm.CensorWordItem, 0, len(temp))
	for _, word := range temp {
		sort.Slice(word.Related, func(i, j int) bool {
			if word.Related[i].Reason == word.Related[j].Reason {
				return word.Related[i].Word < word.Related[j].Word
			}
			return word.Related[i].Reason < word.Related[j].Reason
		})
		data = append(data, word)
	}
	sort.Slice(data, func(i, j int) bool {
		if data[i].Level == data[j].Level {
			return data[i].Main < data[j].Main
		}
		return data[i].Level < data[j].Level
	})
	return response.NewItemResponse(censorm.CensorWordsResp{Data: data}), nil
}

func (s *Service) GetFiles(_ context.Context, _ *request.Empty) (*response.ItemResponse[censorm.CensorFilesResp], error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	files := make([]*censorm.CensorFileInfo, 0, len(s.dice.CensorManager.SensitiveWordsFiles))
	for _, f := range s.dice.CensorManager.SensitiveWordsFiles {
		files = append(files, &censorm.CensorFileInfo{
			Key:      f.Key,
			Count:    f.FileCounter,
			FileType: f.FileType,
			Name:     f.Name,
			Author:   strings.Join(f.Authors, " / "),
			Version:  f.Version,
			Desc:     f.Desc,
			License:  f.License,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return response.NewItemResponse(censorm.CensorFilesResp{Data: files}), nil
}

func (s *Service) UploadFile(_ context.Context, req *censorm.CensorUploadReq) (*response.ItemResponse[censorm.CensorSimpleResp], error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	form, err := extractUploadForm(req.RawBody)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = form.File.Close()
	}()
	filename := sanitizeUploadFilename(form.File.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("文件名不能为空")
	}
	if !isAllowedWordFile(filename) {
		return nil, huma.Error400BadRequest("仅支持 txt 和 toml 词库文件")
	}
	if err := os.MkdirAll("./data/censor", 0o755); err != nil {
		return nil, huma.Error500InternalServerError("创建词库目录失败")
	}
	dst, err := os.Create(filepath.Join("./data/censor", filename))
	if err != nil {
		return nil, huma.Error500InternalServerError("创建词库文件失败")
	}
	defer func() {
		_ = dst.Close()
	}()
	if _, err = io.Copy(dst, form.File); err != nil {
		return nil, huma.Error500InternalServerError("写入词库文件失败")
	}
	return response.NewItemResponse(censorm.CensorSimpleResp{Success: true}), nil
}

func (s *Service) DeleteFiles(_ context.Context, req *censorm.CensorDeleteFilesReq) (*response.ItemResponse[censorm.CensorSimpleResp], error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	s.dice.CensorManager.DeleteCensorWordFiles(req.Body.Body.Keys)
	return response.NewItemResponse(censorm.CensorSimpleResp{Success: true}), nil
}

func (s *Service) DownloadTomlTemplate(_ context.Context, _ *request.Empty) (*huma.StreamResponse, error) {
	buf := &bytes.Buffer{}
	template := censorcore.TomlCensorWordFile{
		Meta: censorcore.TomlMeta{
			Name:    "测试词库",
			Authors: []string{"<匿名>"},
			Version: "1.0",
			Desc:    "一个测试词库",
			License: "CC-BY-NC-SA 4.0",
		},
		Words: censorcore.TomlWords{
			Notice:  []string{"提醒级词汇1", "提醒级词汇2"},
			Caution: []string{"注意级词汇1", "注意级词汇2"},
			Warning: []string{"警告级词汇1", "警告级词汇2"},
			Danger:  []string{"危险级词汇1", "危险级词汇2"},
		},
	}
	if err := toml.NewEncoder(buf).SetArraysMultiline(true).Encode(&template); err != nil {
		return nil, huma.Error500InternalServerError("生成模板失败", err)
	}
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Cache-Control", "no-store")
			ctx.SetHeader("Content-Type", "application/octet-stream")
			ctx.SetHeader("Content-Disposition", "attachment; filename=\"词库模板.toml\"")
			_, _ = ctx.BodyWriter().Write(buf.Bytes())
		},
	}, nil
}

func (s *Service) DownloadTxtTemplate(_ context.Context, _ *request.Empty) (*huma.StreamResponse, error) {
	content := []byte(`#notice
提醒级词汇1
提醒级词汇2
#caution
注意级词汇1
注意级词汇2
#warning
警告级词汇
#danger
危险级词汇
`)
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Cache-Control", "no-store")
			ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
			ctx.SetHeader("Content-Disposition", "attachment; filename=\"词库模板.txt\"")
			_, _ = ctx.BodyWriter().Write(content)
		},
	}, nil
}

func (s *Service) GetLogsPage(_ context.Context, req *censorm.CensorLogPageQuery) (*response.ItemResponse[censorm.CensorLogPageResp], error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	pageNum := req.PageNum
	if pageNum < 1 {
		pageNum = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	total, page, err := service.CensorGetLogPage(s.dice.DBOperator, service.QueryCensorLog{
		PageNum:  pageNum,
		PageSize: pageSize,
		UserID:   strings.TrimSpace(req.UserID),
		Level:    req.Level,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("获取拦截日志失败", err)
	}
	return response.NewItemResponse(censorm.CensorLogPageResp{
		Data:     page,
		Total:    total,
		PageNum:  pageNum,
		PageSize: len(page),
	}), nil
}

func (s *Service) buildStatusResp(testMode bool) censorm.CensorStatusResp {
	isLoading := false
	if s.dice.CensorManager != nil {
		isLoading = s.dice.CensorManager.IsLoading
	}
	return censorm.CensorStatusResp{
		Enable:    s.dice.Config.EnableCensor,
		IsLoading: isLoading,
		TestMode:  testMode,
	}
}

func (s *Service) checkReady() error {
	if !s.dice.Config.EnableCensor {
		return huma.Error400BadRequest("未启用拦截引擎")
	}
	if s.dice.CensorManager == nil {
		return huma.Error400BadRequest("拦截引擎未初始化")
	}
	if s.dice.CensorManager.IsLoading {
		return huma.Error409Conflict("拦截引擎正在加载，请稍候")
	}
	return nil
}

func buildLevelConfig(
	level censorcore.Level,
	thresholds map[censorcore.Level]int,
	handlers map[censorcore.Level]uint8,
	scores map[censorcore.Level]int,
) censorm.CensorLevelConfig {
	handler := handlers[level]
	items := make([]string, 0, 6)
	if handler&(1<<dice.SendWarning) != 0 {
		items = append(items, dice.CensorHandlerText[dice.SendWarning])
	}
	if handler&(1<<dice.SendNotice) != 0 {
		items = append(items, dice.CensorHandlerText[dice.SendNotice])
	}
	if handler&(1<<dice.BanUser) != 0 {
		items = append(items, dice.CensorHandlerText[dice.BanUser])
	}
	if handler&(1<<dice.BanGroup) != 0 {
		items = append(items, dice.CensorHandlerText[dice.BanGroup])
	}
	if handler&(1<<dice.BanInviter) != 0 {
		items = append(items, dice.CensorHandlerText[dice.BanInviter])
	}
	if handler&(1<<dice.AddScore) != 0 {
		items = append(items, dice.CensorHandlerText[dice.AddScore])
	}
	return censorm.CensorLevelConfig{
		Threshold: thresholds[level],
		Handlers:  items,
		Score:     scores[level],
	}
}

func applyLevelConfig(config *dice.Config, level censorcore.Level, input censorm.CensorLevelConfig) {
	config.CensorThresholds[level] = input.Threshold
	config.CensorScores[level] = input.Score
	setLevelHandlers(config, level, input.Handlers)
}

func setLevelHandlers(config *dice.Config, level censorcore.Level, handlers []string) {
	newHandlers := map[dice.CensorHandler]bool{}
	for _, item := range handlers {
		switch item {
		case "SendWarning":
			newHandlers[dice.SendWarning] = true
		case "SendNotice":
			newHandlers[dice.SendNotice] = true
		case "BanUser":
			newHandlers[dice.BanUser] = true
		case "BanGroup":
			newHandlers[dice.BanGroup] = true
		case "BanInviter":
			newHandlers[dice.BanInviter] = true
		case "AddScore":
			newHandlers[dice.AddScore] = true
		}
	}
	var value uint8
	value = nextHandlerValue(value, dice.SendWarning, newHandlers)
	value = nextHandlerValue(value, dice.SendNotice, newHandlers)
	value = nextHandlerValue(value, dice.BanUser, newHandlers)
	value = nextHandlerValue(value, dice.BanGroup, newHandlers)
	value = nextHandlerValue(value, dice.BanInviter, newHandlers)
	value = nextHandlerValue(value, dice.AddScore, newHandlers)
	config.CensorHandlers[level] = value
}

func nextHandlerValue(value uint8, handler dice.CensorHandler, newHandlers map[dice.CensorHandler]bool) uint8 {
	if newHandlers[handler] {
		value |= 1 << handler
		return value
	}
	value &^= 1 << handler
	return value
}

func sanitizeUploadFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return strings.TrimSpace(name)
}

func isAllowedWordFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt", ".toml":
		return true
	default:
		return false
	}
}

func extractUploadForm(raw huma.MultipartFormFiles[censorm.CensorUploadForm]) (*censorm.CensorUploadForm, error) {
	data := raw.Data()
	if data != nil && data.File.IsSet {
		return data, nil
	}
	if raw.Form == nil || len(raw.Form.File["file"]) == 0 {
		return nil, huma.Error400BadRequest("missing file")
	}
	fh := raw.Form.File["file"][0]
	file, err := fh.Open()
	if err != nil {
		return nil, huma.Error400BadRequest("failed to open file")
	}
	return &censorm.CensorUploadForm{
		File: huma.FormFile{
			File:        file,
			ContentType: fh.Header.Get("Content-Type"),
			IsSet:       true,
			Size:        fh.Size,
			Filename:    fh.Filename,
		},
	}, nil
}
