package dice

import (
	"encoding/base64"
	"encoding/json"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	fetch "github.com/fy0/gojax/fetch"
	"github.com/pkg/errors"
	"gopkg.in/elazarl/goproxy.v1"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	// 装载数据库(如果是初次运行)

	// 清理目前的js相关
	d.jsClear()

	// 重建js vm
	reg := new(require.Registry)

	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false), eventloop.WithRegistry(reg))
	_ = fetch.Enable(loop, goproxy.NewProxyHttpServer())
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
		_ = seal.Set("vars", vars)
		//vars.Set("varGet", VarGetValue)
		//vars.Set("varSet", VarSetValue)
		_ = vars.Set("intGet", VarGetValueInt64)
		_ = vars.Set("intSet", VarSetValueInt64)
		_ = vars.Set("strGet", VarGetValueStr)
		_ = vars.Set("strSet", VarSetValueStr)

		ext := vm.NewObject()
		_ = seal.Set("ext", ext)
		_ = ext.Set("newCmdItemInfo", func() *CmdItemInfo {
			return &CmdItemInfo{IsJsSolveFunc: true}
		})
		_ = ext.Set("newCmdExecuteResult", func(solved bool) CmdExecuteResult {
			return CmdExecuteResult{
				Matched: true,
				Solved:  solved,
			}
		})
		_ = ext.Set("new", func(name, author, version string) *ExtInfo {
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
		_ = ext.Set("find", func(name string) *ExtInfo {
			return d.ExtFind(name)
		})
		_ = ext.Set("register", func(ei *ExtInfo) {
			if d.ExtFind(ei.Name) != nil {
				panic("扩展<" + ei.Name + ">已被注册")
			}

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
		_ = coc.Set("newRule", func() *CocRuleInfo {
			return &CocRuleInfo{}
		})
		_ = coc.Set("newRuleCheckResult", func() *CocRuleCheckRet {
			return &CocRuleCheckRet{}
		})
		_ = coc.Set("registerRule", func(rule *CocRuleInfo) bool {
			return d.CocExtraRulesAdd(rule)
		})
		_ = seal.Set("coc", coc)

		deck := vm.NewObject()
		_ = deck.Set("draw", func(ctx *MsgContext, deckName string, isShuffle bool) map[string]interface{} {
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
		_ = deck.Set("reload", func() {
			DeckReload(d)
		})
		_ = seal.Set("deck", deck)

		_ = seal.Set("replyGroup", ReplyGroup)
		_ = seal.Set("replyPerson", ReplyPerson)
		_ = seal.Set("replyToSender", ReplyToSender)
		_ = seal.Set("memberBan", MemberBan)
		_ = seal.Set("memberKick", MemberKick)
		_ = seal.Set("format", DiceFormat)
		_ = seal.Set("formatTmpl", DiceFormatTmpl)
		_ = seal.Set("getCtxProxyFirst", GetCtxProxyFirst)

		// 1.2新增
		_ = seal.Set("newMessage", func() *Message {
			return &Message{}
		})
		_ = seal.Set("createTempCtx", CreateTempCtx)
		_ = seal.Set("applyPlayerGroupCardByTemplate", func(ctx *MsgContext, tmpl string) string {
			if tmpl != "" {
				ctx.Player.AutoSetNameTemplate = tmpl
			}
			if ctx.Player.AutoSetNameTemplate != "" {
				text, _ := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				return text
			}
			return ""
		})
		gameSystem := vm.NewObject()
		_ = gameSystem.Set("newTemplate", func(data string) error {
			tmpl := &GameSystemTemplate{}
			err := json.Unmarshal([]byte(data), tmpl)
			if err != nil {
				return errors.New("解析失败:" + err.Error())
			}
			ret := d.GameSystemTemplateAdd(tmpl)
			if !ret {
				return errors.New("已存在同名模板")
			}
			return nil
		})
		_ = gameSystem.Set("newTemplateByYaml", func(data string) error {
			tmpl := &GameSystemTemplate{}
			err := yaml.Unmarshal([]byte(data), tmpl)
			if err != nil {
				return errors.New("解析失败:" + err.Error())
			}
			ret := d.GameSystemTemplateAdd(tmpl)
			if !ret {
				return errors.New("已存在同名模板")
			}
			return nil
		})
		_ = seal.Set("gameSystem", gameSystem)
		_ = seal.Set("getCtxProxyAtPos", GetCtxProxyAtPos)
		_ = seal.Set("getVersion", func() map[string]interface{} {
			return map[string]interface{}{
				"versionCode": VERSION_CODE,
				"version":     VERSION,
			}
		})
		_ = seal.Set("getEndPoints", func() []*EndPointInfo {
			return d.ImSession.EndPoints
		})

		_ = vm.Set("atob", func(s string) (string, error) {
			// Remove data URI scheme and any whitespace from the string.
			s = strings.Replace(s, "data:text/plain;base64,", "", -1)
			s = strings.Replace(s, " ", "", -1)

			// Decode the base64-encoded string.
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", errors.New("atob: 不合法的base64字串")
			}

			return string(b), nil
		})
		_ = vm.Set("btoa", func(s string) string {
			// 编码
			return base64.StdEncoding.EncodeToString([]byte(s))
		})
		// 1.2新增结束

		_ = seal.Set("inst", d)
		_ = vm.Set("__dirname", "")
		_ = vm.Set("seal", seal)

		_, _ = vm.RunString(`
let e = seal.ext.new('_', '', '');
e.__proto__.storageSet = function(k, v) {
  try {
    // 这里goja会强行抛出异常，等于是将返回error的函数转写成throw形式
    this.storageSetRaw(k, v)
  } catch (error) {
    throw error;
  }
}
e.__proto__.storageGet = function(k, v) {
  try {
    return this.storageGetRaw(k, v);
  } catch (error) {
    if (error.value.toString() !== 'not found') {
      throw error;
    }
  }
}
`)
		_, _ = vm.RunString(`Object.freeze(seal);Object.freeze(seal.deck);Object.freeze(seal.coc);Object.freeze(seal.ext);Object.freeze(seal.vars);`)
	})
	loop.Start()
	d.JsEnable = true
	d.Logger.Info("已加载JS环境")
}

func (d *Dice) JsShutdown() {
	d.JsEnable = false
	d.jsClear()
	d.Logger.Info("已关闭JS环境")
}

func (d *Dice) jsClear() {
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
	// 关闭js vm
	if d.JsLoop != nil {
		d.JsLoop.Stop()
		d.JsLoop = nil
	}
}

func (d *Dice) JsLoadScripts() {
	d.JsScriptList = []*JsScriptInfo{}
	path := filepath.Join(d.BaseConfig.DataDir, "scripts")
	_ = filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
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
	/** 是否启用 */
	Enable bool `json:"enable"`
	/** 版本 */
	Version string `json:"version"`
	/** 作者 */
	Author string `json:"author"`
	/** 许可协议 */
	License string `json:"license"`
	/** 网址 */
	HomePage string `json:"homepage"`
	/** 详细描述 */
	Desc string `json:"desc"`
	/** 所需权限 */
	Grant []string `json:"grant"`
	/** 更新时间 */
	UpdateTime int64 `json:"updateTime"`
	/** 安装时间 - 文件创建时间 */
	InstallTime int64 `json:"installTime"`
	/** 最近一条错误文本 */
	ErrText string `json:"errText"`
	/** 实际文件名 */
	Filename string
}

func (d *Dice) JsLoadScriptRaw(s string, info fs.FileInfo) {
	// 读取文件内容填空，类似油猴脚本那种形式
	jsInfo := &JsScriptInfo{
		Name:        info.Name(),
		Filename:    s,
		InstallTime: info.ModTime().Unix(),
	}

	fileTextRaw, err := os.ReadFile(s)
	if err != nil {
		d.Logger.Error("读取脚本失败(无法访问): ", err.Error())
		return
	}

	// 解析信息
	fileText := string(fileTextRaw)
	re := regexp.MustCompile(`(?s)//[ \t]*==UserScript==[ \t]*\r?\n(.*)//[ \t]*==/UserScript==`)
	m := re.FindStringSubmatch(fileText)
	if len(m) > 0 {
		text := m[0]
		re2 := regexp.MustCompile(`//[ \t]*@(\S+)\s+([^\r\n]+)`)
		data := re2.FindAllStringSubmatch(text, -1)
		for _, item := range data {
			v := strings.TrimSpace(item[2])
			switch item[1] {
			case "name":
				jsInfo.Name = v
			case "homepageURL":
				jsInfo.HomePage = v
			case "license":
				jsInfo.License = v
			case "author":
				jsInfo.Author = v
			case "version":
				jsInfo.Version = v
			case "description":
				jsInfo.Desc = v
			case "timestamp":
				v, err := strconv.ParseInt(v, 10, 64)
				if err == nil {
					jsInfo.UpdateTime = v
				}
			}
		}
	}
	jsInfo.Enable = !d.DisabledJsScripts[jsInfo.Name]

	if jsInfo.Enable {
		_, err = d.JsRequire.Require(s)
	} else {
		d.Logger.Infof("脚本<%s>已被禁用，跳过加载", jsInfo.Name)
	}

	if err != nil {
		errText := err.Error()
		jsInfo.ErrText = errText
		d.Logger.Error("读取脚本失败(解析失败): ", errText)
	} else {
		d.JsScriptList = append(d.JsScriptList, jsInfo)
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
		_ = os.Remove(jsInfo.Filename)
	}
}

func JsEnable(d *Dice, jsInfoName string) {
	delete(d.DisabledJsScripts, jsInfoName)
	for _, jsInfo := range d.JsScriptList {
		if jsInfo.Name == jsInfoName {
			jsInfo.Enable = true
		}
	}
}

func JsDisable(d *Dice, jsInfoName string) {
	d.DisabledJsScripts[jsInfoName] = true
	for _, jsInfo := range d.JsScriptList {
		if jsInfo.Name == jsInfoName {
			jsInfo.Enable = false
		}
	}
}
