package dice

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	fetch "github.com/fy0/gojax/fetch"
	"github.com/golang-module/carbon"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/elazarl/goproxy.v1"
	"gopkg.in/yaml.v3"

	"sealdice-core/static"
	"sealdice-core/utils"
	"sealdice-core/utils/crypto"
)

var (
	// OfficialModPublicKey 官方 Mod 公钥
	OfficialModPublicKey = ``

	signRe = regexp.MustCompile(`^// sign\s+([^\r\n]+)?[\r\n]+$`)
)

type PrinterFunc struct {
	d        *Dice
	isRecord bool
	recorder []string
}

func (p *PrinterFunc) doRecord(_ string, s string) {
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

func (p *PrinterFunc) Log(s string) {
	p.doRecord("log", s)
	p.d.Logger.Info(s)
}

func (p *PrinterFunc) Warn(s string) { p.doRecord("warn", s); p.d.Logger.Warn(s) }

func (p *PrinterFunc) Error(s string) { p.doRecord("error", s); p.d.Logger.Error(s) }

func (d *Dice) JsInit() {
	// 读取官方 Mod 公钥
	if pub, err := static.Scripts.ReadFile("scripts/seal_mod.public.pem"); err == nil && len(pub) > 0 {
		OfficialModPublicKey = string(pub)
	}

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
	reg.RegisterNativeModule("console", console.RequireWithPrinter(printer))

	// 初始化
	loop.Run(func(vm *goja.Runtime) {
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))

		// console 模块
		console.Enable(vm)

		// require 模块
		d.JsRequire = reg.Enable(vm)

		seal := vm.NewObject()

		vars := vm.NewObject()
		_ = seal.Set("vars", vars)
		_ = vars.Set("intGet", VarGetValueInt64)
		_ = vars.Set("intSet", VarSetValueInt64)
		_ = vars.Set("strGet", VarGetValueStr)
		_ = vars.Set("strSet", VarSetValueStr)

		ban := vm.NewObject()
		_ = seal.Set("ban", ban)
		_ = ban.Set("addBan", func(ctx *MsgContext, id string, place string, reason string) {
			d.BanList.AddScoreBase(id, d.BanList.ThresholdBan, place, reason, ctx)
			d.BanList.SaveChanged(d)
		})
		_ = ban.Set("addTrust", func(ctx *MsgContext, id string, place string, reason string) {
			d.BanList.SetTrustByID(id, place, reason)
			d.BanList.SaveChanged(d)
		})
		_ = ban.Set("remove", func(ctx *MsgContext, id string) {
			_, ok := d.BanList.GetByID(id)
			if !ok {
				return
			}
			d.BanList.DeleteByID(d, id)
		})
		_ = ban.Set("getList", func() []BanListInfoItem {
			var list []BanListInfoItem
			d.BanList.Map.Range(func(key string, value *BanListInfoItem) bool {
				list = append(list, *value)
				return true
			})
			return list
		})
		_ = ban.Set("getUser", func(id string) *BanListInfoItem {
			i, ok := d.BanList.GetByID(id)
			if !ok {
				return nil
			}
			cp := *i
			return &cp
		})

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
			var official bool
			if d.JsLoadingScript != nil {
				official = d.JsLoadingScript.Official
			}
			return &ExtInfo{
				Name: name, Author: author, Version: version,
				GetDescText: GetExtensionDesc,
				AutoActive:  true,
				IsJsExt:     true,
				Brief:       "一个JS自定义扩展",
				Official:    official,
				CmdMap:      CmdMapCls{},
				Source:      d.JsLoadingScript,
			}
		})
		_ = ext.Set("find", func(name string) *ExtInfo {
			return d.ExtFind(name)
		})
		_ = ext.Set("register", func(ei *ExtInfo) {
			// NOTE(Xiangze Li): 移动到dice.RegisterExtension里去检查
			// if d.ExtFind(ei.Name) != nil {
			// 	panic("扩展<" + ei.Name + ">已被注册")
			// }

			defer func() {
				// 增加recover, 以免在scripts目录中存在名字冲突扩展时导致启动崩溃
				if e := recover(); e != nil {
					d.Logger.Error(e)
				}
			}()

			d.RegisterExtension(ei)
			if ei.OnLoad != nil {
				ei.OnLoad()
			}
			d.ApplyExtDefaultSettings()
			for _, i := range d.ImSession.ServiceAtNew {
				i.ExtActive(ei)
			}
		})
		_ = ext.Set("registerStringConfig", func(ei *ExtInfo, key string, defaultValue string) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			config := &ConfigItem{
				Key:          key,
				Type:         "string",
				Value:        defaultValue,
				DefaultValue: defaultValue,
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
			return nil
		})
		_ = ext.Set("registerIntConfig", func(ei *ExtInfo, key string, defaultValue int64) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			config := &ConfigItem{
				Key:          key,
				Type:         "int",
				Value:        defaultValue,
				DefaultValue: defaultValue,
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
			return nil
		})
		_ = ext.Set("registerBoolConfig", func(ei *ExtInfo, key string, defaultValue bool) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			config := &ConfigItem{
				Key:          key,
				Type:         "bool",
				Value:        defaultValue,
				DefaultValue: defaultValue,
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
			return nil
		})
		_ = ext.Set("registerFloatConfig", func(ei *ExtInfo, key string, defaultValue float64) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			config := &ConfigItem{
				Key:          key,
				Type:         "float",
				Value:        defaultValue,
				DefaultValue: defaultValue,
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
			return nil
		})
		_ = ext.Set("registerTemplateConfig", func(ei *ExtInfo, key string, defaultValue []string) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			config := &ConfigItem{
				Key:          key,
				Type:         "template",
				Value:        defaultValue,
				DefaultValue: defaultValue,
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
			return nil
		})
		_ = ext.Set("registerOptionConfig", func(ei *ExtInfo, key string, defaultValue string, option []string) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			config := &ConfigItem{
				Key:          key,
				Type:         "option",
				Value:        defaultValue,
				DefaultValue: defaultValue,
				Option:       option,
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
			return nil
		})
		_ = ext.Set("newConfigItem", func(ei *ExtInfo, key string, defaultValue interface{}) *ConfigItem {
			if ei.dice == nil {
				panic(errors.New("请先完成此扩展的注册"))
			}
			return d.ConfigManager.NewConfigItem(key, defaultValue)
		})
		_ = ext.Set("registerConfig", func(ei *ExtInfo, config ...*ConfigItem) error {
			if ei.dice == nil {
				return errors.New("请先完成此扩展的注册")
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config...)
			return nil
		})
		_ = ext.Set("getConfig", func(ei *ExtInfo, key string) *ConfigItem {
			if ei.dice == nil {
				return nil
			}
			return d.ConfigManager.getConfig(ei.Name, key)
		})
		_ = ext.Set("getStringConfig", func(ei *ExtInfo, key string) string {
			if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "string" {
				panic("配置不存在或类型不匹配")
			}
			return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
		})
		_ = ext.Set("getIntConfig", func(ei *ExtInfo, key string) int64 {
			if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "int" {
				panic("配置不存在或类型不匹配")
			}
			return d.ConfigManager.getConfig(ei.Name, key).Value.(int64)
		})
		_ = ext.Set("getBoolConfig", func(ei *ExtInfo, key string) bool {
			if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "bool" {
				panic("配置不存在或类型不匹配")
			}
			return d.ConfigManager.getConfig(ei.Name, key).Value.(bool)
		})
		_ = ext.Set("getFloatConfig", func(ei *ExtInfo, key string) float64 {
			if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "float" {
				panic("配置不存在或类型不匹配")
			}
			return d.ConfigManager.getConfig(ei.Name, key).Value.(float64)
		})
		_ = ext.Set("getTemplateConfig", func(ei *ExtInfo, key string) []string {
			if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "template" {
				panic("配置不存在或类型不匹配")
			}
			return d.ConfigManager.getConfig(ei.Name, key).Value.([]string)
		})
		_ = ext.Set("getOptionConfig", func(ei *ExtInfo, key string) string {
			if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "option" {
				panic("配置不存在或类型不匹配")
			}
			return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
		})
		_ = ext.Set("unregisterConfig", func(ei *ExtInfo, key ...string) {
			if ei.dice == nil {
				return
			}
			d.ConfigManager.UnregisterConfig(ei.Name, key...)
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
				"versionCode":   VERSION_CODE,
				"version":       VERSION.String(),
				"versionSimple": VERSION_MAIN + VERSION_PRERELEASE,
				"versionDetail": map[string]interface{}{
					"major":         VERSION.Major(),
					"minor":         VERSION.Minor(),
					"patch":         VERSION.Patch(),
					"prerelease":    VERSION.Prerelease(),
					"buildMetaData": VERSION.Metadata(),
				},
			}
		})
		_ = seal.Set("getEndPoints", func() []*EndPointInfo {
			return d.ImSession.EndPoints
		})

		_ = vm.Set("atob", func(s string) (string, error) {
			// Remove data URI scheme and any whitespace from the string.
			s = strings.ReplaceAll(s, "data:text/plain;base64,", "")
			s = strings.ReplaceAll(s, " ", "")

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

		_ = seal.Set("setPlayerGroupCard", SetPlayerGroupCardByTemplate)
		_ = seal.Set("base64ToImage", utils.Base64ToImageFunc(d.Logger))

		// Note: Szzrain 暴露dice对象给js会导致js可以调用dice的所有Export的方法
		// 这是不安全的, 所有需要用到dice实例的函数都可以以传入ctx作为替代
		// _ = seal.Set("inst", d)
		_ = vm.Set("__dirname", "")
		_ = vm.Set("seal", seal)

		// Note(Szzrain): 不要修改原型链, 会导致一些奇怪的问题，比如无法使用某些 TS 库
		//		_, _ = vm.RunString(`
		// let e = seal.ext.new('_', '', '');
		// e.__proto__.storageSet = function(k, v) {
		//  try {
		//    // 这里goja会强行抛出异常，等于是将返回error的函数转写成throw形式
		//    this.storageSetRaw(k, v)
		//  } catch (error) {
		//    throw error;
		//  }
		// }
		// e.__proto__.storageGet = function(k, v) {
		//  try {
		//    return this.storageGetRaw(k, v);
		//  } catch (error) {
		//    if (error.value.toString() !== 'not found') {
		//      throw error;
		//    }
		//  }
		// }
		// `)
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
	builtinPath := filepath.Join(path, "_builtin")

	// 导出内置脚本数据
	builtinScripts, _ := fs.ReadDir(static.Scripts, "scripts")
	_ = os.MkdirAll(builtinPath, 0o755)
	for _, script := range builtinScripts {
		if !script.IsDir() && filepath.Ext(script.Name()) == ".js" {
			target := filepath.Join(builtinPath, script.Name())
			data, _ := static.Scripts.ReadFile("scripts/" + script.Name())
			d.JsBuiltinDigestSet[crypto.CalculateSHA512Str(data)] = true
			// 判断是否有更新后的内置脚本
			_, err := os.Stat(target)
			if errors.Is(err, os.ErrNotExist) {
				_ = os.WriteFile(target, data, 0o644)
			} else {
				// 检查同名内置脚本的签名，检查不通过则覆盖
				scriptData, _ := os.ReadFile(target)
				if ok, _ := CheckJsSign(scriptData); !ok {
					d.Logger.Warnf("已存在的内置脚本「%s」未通过校验，进行覆盖", script.Name())
					_ = os.WriteFile(target, scriptData, 0o644)
				}
			}
		}
	}

	var jsInfos []*JsScriptInfo
	// 解析内置脚本
	_ = filepath.Walk(builtinPath, func(path string, info fs.FileInfo, err error) error {
		if filepath.Ext(path) == ".js" {
			d.Logger.Info("正在读取内置脚本: ", path)
			data, err := os.ReadFile(path)
			if err != nil {
				d.Logger.Error("读取内置脚本失败(无法访问): ", err.Error())
				return nil
			}
			// 检查内置脚本签名，检查不通过则拒绝加载
			scriptData, _ := os.ReadFile(path)
			if ok, _ := CheckJsSign(scriptData); ok {
				jsInfo, err := d.JsParseMeta("./"+path, info.ModTime(), data, true)
				if err != nil {
					d.Logger.Error("读取内置脚本失败(错误依赖)", err.Error())
					return nil
				}
				jsInfos = append(jsInfos, jsInfo)
			} else {
				d.Logger.Warnf("内置脚本「%s」校验未通过，拒绝加载", path)
			}
		}
		return nil
	})

	// 解析第三方脚本
	_ = filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && info.Name() == "_builtin" {
			return fs.SkipDir
		}
		if filepath.Ext(path) == ".js" {
			d.Logger.Info("正在读取脚本: ", path)
			data, err := os.ReadFile(path)
			if err != nil {
				d.Logger.Error("读取脚本失败(无法访问): ", err.Error())
				return nil
			}
			jsInfo, err := d.JsParseMeta("./"+path, info.ModTime(), data, false)
			if err != nil {
				d.Logger.Error("读取脚本失败(错误依赖)", err.Error())
				return nil
			}
			jsInfos = append(jsInfos, jsInfo)
		}
		return nil
	})

	// 检查依赖是否满足
	unloadKeySet := make(map[string]bool)
	var unloadInfos []string
	scripts, invalidInfoMap := checkJsScriptsDeps(jsInfos)
	if len(invalidInfoMap) > 0 {
		// 部分插件依赖不满足，不进行加载
		var infos []string
		for k, v := range invalidInfoMap {
			unloadKeySet[k] = true
			infos = append(infos, v...)
		}
		unloadInfos = append(unloadInfos, infos...)
	}
	// 分析加载顺序
	sortedJsInfos, invalidInfoMap := sortJsScripts(scripts)
	if len(invalidInfoMap) != 0 {
		// 部分插件存在循环依赖，不进行加载
		var infos []string
		for k, v := range invalidInfoMap {
			unloadKeySet[k] = true
			infos = append(infos, v...)
		}
		unloadInfos = append(unloadInfos, infos...)
	}
	if len(unloadInfos) > 0 {
		var keys []string
		for key := range unloadKeySet {
			keys = append(keys, key)
		}
		d.Logger.Warnf("插件「%s」拒绝加载：\n%s", strings.Join(keys, "、"), strings.Join(unloadInfos, "\n"))
	}

	// 按顺序加载
	for _, jsInfo := range sortedJsInfos {
		if len(jsInfo.Depends) == 0 {
			d.Logger.Infof("正在加载脚本「%s:%s:%s」", jsInfo.Author, jsInfo.Name, jsInfo.Version)
		} else {
			var depends []string
			for _, dep := range jsInfo.Depends {
				depends = append(depends, dep.RawKey)
			}
			d.Logger.Infof("正在加载脚本「%s:%s:%s」，其依赖：%s", jsInfo.Author, jsInfo.Name, jsInfo.Version, strings.Join(depends, "、"))
		}
		d.JsLoadScriptRaw(jsInfo)
	}
}

func (d *Dice) JsReload() {
	d.JsInit()
	_ = d.ConfigManager.Load()
	d.JsLoadScripts()
}

// JsExtSettingVacuum 清理已被删除的脚本对应的插件配置
//
// Deprecated: bug
func (d *Dice) JsExtSettingVacuum() {
	// NOTE(Xiangze Li): 这里jsInfo中的Name字段是JS文件头中定义的@name,
	// 而ExtDefaultSettings中的Name字段是插件的名称,
	// 这两者的内容没有任何关联, 也没有字段在两者之间建立关系, 因此不能用来匹配.
	//
	// 另外, 对于已经删除/禁用的JS, ExtDefaultSetting中的ExtItem指针可能是nil

	jsMap := map[string]bool{}
	for _, jsInfo := range d.JsScriptList {
		jsMap[jsInfo.Name] = true
	}

	idxToDel := []int{}
	for k, v := range d.ExtDefaultSettings {
		if !v.ExtItem.IsJsExt {
			continue
		}
		if !jsMap[v.Name] {
			idxToDel = append(idxToDel, k)
		}
	}

	for i := len(idxToDel) - 1; i >= 0; i-- {
		idx := idxToDel[i]
		d.ExtDefaultSettings = append(d.ExtDefaultSettings[:idx], d.ExtDefaultSettings[idx+1:]...)
	}

	panic("DONT USE ME")
}

type Prop struct {
	Key   string `json:"key"`
	Value string `json:"value"`

	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Required bool   `json:"required"`
	Default  string `json:"default"`
}

type SignStatus int8

const (
	// ErrorSign 错误签名
	ErrorSign SignStatus = -1
	// UnknownSign 无签名
	UnknownSign SignStatus = 0
	// OfficialSign 官方签名
	OfficialSign SignStatus = 1
)

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
	Filename string `json:"filename"`
	/** 更新链接 */
	UpdateUrls []string `json:"updateUrls"`
	/** etag */
	Etag string `json:"etag"`
	/** 是否官方插件 */
	Official bool `json:"official"`
	/** 签名状态 */
	signStatus SignStatus
	/** 是否内置插件 */
	Builtin bool `json:"builtin"`
	/** 内容摘要 */
	Digest string `json:"-"`
	/** 依赖项 */
	Depends []JsScriptDepends `json:"depends"`
}

type JsScriptDepends struct {
	/** 作者 */
	Author string `json:"author"`
	/** 名称 */
	Name string `json:"name"`
	/** 版本限制 */
	Constraint *semver.Constraints `json:"constraint"`
	/** 原始依赖Key */
	RawKey string `json:"rawKey"`
}

func (d *Dice) JsParseMeta(s string, installTime time.Time, rawData []byte, builtin bool) (*JsScriptInfo, error) {
	// 读取文件内容填空，类似油猴脚本那种形式
	jsInfo := &JsScriptInfo{
		Name:        filepath.Base(s),
		Filename:    s,
		InstallTime: installTime.Unix(),
	}
	d.JsScriptList = append(d.JsScriptList, jsInfo)

	jsInfo.Builtin = builtin
	jsInfo.Digest = crypto.CalculateSHA512Str(rawData)

	// 解析签名
	official, signStatus := CheckJsSign(rawData)
	jsInfo.Official = official
	jsInfo.signStatus = signStatus

	// 解析信息
	fileText := string(rawData)
	re := regexp.MustCompile(`(?s)//[ \t]*==UserScript==[ \t]*\r?\n(.*)//[ \t]*==/UserScript==`)
	m := re.FindStringSubmatch(fileText)
	var errMsg []string

	if len(m) > 0 {
		text := m[0]
		re2 := regexp.MustCompile(`//[ \t]*@(\S+)\s+([^\r\n]+)`)
		data := re2.FindAllStringSubmatch(text, -1)
		updateUrls := make([]string, 0)

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
				v = strings.ReplaceAll(v, "\\n", "\n")
				jsInfo.Desc = v
			case "timestamp":
				timestamp, errParse := strconv.ParseInt(v, 10, 64)
				if errParse == nil {
					jsInfo.UpdateTime = timestamp
				} else {
					t := carbon.Parse(v)
					if t.IsValid() {
						jsInfo.UpdateTime = t.Timestamp()
					}
				}
			case "updateUrl":
				updateUrls = append(updateUrls, v)
			case "etag":
				jsInfo.Etag = v
			case "depends":
				dependsStr := strings.SplitN(v, ":", 2)
				if len(dependsStr) != 2 {
					errMsg = append(errMsg, fmt.Sprintf("插件「%s」指定依赖格式不正确，应为 作者:插件名:[SemVer版本约束，可选]，现为「%s」", jsInfo.Name, v))
					continue
				}
				author := dependsStr[0]
				name := dependsStr[1]
				var dependsInfo JsScriptDepends
				dependsInfo.Author = author
				dependsInfo.RawKey = v

				if strings.Contains(name, ":") {
					split := strings.SplitN(name, ":", 2)
					constraint, err := semver.NewConstraint(split[1])
					if err != nil {
						errMsg = append(errMsg, fmt.Sprintf("插件「%s」指定依赖格式不正确，应为 作者:插件名:[SemVer版本约束，可选]，现为「%s」", jsInfo.Name, v))
						continue
					}
					dependsInfo.Name = split[0]
					dependsInfo.Constraint = constraint
				} else {
					dependsInfo.Name = name
					dependsInfo.Constraint, _ = semver.NewConstraint("")
				}
				jsInfo.Depends = append(jsInfo.Depends, dependsInfo)
			case "sealVersion":
				vc, err := semver.NewConstraint(v)
				if err != nil {
					errMsg = append(errMsg, fmt.Sprintf("插件「%s」限制海豹版本的格式不正确，应满足semver版本范围语法，例如「1.4.0, >=1.4.0, 1.4.5-dev」等，当前为「%s」", jsInfo.Name, v))
					continue
				}

				var verOK bool
				// 有特殊符号时，进行严格的版本检查(只检查当前版本)
				if strings.ContainsAny(v, "~*^<=>|") || strings.Contains(v, " - ") {
					verOK = vc.Check(VERSION)
				} else {
					_, verOK = lo.Find(VERSION_JSAPI_COMPATIBLE, func(v *semver.Version) bool {
						return vc.Check(v)
					})
				}

				if !verOK {
					errMsg = append(errMsg, fmt.Sprintf("插件「%s」依赖的海豹版本限制在 %s，与海豹版本(%s)的JSAPI不兼容", jsInfo.Name, v, VERSION.String()))
				}
			}
		}
		jsInfo.UpdateUrls = updateUrls
	}

	if len(errMsg) > 0 {
		jsInfo.Enable = false
		jsInfo.ErrText = strings.Join(errMsg, "\n")
		return nil, errors.New(strings.Join(errMsg, "|"))
	}
	jsInfo.Enable = !d.DisabledJsScripts[jsInfo.Name]
	return jsInfo, nil
}

func (d *Dice) JsLoadScriptRaw(jsInfo *JsScriptInfo) {
	var err error
	if jsInfo.Enable {
		d.JsLoadingScript = jsInfo
		_, err = d.JsRequire.Require(jsInfo.Filename)
		d.JsLoadingScript = nil
	} else {
		d.Logger.Infof("脚本<%s>已被禁用，跳过加载", jsInfo.Name)
	}

	if err != nil {
		errText := err.Error()
		jsInfo.ErrText = errText
		jsInfo.Enable = false
		d.Logger.Error("读取脚本失败(解析失败): ", errText)
	}
}

func CheckJsSign(rawData []byte) (bool, SignStatus) {
	if OfficialModPublicKey == "" || len(rawData) == 0 {
		return false, UnknownSign
	}
	r := bufio.NewReader(bytes.NewReader(rawData))
	// 读取第一行判断签名
	fl, err := r.ReadBytes('\n')
	if err != nil {
		return false, UnknownSign
	}
	matches := signRe.FindSubmatch(fl)
	if len(matches) <= 1 {
		return false, UnknownSign
	}
	sign := string(matches[1])
	// 读取剩余内容
	data, err := io.ReadAll(r)
	if err != nil {
		return false, UnknownSign
	}
	err = crypto.RSAVerify(data, sign, OfficialModPublicKey)
	if err == nil {
		return true, OfficialSign
	}
	return false, ErrorSign
}

func JsDelete(_ *Dice, jsInfo *JsScriptInfo) {
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

func (d *Dice) JsCheckUpdate(jsScriptInfo *JsScriptInfo) (string, string, string, error) {
	// FIXME: dirty, copy from check deck update.
	if len(jsScriptInfo.UpdateUrls) == 0 {
		return "", "", "", fmt.Errorf("插件未提供更新链接")
	}

	statusCode, newData, err := GetCloudContent(jsScriptInfo.UpdateUrls, jsScriptInfo.Etag)
	if err != nil {
		return "", "", "", err
	}
	if statusCode == http.StatusNotModified {
		return "", "", "", fmt.Errorf("插件没有更新")
	}
	if statusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("未获取到插件更新")
	}
	oldData, err := os.ReadFile(jsScriptInfo.Filename)
	if err != nil {
		return "", "", "", err
	}

	// 内容预处理
	if isPrefixWithUtf8Bom(oldData) {
		oldData = oldData[3:]
	}
	oldJs := strings.ReplaceAll(string(oldData), "\r\n", "\n")
	if isPrefixWithUtf8Bom(newData) {
		newData = newData[3:]
	}
	newJs := strings.ReplaceAll(string(newData), "\r\n", "\n")

	temp, err := os.CreateTemp("", "new-*-"+filepath.Base(jsScriptInfo.Filename))
	if err != nil {
		return "", "", "", err
	}
	defer func(temp *os.File) {
		_ = temp.Close()
	}(temp)

	_, err = temp.WriteString(newJs)
	if err != nil {
		return "", "", "", err
	}
	return oldJs, newJs, temp.Name(), nil
}

func (d *Dice) JsUpdate(jsScriptInfo *JsScriptInfo, tempFileName string) error {
	newData, err := os.ReadFile(tempFileName)
	_ = os.Remove(tempFileName)
	if err != nil {
		return err
	}
	if len(newData) == 0 {
		return fmt.Errorf("new data is empty")
	}
	// 更新插件
	err = os.WriteFile(jsScriptInfo.Filename, newData, 0o755)
	if err != nil {
		d.Logger.Errorf("插件“%s”更新时保存文件出错，%s", jsScriptInfo.Name, err.Error())
		return err
	}
	d.Logger.Infof("插件“%s”更新成功", jsScriptInfo.Name)
	return nil
}

func checkJsScriptsDeps(jsScripts []*JsScriptInfo) ([]*JsScriptInfo, map[string][]string) {
	canLoad := make([]*JsScriptInfo, 0, len(jsScripts))
	invalidInfoMap := make(map[string][]string)
	scriptMap := make(map[string]*JsScriptInfo)
	for _, jsScript := range jsScripts {
		key := fmt.Sprintf("%s:%s", jsScript.Author, jsScript.Name)
		scriptMap[key] = jsScript
	}

	// 检查依赖是否存在，且是否符合版本要求
	for _, script := range jsScripts {
		key := script.Author + ":" + script.Name
		if len(script.Depends) > 0 {
			for _, dep := range script.Depends {
				// 依赖是否存在
				depKey := fmt.Sprintf("%s:%s", dep.Author, dep.Name)
				depScript, ok := scriptMap[depKey]
				if !ok {
					invalidInfoMap[key] = append(invalidInfoMap[key],
						fmt.Sprintf("「%s」依赖的「%s」不存在，所需版本：%s", key, depKey, dep.Constraint.String()))
					continue
				}
				// 版本是否符合要求
				depVersion, vErr := semver.NewVersion(depScript.Version)
				if vErr != nil {
					invalidInfoMap[key] = append(invalidInfoMap[key],
						fmt.Sprintf(
							"「%s」依赖的「%s」无法正确识别版本，现为：%s",
							key, depKey, depScript.Version,
						))
					continue
				}
				if !dep.Constraint.Check(depVersion) {
					invalidInfoMap[key] = append(invalidInfoMap[key], fmt.Sprintf(
						"「%s」依赖的「%s」版本不满足要求：要求 %s，现为 %s",
						key, depKey, dep.Constraint.String(), depScript.Version,
					))
					continue
				}
			}
		}
		if len(invalidInfoMap[key]) == 0 {
			canLoad = append(canLoad, script)
		} else {
			script.Enable = false
			script.ErrText = strings.Join(invalidInfoMap[key], "\n")
		}
	}
	return canLoad, invalidInfoMap
}

// sortJsScripts 使用 Kahn 算法分析依赖加载顺序，同时保证所有内置脚本均在外置脚本前加载
func sortJsScripts(jsScripts []*JsScriptInfo) ([]*JsScriptInfo, map[string][]string) {
	type boxedScript struct {
		key string
		js  *JsScriptInfo
	}

	var queue []*boxedScript
	relations := make(map[string][]string)
	inDegrees := make(map[string]int)
	vertices := make(map[string]*boxedScript)
	// 为了方便计算，添加一个 builtin 节点作为所有外置插件的依赖，其依赖所有内置插件
	dummy := "sealdice:_builtin"
	vertices[dummy] = &boxedScript{
		key: dummy,
	}
	inDegrees[dummy] = 0
	for _, jsScript := range jsScripts {
		key := fmt.Sprintf("%s:%s", jsScript.Author, jsScript.Name)
		if len(jsScript.Depends) > 0 {
			for _, dep := range jsScript.Depends {
				depKey := fmt.Sprintf("%s:%s", dep.Author, dep.Name)
				relations[depKey] = append(relations[depKey], key)
				inDegrees[key]++
			}
		}
		if jsScript.Builtin {
			relations[key] = append(relations[key], dummy)
			inDegrees[dummy]++
		} else {
			relations[dummy] = append(relations[dummy], key)
			inDegrees[key]++
		}

		vertices[key] = &boxedScript{
			key: key,
			js:  jsScript,
		}
	}

	for key, vertex := range vertices {
		if inDegrees[key] == 0 {
			queue = append(queue, vertex)
		}
	}
	var boxedResult []*boxedScript
	for len(queue) > 0 {
		vertex := queue[0]
		queue = queue[1:]
		boxedResult = append(boxedResult, vertex)
		for _, key := range relations[vertex.key] {
			inDegrees[key]--
			if inDegrees[key] == 0 {
				queue = append(queue, vertices[key])
			}
		}
	}

	// 是否入度都归零了，未归零说明存在循环依赖
	infos := make(map[string][]string)
	for key, inDegree := range inDegrees {
		script := vertices[key].js
		if inDegree != 0 && script != nil {
			var deps []string
			for _, dep := range script.Depends {
				deps = append(deps, dep.RawKey)
			}
			infos[key] = append(infos[key], fmt.Sprintf("「%s」存在循环依赖，请检查，依赖列表：%s", key, strings.Join(deps, "、")))
			script.Enable = false
			script.ErrText = strings.Join(infos[key], "\n")
		}
	}

	var result []*JsScriptInfo
	for _, boxed := range boxedResult {
		if boxed.js != nil {
			result = append(result, boxed.js)
		}
	}
	return result, infos
}
