package dice

import (
	"github.com/alexmullins/zip"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// 可勾选自定义文本、自定义回复、QQ帐号信息、牌堆等

type AllBackupConfig struct {
	Decks   bool                        `json:"decks"`
	HelpDoc bool                        `json:"helpDoc"`
	Dices   map[string]*OneBackupConfig `json:"dices"`
}

type OneBackupConfig struct {
	MiscConfig  bool `json:"miscConfig"`  // 综合设置
	PlayerData  bool `json:"playerData"`  // 用户数据
	CustomReply bool `json:"customReply"` // 自定义回复
	CustomText  bool `json:"customText"`  // 自定义文本
	Accounts    bool `json:"accounts"`    // 帐号
}

func (dm *DiceManager) Backup(cfg AllBackupConfig, bakFilename string) error {
	dirpath := "./backups"
	os.MkdirAll(dirpath, 0755)

	fzip, _ := ioutil.TempFile(dirpath, bakFilename)
	writer := zip.NewWriter(fzip)
	defer writer.Close()

	backup := func(d *Dice, fn string) {
		data, err := ioutil.ReadFile(fn)
		if err != nil {
			if d != nil {
				d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			}
			return
		}

		h := &zip.FileHeader{Name: fn, Method: zip.Deflate, Flags: 0x800}
		fileWriter, _ := writer.CreateHeader(h)
		//fileWriter, _ := writer.Create(fn)
		fileWriter.Write(data)
	}

	if cfg.Decks {
		filepath.Walk("data/decks", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				backup(nil, path)
			}
			return nil
		})
	}

	if cfg.HelpDoc {
		filepath.Walk("data/helpdoc", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				backup(nil, path)
			}
			return nil
		})
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
			backup(d, filepath.Join(d.BaseConfig.DataDir, "data.bdb"))
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

	writer.Close()
	fzip.Close()
	return nil
}

func (dm *DiceManager) BackupAuto() error {
	return dm.Backup(AllBackupConfig{
		Decks:   false,
		HelpDoc: false,
		Dices: map[string]*OneBackupConfig{
			"default": &OneBackupConfig{
				MiscConfig:  true,
				PlayerData:  true,
				CustomReply: true,
				CustomText:  true,
				Accounts:    true,
			},
		},
	}, "data-bak-"+time.Now().Format("2006_01_02_15_04_05")+"_auto_"+"*.zip")
}

func (dm *DiceManager) BackupSimple() error {
	return dm.Backup(AllBackupConfig{
		Decks:   false,
		HelpDoc: false,
		Dices: map[string]*OneBackupConfig{
			"default": &OneBackupConfig{
				MiscConfig:  true,
				PlayerData:  true,
				CustomReply: true,
				CustomText:  true,
				Accounts:    true,
			},
		},
	}, "data-bak-"+time.Now().Format("2006_01_02_15_04_05")+"_"+"*.zip")
}
