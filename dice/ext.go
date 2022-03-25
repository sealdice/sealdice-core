package dice

import (
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
}

func (d *Dice) RegisterExtension(extInfo *ExtInfo) {
	d.ExtList = append(d.ExtList, extInfo)
}

func (d *Dice) GetExtDataDir(extName string) string {
	p := path.Join(d.BaseConfig.DataDir, "extensions", extName)
	os.MkdirAll(p, 0644)
	return p
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
		if i.Help == "" {
			text += "." + i.Name + "\n"
		} else {
			text += i.Help + "\n"
		}
	}

	return text
}
