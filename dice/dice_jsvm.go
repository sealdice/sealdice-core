package dice

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/monaco-io/request"
	"io/fs"
	"path/filepath"
)

func (d *Dice) JsInit() {
	d.JsVM = goja.New()
	//d.JsRequire = d.Parent.JsRegistry.Enable(d.JsVM)
	d.JsRequire = new(require.Registry).Enable(d.JsVM)
	console.Enable(d.JsVM)

	d.JsVM.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", false))

	dice := d.JsVM.NewObject()
	//dice.Set("setVarInt", VarSetValueInt64)
	//dice.Set("setVarStr", VarSetValueStr)

	dice.Set("varGet", VarGetValue)
	dice.Set("varSet", VarSetValue)

	dice.Set("varGetInt", VarGetValueInt64)
	dice.Set("varSetInt", VarSetValueInt64)
	dice.Set("varGetStr", VarGetValueStr)
	dice.Set("varSetStr", VarSetValueStr)

	dice.Set("replyGroup", ReplyGroup)
	dice.Set("replyPerson", ReplyPerson)
	dice.Set("replyToSender", ReplyToSender)
	dice.Set("format", DiceFormat)
	dice.Set("formatTmpl", DiceFormatTmpl)
	dice.Set("getCtxProxyFirst", GetCtxProxyFirst)

	dice.Set("newCmdItemInfo", func() *CmdItemInfo {
		return &CmdItemInfo{}
	})

	dice.Set("newCmdExecuteResult", func(solved bool) CmdExecuteResult {
		return CmdExecuteResult{
			Matched: true,
			Solved:  solved,
		}
	})

	dice.Set("newHttpRequest", func() *request.Client {
		return &request.Client{}
	})

	dice.Set("newExt", func() *ExtInfo {
		return &ExtInfo{}
	})

	dice.Set("newCocRuleInfo", func() *CocRuleInfo {
		return &CocRuleInfo{}
	})

	dice.Set("newCocRuleCheckRet", func() *CocRuleCheckRet {
		return &CocRuleCheckRet{}
	})

	dice.Set("instance", d)
	d.JsVM.Set("__dirname", "")
	d.JsVM.Set("dice", dice)

	//	fmt.Println(d.JsVM.RunString(`
	//console.log(333, dice.newExt())
	//	dice.newExt()
	//`))
	//
	//	val, err := d.JsVM.RunString(`
	//ext = dice.newExt()
	//console.log(222, ext)
	//
	//ext.OnLoad = function() {
	//	console.log(1111111111)
	//}
	//
	//ext
	//
	//`)
	//	if err == nil {
	//		fmt.Println(val.Export(), val.ExportType())
	//		e := val.Export().(*ExtInfo)
	//		e.OnLoad()
	//		return
	//	}
}

func (d *Dice) JsLoadScripts() {
	path := filepath.Join(d.BaseConfig.DataDir, "scripts")
	filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if filepath.Ext(path) == ".js" {
			d.Logger.Info("正在读取脚本: ", path)
			_, err := d.JsRequire.Require("./" + path)
			if err != nil {
				d.Logger.Info("读取脚本失败: ", err.Error())
			}
		}
		return nil
	})
}
