package dice

import (
	"errors"
	"github.com/dop251/goja"
	"github.com/tidwall/buntdb"
	"os"
	"path"
	"sort"
)

func (d *Dice) RegisterBuiltinExt() {
	RegisterBuiltinExtCoc7(d)
	RegisterBuiltinExtLog(d)
	RegisterBuiltinExtFun(d)
	RegisterBuiltinExtDeck(d)
	RegisterBuiltinExtReply(d)
	RegisterBuiltinExtDnd5e(d)
	RegisterBuiltinStory(d)
	RegisterBuiltinExtExp(d)
}

func (d *Dice) RegisterExtension(extInfo *ExtInfo) {
	extInfo.dice = d
	d.ExtList = append(d.ExtList, extInfo)
}

func (d *Dice) GetExtDataDir(extName string) string {
	p := path.Join(d.BaseConfig.DataDir, "extensions", extName)
	os.MkdirAll(p, 0755)
	return p
}

func (d *Dice) GetDiceDataPath(name string) string {
	return path.Join(d.BaseConfig.DataDir, name)
}

func (d *Dice) GetExtConfigFilePath(extName string, filename string) string {
	return path.Join(d.GetExtDataDir(extName), filename)
}

func GetExtensionDesc(ei *ExtInfo) string {
	text := "> " + ei.Brief + "\n" + "提供命令:\n"
	keys := make([]string, 0, len(ei.CmdMap))

	valueMap := map[*CmdItemInfo]bool{}

	for k, _ := range ei.CmdMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, i := range keys {
		i := ei.CmdMap[i]
		if valueMap[i] {
			continue
		}
		valueMap[i] = true
		if i.ShortHelp == "" {
			text += "." + i.Name + "\n"
		} else {
			text += i.ShortHelp + "\n"
		}
	}

	return text
}

func (i *ExtInfo) callWithJsCheck(d *Dice, f func()) {
	if i.IsJsExt {
		waitRun := make(chan int, 1)
		d.JsLoop.RunOnLoop(func(vm *goja.Runtime) {
			defer func() {
				// 防止崩掉进程
				if r := recover(); r != nil {
					d.Logger.Error("JS脚本报错:", r)
				}
				waitRun <- 1
			}()

			f()
		})
		<-waitRun
	} else {
		f()
	}
}

func (i *ExtInfo) StorageInit() error {
	var err error
	if i.dice == nil {
		return errors.New("请先完成此扩展的注册")
	}
	d := i.dice
	// 注: 这里可能会有极小概率并发问题
	if i.Storage == nil {
		dir := d.GetExtDataDir(i.Name)
		fn := path.Join(dir, "storage.db")
		i.Storage, err = buntdb.Open(fn)
		if err != nil {
			d.Logger.Error("初始化扩展数据库失败", fn)
			d.Logger.Error(err.Error())
			return err
		}
	}
	return err
}

func (i *ExtInfo) StorageSetRaw(k, v string) error {
	if err := i.StorageInit(); err != nil {
		return err
	}

	db := i.Storage
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(k, v, nil)
		return err
	})
}

func (i *ExtInfo) StorageGetRaw(k string) (string, error) {
	if err := i.StorageInit(); err != nil {
		return "", err
	}
	var val string
	var err error

	db := i.Storage
	err = db.View(func(tx *buntdb.Tx) error {
		val, err = tx.Get(k)
		if err != nil {
			return err
		}
		return nil
	})

	return val, err
}
