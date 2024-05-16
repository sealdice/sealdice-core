package dice

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/dop251/goja"
	"github.com/tidwall/buntdb"
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

// RegisterExtension 注册扩展
//
// panic 如果扩展的Name或Aliases冲突
func (d *Dice) RegisterExtension(extInfo *ExtInfo) {
	for _, name := range append(extInfo.Aliases, extInfo.Name) {
		if collide := d.ExtFind(name); collide != nil {
			panicMsg := fmt.Sprintf("扩展<%s>的名字%q与现存扩展<%s>冲突", extInfo.Name, name, collide.Name)
			panic(panicMsg)
		}
	}

	extInfo.dice = d
	d.ExtList = append(d.ExtList, extInfo)
}

func (d *Dice) GetExtDataDir(extName string) string {
	p := path.Join(d.BaseConfig.DataDir, "extensions", extName)
	_ = os.MkdirAll(p, 0755)
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

	for k := range ei.CmdMap {
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
		if d.Config.JsEnable {
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
			d.Logger.Infof("当前已关闭js，跳过<%v>", i.Name)
		}
	} else {
		f()
	}
}

func (i *ExtInfo) StorageInit() error {
	var err error
	if i.dice == nil {
		return errors.New("[扩展]:请先完成此扩展的注册")
	}
	d := i.dice

	// 使用互斥锁保护初始化过程，确保只初始化一次
	i.dbMu.Lock()
	defer i.dbMu.Unlock()
	d.Logger.Debugf("[扩展]：%s 正在尝试获取锁进行初始化", i.Name)
	if i.init {
		d.Logger.Debug("[扩展]:初始化调用，但数据库已经加载")
		// 如果已经初始化，则直接返回
		return nil
	}

	dir := d.GetExtDataDir(i.Name)
	fn := path.Join(dir, "storage.db")
	i.Storage, err = buntdb.Open(fn)
	if err != nil {
		d.Logger.Errorf("[扩展]:初始化扩展数据库失败，原因：%v，路径为：%s", err, fn)
		return err
	}
	// 否则初始化后使用
	i.init = true
	return nil
}

func (i *ExtInfo) StorageClose() error {
	// 先上锁
	i.dbMu.Lock()
	// 保证还锁
	defer i.dbMu.Unlock()
	// 检查是否在init中，若已经关闭了就不需要处理了
	if !i.init {
		return nil
	}
	// 说初始化了但没有初始化，应该抛出异常
	if i.Storage == nil {
		return errors.New("[扩展]:Storage初始化错误")
	}
	err := i.Storage.Close()
	// 经Xiangze-Li 佬提示，如果关闭失败，应该直接返回异常
	// 不过向上是否有正确的err处理逻辑？我暂且蒙在鼓里
	if err != nil {
		i.dice.Logger.Errorf("[扩展]:关闭扩展数据库失败，原因：%v", err)
		return err
	}
	// 关闭成功，将Storage放空（实际我也不清楚是否需要该操作），返回nil.
	i.Storage = nil
	// 将init放为初始值false
	i.init = false
	return nil
}

func (i *ExtInfo) StorageSet(k, v string) error {
	if err := i.StorageInit(); err != nil {
		return err
	}
	db := i.Storage
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(k, v, nil)
		return err
	})
}

func (i *ExtInfo) StorageGet(k string) (string, error) {
	if err := i.StorageInit(); err != nil {
		return "", err
	}

	var val string
	var err error

	db := i.Storage
	err = db.View(func(tx *buntdb.Tx) error {
		val, err = tx.Get(k)
		if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		return nil
	})

	return val, err
}
