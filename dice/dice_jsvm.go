package dice

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/olebedev/gojax/fetch"
	"gopkg.in/elazarl/goproxy.v1"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

func (p *PrinterFunc) RecordStart() { p.recorder = []string{}; p.isRecord = true }
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
	// 清理脚本列表
	d.JsScriptList = []*JsScriptInfo{}

	// 重建js vm
	if d.JsLoop != nil {
		d.JsLoop.Stop()
	}
	reg := new(require.Registry)

	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false), eventloop.WithRegistry(reg))
	fetch.Enable(loop, goproxy.NewProxyHttpServer())
	d.JsLoop = loop

	printer := &PrinterFunc{d, false, []string{}}
	d.JsPrinter = printer
	reg.RegisterNativeModule("node:console", console.RequireWithPrinter(printer))

	// 初始化
	loop.Run(func(vm *goja.Runtime) {
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))

		// console 模块
		console.Enable(vm)

		// require 模块
		d.JsRequire = reg.Enable(vm)

		seal := vm.NewObject()
		//seal.Set("setVarInt", VarSetValueInt64)
		//seal.Set("setVarStr", VarSetValueStr)

		vars := vm.NewObject()
		seal.Set("vars", vars)
		//vars.Set("varGet", VarGetValue)
		//vars.Set("varSet", VarSetValue)
		vars.Set("intGet", VarGetValueInt64)
		vars.Set("intSet", VarSetValueInt64)
		vars.Set("strGet", VarGetValueStr)
		vars.Set("strSet", VarSetValueStr)

		ext := vm.NewObject()
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
		coc := vm.NewObject()
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

		deck := vm.NewObject()
		deck.Set("draw", func(ctx *MsgContext, deckName string, isShuffle bool) map[string]interface{} {
			exists, result, err := deckDraw(ctx, deckName, isShuffle)
			var errText string
			if err != nil {
				errText = err.Error()
			}
			return map[string]interface{}{
				"exists": exists,
				"err":    errText,
				"result": result,
			}
		})
		deck.Set("reload", func() {
			DeckReload(d)
		})
		seal.Set("deck", deck)

		seal.Set("replyGroup", ReplyGroup)
		seal.Set("replyPerson", ReplyPerson)
		seal.Set("replyToSender", ReplyToSender)
		seal.Set("format", DiceFormat)
		seal.Set("formatTmpl", DiceFormatTmpl)
		seal.Set("getCtxProxyFirst", GetCtxProxyFirst)

		seal.Set("inst", d)
		vm.Set("__dirname", "")
		vm.Set("seal", seal)
	})
	loop.Start()
}

func (d *Dice) JsLoadScripts() {
	d.JsScriptList = []*JsScriptInfo{}
	path := filepath.Join(d.BaseConfig.DataDir, "scripts")
	filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if filepath.Ext(path) == ".js" {
			d.Logger.Info("正在读取脚本: ", path)
			d.JsLoadScriptRaw("./"+path, info)
		}
		return nil
	})
}

type JsScriptInfo struct {
	/** 名称 */
	Name string `json:"name"`
	/** 版本 */
	Version string `json:"version"`
	/** 作者 */
	Author string `json:"author"`
	/** 许可协议 */
	License string `json:"license"`
	/** 网址 */
	Website string `json:"website"`
	/** 详细描述 */
	Desc string `json:"desc"`
	/** 所需权限 */
	Grant []string `json:"grant"`
	/** 更新时间 */
	UpdateTime int64 `json:"updateTime"`

	/** 是否启用 未来再加这个功能吧，现在所有的都默认启用 */
	//Enable bool `json:"enable"`
	/** 安装时间 - 文件创建时间 */
	InstallTime int64 `json:"installTime"`
	/** 最近一条错误文本 */
	ErrText string `json:"errText"`
	/** 实际文件名 */
	Filename string
}

func (d *Dice) JsLoadScriptRaw(s string, info fs.FileInfo) {
	// TODO: 读取文件内容填空，类似油猴脚本那种形式
	jsInfo := &JsScriptInfo{
		Name:        info.Name(),
		Filename:    s,
		InstallTime: info.ModTime().Unix(),
	}
	d.JsScriptList = append(d.JsScriptList, jsInfo)
	_, err := d.JsRequire.Require(s)
	if err != nil {
		errText := err.Error()
		jsInfo.ErrText = errText
		d.Logger.Error("读取脚本失败: ", errText)
	}
}

func JsDelete(d *Dice, jsInfo *JsScriptInfo) {
	dirpath := filepath.Dir(jsInfo.Filename)
	dirname := filepath.Base(dirpath)

	if strings.HasPrefix(dirname, "_") && strings.HasSuffix(dirname, ".deck") {
		// 可能是zip解压出来的，那么删除目录和压缩包
		_ = os.RemoveAll(dirpath)
		zipFilename := filepath.Join(filepath.Dir(dirpath), dirname[1:])
		_ = os.Remove(zipFilename)
	} else {
		fmt.Println("???", jsInfo.Filename)
		_ = os.Remove(jsInfo.Filename)
	}
}
