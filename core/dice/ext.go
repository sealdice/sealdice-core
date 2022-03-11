package dice

import (
	"os"
	"path"
)

func (d *Dice) RegisterBuiltinExt() {
	RegisterBuiltinExtCoc7(d)
	RegisterBuiltinExtLog(d)
	RegisterBuiltinExtFun(d)
	RegisterBuiltinExtDeck(d)
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
