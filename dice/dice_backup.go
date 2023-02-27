package dice

import (
	"encoding/json"
	"github.com/alexmullins/zip"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sealdice-core/dice/model"
	"time"
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
	dirpath := "./backups"
	_ = os.MkdirAll(dirpath, 0755)

	fzip, _ := ioutil.TempFile(dirpath, bakFilename)
	writer := zip.NewWriter(fzip)
	defer func(writer *zip.Writer) {
		_ = writer.Close()
	}(writer)

	backup := func(d *Dice, fn string) {
		data, err := ioutil.ReadFile(fn)
		if err != nil {
			if d != nil {
				d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
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

			backup(d, filepath.Join(d.BaseConfig.DataDir, "data.db"))
			backup(d, filepath.Join(d.BaseConfig.DataDir, "data-logs.db"))

			//bakTestPath, _ := filepath.Abs("./data-logs-bak.db")
			//model.Backup(d.DBData)
			//backup(d, filepath.Join(d.BaseConfig.DataDir, "data.bdb"))
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
					pa := i.Adapter.(*PlatformAdapterQQOnebot)
					if pa.UseInPackGoCqhttp {
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

	_ = writer.Close()
	_ = fzip.Close()
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
