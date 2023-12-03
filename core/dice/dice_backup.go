package dice

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sealdice-core/dice/model"
	"sealdice-core/utils"
	"sort"
	"strings"
	"time"

	"github.com/alexmullins/zip"
)

const BackupDir = "./backups"

type BackupCleanStrategy int

const (
	BackupCleanStrategyDisabled BackupCleanStrategy = iota
	BackupCleanStrategyByCount
	BackupCleanStrategyByTime
)

type BackupCleanTrigger int

const (
	// BackupCleanTriggerCron 通过独立定时任务触发
	BackupCleanTriggerCron BackupCleanTrigger = 1 << iota
	// BackupCleanTriggerRotate 通过自动备份触发
	BackupCleanTriggerRotate
)

// 可勾选自定义文本、自定义回复、QQ帐号信息、牌堆等

type AllBackupConfig struct {
	Decks   bool                        `json:"decks"`
	HelpDoc bool                        `json:"helpDoc"`
	Dices   map[string]*OneBackupConfig `json:"dices"`
	Global  bool                        `json:"global"`
}

type OneBackupConfig struct {
	MiscConfig  bool `json:"miscConfig"`  // 综合设置
	PlayerData  bool `json:"playerData"`  // 用户数据
	CustomReply bool `json:"customReply"` // 自定义回复
	CustomText  bool `json:"customText"`  // 自定义文本
	Accounts    bool `json:"accounts"`    // 帐号
}

func (dm *DiceManager) Backup(cfg AllBackupConfig, bakFilename string) (string, error) {
	_ = os.MkdirAll(BackupDir, 0755)

	fzip, err := os.CreateTemp(BackupDir, bakFilename)
	if err != nil {
		return "", err
	}
	defer func() { _ = fzip.Close() }()
	writer := zip.NewWriter(fzip)
	defer func(writer *zip.Writer) {
		_ = writer.Close()
	}(writer)

	backup := func(d *Dice, fn string) {
		data, err := os.ReadFile(fn)
		if err != nil {
			if d != nil {
				if !strings.Contains(fn, "session.token") {
					d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
				}
			}
			return
		}

		h := &zip.FileHeader{Name: fn, Method: zip.Deflate, Flags: 0x800}
		fileWriter, err := writer.CreateHeader(h)
		if err != nil {
			if d != nil {
				d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			}
			return
		}
		_, _ = fileWriter.Write(data)
	}

	if cfg.Decks {
		_ = filepath.Walk("data/decks", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				backup(nil, path)
			}
			return nil
		})
	}

	if cfg.HelpDoc {
		_ = filepath.Walk("data/helpdoc", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				backup(nil, path)
			}
			return nil
		})
	}

	if cfg.Global {
		backup(nil, "data/dice.yaml")
	}

	for k, cfg2 := range cfg.Dices {
		var d *Dice
		for _, i := range dm.Dice {
			if i.BaseConfig.Name == k {
				d = dm.Dice[0]
				break
			}
		}

		if d == nil {
			continue
		}

		if cfg2.MiscConfig {
			backup(d, filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))
		}

		if cfg2.PlayerData {
			err := model.FlushWAL(d.DBData)
			if err != nil {
				d.Logger.Warnln("备份时data数据库flush出错", err.Error())
			}
			err = model.FlushWAL(d.DBLogs)
			if err != nil {
				d.Logger.Warnln("备份时logs数据库flush出错", err.Error())
			}
			if d.CensorManager != nil && d.CensorManager.DB != nil {
				err = model.FlushWAL(d.CensorManager.DB)
				if err != nil {
					d.Logger.Warnln("备份时censor数据库flush出错", err.Error())
				}
			}

			backup(d, filepath.Join(d.BaseConfig.DataDir, "data.db"))
			backup(d, filepath.Join(d.BaseConfig.DataDir, "data-logs.db"))
			if _, err = os.Stat(filepath.Join(d.BaseConfig.DataDir, "data-censor.db")); err == nil {
				backup(d, filepath.Join(d.BaseConfig.DataDir, "data-censor.db"))
			}

			// bakTestPath, _ := filepath.Abs("./data-logs-bak.db")
			// model.Backup(d.DBData)
			// backup(d, filepath.Join(d.BaseConfig.DataDir, "data.bdb"))
		}
		if cfg2.CustomReply {
			backup(d, filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml"))
		}
		if cfg2.CustomText {
			backup(d, filepath.Join(d.BaseConfig.DataDir, "extensions/reply/reply.yaml"))
		}
		if cfg2.Accounts {
			for _, i := range d.ImSession.EndPoints {
				if i.Platform == "QQ" {
					if pa, ok := i.Adapter.(*PlatformAdapterGocq); ok && pa.UseInPackGoCqhttp {
						backup(d, filepath.Join(d.BaseConfig.DataDir, i.RelWorkDir, "config.yml"))
						backup(d, filepath.Join(d.BaseConfig.DataDir, i.RelWorkDir, "device.json"))
						backup(d, filepath.Join(d.BaseConfig.DataDir, i.RelWorkDir, "session.token"))
					}
				}
			}
		}
	}

	// 写入文件信息
	data, _ := json.Marshal(map[string]interface{}{
		"config":      cfg,
		"version":     VERSION,
		"versionCode": VERSION_CODE,
	})

	h := &zip.FileHeader{Name: "backup_info.json", Method: zip.Deflate, Flags: 0x800}
	fileWriter, _ := writer.CreateHeader(h)
	_, _ = fileWriter.Write(data)

	return fzip.Name(), nil
}

func (dm *DiceManager) BackupAuto() error {
	_, err := dm.Backup(AllBackupConfig{
		Global:  true,
		Decks:   false,
		HelpDoc: false,
		Dices: map[string]*OneBackupConfig{
			"default": {
				MiscConfig:  true,
				PlayerData:  true,
				CustomReply: true,
				CustomText:  true,
				Accounts:    true,
			},
		},
	}, "bak-"+time.Now().Format("060102_150405")+"_auto_"+"*.zip")
	return err
}

func (dm *DiceManager) BackupSimple() (string, error) {
	fn := "bak-" + time.Now().Format("060102_150405") + "_" + "*.zip"
	return dm.Backup(AllBackupConfig{
		Global:  true,
		Decks:   false,
		HelpDoc: false,
		Dices: map[string]*OneBackupConfig{
			"default": {
				MiscConfig:  true,
				PlayerData:  true,
				CustomReply: true,
				CustomText:  true,
				Accounts:    true,
			},
		},
	}, fn)
}

func (dm *DiceManager) BackupClean(fromAuto bool) (err error) {
	if dm.BackupCleanStrategy == BackupCleanStrategyDisabled {
		return nil
	}

	if fromAuto && (dm.BackupCleanTrigger&BackupCleanTriggerRotate == 0) {
		return nil
	}

	// fmt.Println("开始定时清理备份", fromAuto)

	backupDir, err := os.Open(BackupDir)
	if err != nil {
		return err
	}
	defer func() { _ = backupDir.Close() }()
	if i, _ := backupDir.Stat(); !i.IsDir() {
		return fmt.Errorf("backup directory %q is not a directory", BackupDir)
	}

	files, err := backupDir.ReadDir(-1)
	if err != nil {
		return err
	}

	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if fi, err := f.Info(); err == nil {
			fileInfos = append(fileInfos, fi)
		}
	}

	sort.Sort(utils.ByModtime(fileInfos))

	var fileInfoOld []os.FileInfo
	switch dm.BackupCleanStrategy {
	case BackupCleanStrategyByCount:
		if len(fileInfos) > dm.BackupCleanKeepCount {
			fileInfoOld = fileInfos[:len(fileInfos)-dm.BackupCleanKeepCount]
		}
	case BackupCleanStrategyByTime:
		threshold := time.Now().Add(-dm.BackupCleanKeepDur)
		idx, _ := sort.Find(len(fileInfos), func(i int) int {
			return threshold.Compare(fileInfos[i].ModTime())
		})
		fileInfoOld = fileInfos[:idx+1]
	default:
		// no-op
	}

	errDel := []string{}
	for _, fi := range fileInfoOld {
		errDelete := os.Remove(filepath.Join(BackupDir, fi.Name()))
		if errDelete != nil {
			errDel = append(errDel, errDelete.Error())
		}
	}

	if len(errDel) > 0 {
		return fmt.Errorf("error(s) occured when deleting files:\n" + strings.Join(errDel, "\n"))
	}
	return nil
}
