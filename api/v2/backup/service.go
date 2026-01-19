package backup

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	backupm "sealdice-core/api/v2/model/backup"
	cmm "sealdice-core/api/v2/model/common"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/crypto"
)

type BackupService struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewBackupService(dm *dice.DiceManager) *BackupService {
	return &BackupService{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *BackupService) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/list", s.GetList, func(o *huma.Operation) {
		o.Description = "获取备份文件列表"
		o.Summary = "获取备份文件列表"
	})
	huma.Get(grp, "/download", s.Download, func(o *huma.Operation) {
		o.Description = "下载备份文件（流式附件下载）"
		o.Summary = "下载备份文件"
	})
	huma.Get(grp, "/config", s.ConfigGet, func(o *huma.Operation) {
		o.Description = "获取备份配置"
		o.Summary = "获取备份配置"
	})
}

func (s *BackupService) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/delete", s.Delete, func(o *huma.Operation) {
		o.Description = "删除备份文件"
	})
	huma.Post(grp, "/batch/delete", s.BatchDelete, func(o *huma.Operation) {
		o.Description = "批量删除备份文件"
	})
	huma.Post(grp, "/exec", s.Exec, func(o *huma.Operation) {
		o.Description = "执行快速备份"
	})
	huma.Post(grp, "/config/save", s.ConfigSave, func(o *huma.Operation) {
		o.Description = "保存备份配置"
	})
}

func (s *BackupService) GetList(_ context.Context, _ *request.Empty) (*response.ItemResponse[backupm.FileListResp], error) {
	reFn := regexp.MustCompile(`^(bak_\d{6}_\d{6}(?:_auto)?_r([0-9a-f]+))_([0-9a-f]{8})\.zip$`)
	var items []*backupm.FileItem
	_ = filepath.Walk(dice.BackupDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && !info.IsDir() {
			fn := info.Name()
			matches := reFn.FindStringSubmatch(fn)
			selection := int64(0)
			if len(matches) == 4 {
				hashed := crypto.CalculateSHA512Str([]byte(matches[1]))
				if hashed[:8] == matches[3] {
					selection, _ = strconv.ParseInt(matches[2], 16, 64)
				} else {
					selection = -1
				}
			}
			items = append(items, &backupm.FileItem{
				Name:      fn,
				FileSize:  info.Size(),
				Selection: selection,
			})
		}
		return nil
	})
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return response.NewItemResponse[backupm.FileListResp](backupm.FileListResp{Items: items}), nil
}

func (s *BackupService) Download(_ context.Context, ol *backupm.DownloadReq) (*huma.StreamResponse, error) {
	if s.dm.JustForTest {
		return &huma.StreamResponse{
			Body: func(ctx huma.Context) {
				ctx.SetHeader("Content-Type", "application/json")
				_, _ = ctx.BodyWriter().Write([]byte(`{"testMode": true}`))
			},
		}, nil
	}
	name := ol.Name
	if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
		fp := filepath.Join(dice.BackupDir, name)
		info, err := os.Stat(fp)
		if err != nil || info.IsDir() {
			return nil, huma.Error404NotFound("not found")
		}
		return &huma.StreamResponse{
			Body: func(ctx huma.Context) {
				ctx.SetHeader("Content-Type", "application/zip")
				ctx.SetHeader("Content-Disposition", "attachment; filename=\""+name+"\"")
				ctx.SetHeader("Content-Length", strconv.FormatInt(info.Size(), 10))
				f, err := os.Open(fp)
				if err != nil {
					return
				}
				defer func() { _ = f.Close() }()
				w := ctx.BodyWriter()
				buf := make([]byte, 64*1024)
				for {
					n, rerr := f.Read(buf)
					if n > 0 {
						_, _ = w.Write(buf[:n])
						if fl, ok := w.(http.Flusher); ok {
							fl.Flush()
						}
					}
					if rerr != nil {
						break
					}
				}
			},
		}, nil
	}
	return nil, huma.Error400BadRequest("invalid name")
}

func (s *BackupService) Delete(_ context.Context, ol *request.RequestWrapper[backupm.BkupDeleteReq]) (*response.ItemResponse[cmm.SimpleOK], error) {
	name := ol.Body.Name
	var err error
	if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
		err = os.Remove(filepath.Join(dice.BackupDir, name))
	}
	return response.NewItemResponse[cmm.SimpleOK](cmm.SimpleOK{Success: err == nil}), nil
}

func (s *BackupService) BatchDelete(_ context.Context, ol *request.RequestWrapper[cmm.NameListReq]) (*response.ItemResponse[cmm.BatchDeleteResp], error) {
	v := ol.Body
	fails := make([]string, 0, len(v.Names))
	for _, name := range v.Names {
		if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
			err := os.Remove(filepath.Join(dice.BackupDir, name))
			if err != nil {
				fails = append(fails, name)
			}
		}
	}
	return response.NewItemResponse[cmm.BatchDeleteResp](cmm.BatchDeleteResp{Fails: fails}), nil
}

func (s *BackupService) Exec(_ context.Context, ol *request.RequestWrapper[backupm.ExecReq]) (*response.ItemResponse[cmm.SimpleOK], error) {
	_, err := s.dm.Backup(dice.BackupSelection(ol.Body.Selection), false)
	return response.NewItemResponse[cmm.SimpleOK](cmm.SimpleOK{Success: err == nil}), nil
}

func (s *BackupService) ConfigGet(_ context.Context, _ *request.Empty) (*response.ItemResponse[backupm.Config], error) {
	bc := backupm.Config{}
	bc.AutoBackupEnable = s.dm.AutoBackupEnable
	bc.AutoBackupTime = s.dm.AutoBackupTime
	bc.AutoBackupSelection = uint64(s.dm.AutoBackupSelection)
	bc.BackupCleanStrategy = int(s.dm.BackupCleanStrategy)
	bc.BackupCleanKeepCount = s.dm.BackupCleanKeepCount
	bc.BackupCleanKeepDur = s.dm.BackupCleanKeepDur.String()
	bc.BackupCleanTrigger = int(s.dm.BackupCleanTrigger)
	bc.BackupCleanCron = s.dm.BackupCleanCron
	return response.NewItemResponse[backupm.Config](bc), nil
}

func (s *BackupService) ConfigSave(_ context.Context, ol *request.RequestWrapper[backupm.Config]) (*response.ItemResponse[struct{}], error) {
	v := ol.Body
	s.dm.AutoBackupEnable = v.AutoBackupEnable
	s.dm.AutoBackupTime = v.AutoBackupTime
	s.dm.AutoBackupSelection = dice.BackupSelection(v.AutoBackupSelection)
	if int(dice.BackupCleanStrategyDisabled) <= v.BackupCleanStrategy && v.BackupCleanStrategy <= int(dice.BackupCleanStrategyByTime) {
		s.dm.BackupCleanStrategy = dice.BackupCleanStrategy(v.BackupCleanStrategy)
		if s.dm.BackupCleanStrategy == dice.BackupCleanStrategyByCount && v.BackupCleanKeepCount > 0 {
			s.dm.BackupCleanKeepCount = v.BackupCleanKeepCount
		}
		if s.dm.BackupCleanStrategy == dice.BackupCleanStrategyByTime && len(v.BackupCleanKeepDur) > 0 {
			if dur, err := time.ParseDuration(v.BackupCleanKeepDur); err == nil {
				s.dm.BackupCleanKeepDur = dur
			}
		}
		if v.BackupCleanTrigger > 0 {
			s.dm.BackupCleanTrigger = dice.BackupCleanTrigger(v.BackupCleanTrigger)
			if s.dm.BackupCleanTrigger&dice.BackupCleanTriggerCron > 0 {
				s.dm.BackupCleanCron = v.BackupCleanCron
			}
		}
	}
	s.dm.ResetAutoBackup()
	s.dm.ResetBackupClean()
	s.dm.Save()
	return response.NewItemResponse[struct{}](struct{}{}), nil
}
