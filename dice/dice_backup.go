package dice

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexmullins/zip"

	"sealdice-core/dice/model"
	"sealdice-core/utils"
	"sealdice-core/utils/crypto"
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

type backupConfigGlobal struct {
	Global  bool                         `json:"global"`
	Decks   bool                         `json:"decks"`
	HelpDoc bool                         `json:"helpDoc"`
	Censor  bool                         `json:"censor"`
	Names   bool                         `json:"names"`
	Images  bool                         `json:"images"`
	Dices   map[string]*backupConfigDice `json:"dices"`
}

type backupConfigDice struct {
	Accounts    bool `json:"accounts"`    // 帐号
	MiscConfig  bool `json:"miscConfig"`  // 综合设置
	PlayerData  bool `json:"playerData"`  // 用户数据
	CustomReply bool `json:"customReply"` // 文案模板
	CustomText  bool `json:"customText"`  // 自定义回复
	JSScripts   bool `json:"jsScripts"`   // JS脚本
}

type BackupSelection uint64

const (
	BackupSelectionJS BackupSelection = 1 << iota
	BackupSelectionDecks
	BackupSelectionHelpDoc
	BackupSelectionCensor
	BackupSelectionNames
	BackupSelectionImages

	BackupSelectionBasic     BackupSelection = 0
	BackupSelectionResources BackupSelection = BackupSelectionImages
	BackupSelectionAll       BackupSelection = BackupSelectionBasic |
		BackupSelectionJS |
		BackupSelectionDecks |
		BackupSelectionHelpDoc |
		BackupSelectionCensor |
		BackupSelectionNames |
		BackupSelectionResources
)

func (dm *DiceManager) Backup(sel BackupSelection, fromAuto bool) (string, error) {
	_ = os.MkdirAll(BackupDir, 0o755)
	logger := dm.Dice[0].Logger

	cfgGlb := backupConfigGlobal{
		Global: true,
		Dices:  map[string]*backupConfigDice{},
	}
	cfgDice := backupConfigDice{
		Accounts:    true,
		MiscConfig:  true,
		PlayerData:  true,
		CustomReply: true,
		CustomText:  true,
	}

	bakFn := "bak_" + time.Now().Format("060102_150405")
	if fromAuto {
		bakFn += "_auto"
	}
	bakFn += "_r" + strconv.FormatUint(uint64(sel), 16)
	fnHashed := crypto.CalculateSHA512Str([]byte(bakFn))[:8]
	bakFn += "_" + fnHashed + ".zip"

	fzip, err := os.OpenFile(filepath.Join(BackupDir, bakFn),
		os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer func() { _ = fzip.Close() }()

	writer := zip.NewWriter(fzip)
	defer func(writer *zip.Writer) {
		_ = writer.Close()
	}(writer)

	fileOK := func(fn string) bool {
		stat, err := os.Stat(fn)
		return err == nil && !stat.IsDir()
	}
	dirOK := func(fn string) bool {
		stat, err := os.Stat(fn)
		return err == nil && stat.IsDir()
	}

	backup := func(d *Dice, fn string) {
		data, err := os.ReadFile(fn)
		if err != nil && !strings.Contains(fn, "session.token") {
			if d != nil {
				d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			} else {
				logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			}
			return
		}

		h := &zip.FileHeader{Name: fn, Method: zip.Deflate, Flags: 0x800}
		fileWriter, err := writer.CreateHeader(h)
		if err != nil {
			if d != nil {
				d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			} else {
				logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			}
			return
		}
		_, _ = fileWriter.Write(data)
	}

	backupDir := func(path string, info fs.FileInfo, _ error) error {
		if !info.IsDir() {
			backup(nil, path)
		}
		return nil
	}

	backup(nil, "data/dice.yaml")

	if sel&BackupSelectionDecks != 0 {
		cfgGlb.Decks = true
		_ = filepath.Walk("data/decks", func(path string, info fs.FileInfo, _ error) error {
			if !info.IsDir() {
				backup(nil, path)
				return nil
			}
			base := filepath.Base(path)
			// 跳过 deck 压缩包解压出的目录
			if strings.HasPrefix(base, "_") && strings.HasSuffix(base, ".deck") {
				if fileOK(filepath.Join(filepath.Dir(path), base[1:])) {
					return filepath.SkipDir
				}
			}
			return nil
		})
	}

	if sel&BackupSelectionHelpDoc != 0 {
		if !dirOK("data/helpdoc") {
			logger.Warn("备份 helpdoc 失败: 不存在或不是目录")
		} else {
			cfgGlb.HelpDoc = true
			_ = filepath.Walk("data/helpdoc", backupDir)
		}
	}

	if sel&BackupSelectionCensor != 0 {
		if !dirOK("data/censor") {
			logger.Warn("备份 censor 失败: 不存在或不是目录")
		} else {
			cfgGlb.Censor = true
			_ = filepath.Walk("data/censor", backupDir)
		}
	}

	if sel&BackupSelectionNames != 0 {
		if !dirOK("data/names") {
			logger.Warn("备份 names 失败: 不存在或不是目录")
		} else {
			cfgGlb.Names = true
			_ = filepath.Walk("data/names", backupDir)
		}
	}

	if sel&BackupSelectionImages != 0 {
		if !dirOK("data/images") {
			logger.Warn("备份 images 失败: 不存在或不是目录")
		} else {
			cfgGlb.Images = true
			_ = filepath.Walk("data/images", backupDir)
		}
	}

	withJS := sel&BackupSelectionJS != 0
	cfgDice.JSScripts = withJS

	for _, d := range dm.Dice {
		cfgGlb.Dices[d.BaseConfig.Name] = &cfgDice
		dataDir := d.BaseConfig.DataDir

		backup(d, filepath.Join(dataDir, "serve.yaml"))
		if fn := filepath.Join(dataDir, "advanced.yaml"); fileOK(fn) {
			backup(d, fn)
		}
		if fn := filepath.Join(dataDir, "configs", "plugin-configs.json"); fileOK(fn) {
			backup(d, fn)
		}

		err := model.FlushWAL(d.DBData)
		if err != nil {
			d.Logger.Errorf("备份时data数据库flush出错 错误为:%v", err.Error())
		} else {
			backup(d, filepath.Join(dataDir, "data.db"))
		}
		err = model.FlushWAL(d.DBLogs)
		if err != nil {
			d.Logger.Errorf("备份时logs数据库flush出错 错误为:%v", err.Error())
		} else {
			backup(d, filepath.Join(dataDir, "data-logs.db"))
		}
		if d.CensorManager != nil && d.CensorManager.DB != nil {
			err = model.FlushWAL(d.CensorManager.DB)
			if err != nil {
				d.Logger.Errorf("备份时censor数据库flush出错 %v", err.Error())
			} else {
				backup(d, filepath.Join(dataDir, "data-censor.db"))
			}
		}

		backup(d, filepath.Join(dataDir, "configs/text-template.yaml"))

		_ = filepath.WalkDir(filepath.Join(dataDir, "extensions/reply"), func(path string, info fs.DirEntry, _ error) error {
			// NOTE(Xiangze Li): copied from dice.ReplyReload. Should extract as function, but I'm lazy
			if info.IsDir() {
				if strings.EqualFold(info.Name(), "assets") || strings.EqualFold(info.Name(), "images") {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(info.Name(), ".reply") || info.Name() == "info.yaml" {
				return nil
			}

			ext := filepath.Ext(path)
			if ext == ".yaml" || ext == "" {
				backup(d, path)
			}
			return nil
		})

		for _, i := range d.ImSession.EndPoints {
			if i.Platform == "QQ" {
				if pa, ok := i.Adapter.(*PlatformAdapterGocq); ok && pa.UseInPackClient {
					workDir := i.RelWorkDir
					if pa.BuiltinMode == "lagrange" {
						backup(d, filepath.Join(dataDir, workDir, "appsettings.json"))
						backup(d, filepath.Join(dataDir, workDir, "device.json"))
						backup(d, filepath.Join(dataDir, workDir, "keystore.json"))
					} else {
						backup(d, filepath.Join(dataDir, workDir, "config.yml"))
						backup(d, filepath.Join(dataDir, workDir, "device.json"))
						backup(d, filepath.Join(dataDir, workDir, "session.token"))
					}
				}
			}
		}

		if withJS {
			_ = filepath.WalkDir(filepath.Join(dataDir, "scripts"), func(path string, info fs.DirEntry, _ error) error {
				if info.IsDir() {
					if info.Name() == "_builtin" {
						return filepath.SkipDir
					}
					return nil
				}
				if filepath.Ext(info.Name()) == ".js" {
					backup(d, path)
				}
				return nil
			})
			extDataDir := filepath.Join(dataDir, "extensions")
			_ = filepath.WalkDir(extDataDir, func(path string, info fs.DirEntry, err error) error {
				if info.IsDir() {
					if filepath.Dir(path) == extDataDir {
						if ext := d.ExtFind(info.Name()); ext == nil || !ext.IsJsExt {
							return filepath.SkipDir
						}
					}
					return nil
				}
				backup(d, path)
				return nil
			})
		}
	}

	// 写入文件信息
	data, _ := json.Marshal(map[string]interface{}{
		"config":      cfgGlb,
		"version":     VERSION.String(),
		"versionCode": VERSION_CODE,
	})

	h := &zip.FileHeader{Name: "backup_info.json", Method: zip.Deflate, Flags: 0x800}
	fileWriter, _ := writer.CreateHeader(h)
	_, _ = fileWriter.Write(data)

	return fzip.Name(), nil
}

func (dm *DiceManager) BackupAuto() error {
	_, err := dm.Backup(dm.AutoBackupSelection, true)
	return err
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
