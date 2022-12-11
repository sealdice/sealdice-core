package dice

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/monaco-io/request"
	"io/fs"
	"path/filepath"
)

type PrinterFunc struct {
	d        *Dice
	isRecord bool
	recorder []string
}

func (p *PrinterFunc) doRecord(_type string, s string) {
	if p.isRecord {
		p.recorder = append(p.recorder, s)
	}
}

func (p *PrinterFunc) RecordStart() { p.isRecord = true }
func (p *PrinterFunc) RecordEnd() []string {
	r := p.recorder
	p.recorder = []string{}
	return r
}

func (p *PrinterFunc) Log(s string) { p.doRecord("log", s); p.d.Logger.Info(s) }

func (p *PrinterFunc) Warn(s string) { p.doRecord("warn", s); p.d.Logger.Warn(s) }

func (p *PrinterFunc) Error(s string) { p.doRecord("error", s); p.d.Logger.Error(s) }

func (d *Dice) JsInit() {
	// 清理js扩展
	prepareRemove := []*ExtInfo{}
	for _, i := range d.ExtList {
		if i.IsJsExt {
			prepareRemove = append(prepareRemove, i)
		}
	}
	for _, i := range prepareRemove {
		d.ExtRemove(i)
	}
	// 清理coc扩展规则
	d.CocExtraRules = map[int]*CocRuleInfo{}

	// 重建js vm
	d.JsVM = goja.New()
	//d.JsRequire = d.Parent.JsRegistry.Enable(d.JsVM)

	reg := new(require.Registry)
	reg.Enable(d.JsVM)
	d.JsRequire = reg.Enable(d.JsVM) // 有点忘了为啥保留这个引用

	printer := &PrinterFunc{d, false, []string{}}
	d.JsPrinter = printer

	reg.RegisterNativeModule("console", console.RequireWithPrinter(printer))
	console.Enable(d.JsVM)

	d.JsVM.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))

	seal := d.JsVM.NewObject()
	//seal.Set("setVarInt", VarSetValueInt64)
	//seal.Set("setVarStr", VarSetValueStr)

	vars := d.JsVM.NewObject()
	seal.Set("vars", vars)
	//vars.Set("varGet", VarGetValue)
	//vars.Set("varSet", VarSetValue)
	vars.Set("intGet", VarGetValueInt64)
	vars.Set("intSet", VarSetValueInt64)
	vars.Set("strGet", VarGetValueStr)
	vars.Set("strSet", VarSetValueStr)

	ext := d.JsVM.NewObject()
	seal.Set("ext", ext)
	ext.Set("newCmdItemInfo", func() *CmdItemInfo {
		return &CmdItemInfo{IsJsSolveFunc: true}
	})
	ext.Set("newCmdExecuteResult", func(solved bool) CmdExecuteResult {
		return CmdExecuteResult{
			Matched: true,
			Solved:  solved,
		}
	})
	ext.Set("new", func(name, author, version string) *ExtInfo {
		return &ExtInfo{Name: name, Author: author, Version: version,
			GetDescText: func(i *ExtInfo) string {
				return GetExtensionDesc(i)
			},
			AutoActive: true,
			IsJsExt:    true,
			Brief:      "一个JS自定义扩展",
			CmdMap:     CmdMapCls{},
		}
	})
	ext.Set("find", func(name string) *ExtInfo {
		return d.ExtFind(name)
	})
	ext.Set("register", func(ei *ExtInfo) {
		d.RegisterExtension(ei)
		if ei.OnLoad != nil {
			ei.OnLoad()
		}
		d.ApplyExtDefaultSettings()
		for _, i := range d.ImSession.ServiceAtNew {
			i.ExtActive(ei)
		}
	})

	// COC规则自定义
	coc := d.JsVM.NewObject()
	coc.Set("newRule", func() *CocRuleInfo {
		return &CocRuleInfo{}
	})
	coc.Set("newRuleCheckResult", func() *CocRuleCheckRet {
		return &CocRuleCheckRet{}
	})
	coc.Set("registerRule", func(rule *CocRuleInfo) bool {
		return d.CocExtraRulesAdd(rule)
	})
	seal.Set("coc", coc)

	seal.Set("replyGroup", ReplyGroup)
	seal.Set("replyPerson", ReplyPerson)
	seal.Set("replyToSender", ReplyToSender)
	seal.Set("format", DiceFormat)
	seal.Set("formatTmpl", DiceFormatTmpl)
	seal.Set("getCtxProxyFirst", GetCtxProxyFirst)

	seal.Set("newHttpRequest", func() *request.Client {
		return &request.Client{}
	})

	seal.Set("inst", d)
	d.JsVM.Set("__dirname", "")
	d.JsVM.Set("seal", seal)
	//})

	//d.JsVM.Set("dice", d)

	//fmt.Println(d.JsVM.RunString(`
	//console.log(333, seal.newExt())
	//	seal.newExt()
	//`))
	//
	//	val, err := d.JsVM.RunString(`
	//ext = seal.newExt()
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
