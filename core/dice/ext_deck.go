// 抽牌模块
// Dice牌堆参考资料 https://forum.kokona.tech/d/167-json-pai-dui-bian-xie-cong-ru-men-dao-jin-jie

package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	wr "github.com/mroth/weightedrand"
	"github.com/pelletier/go-toml/v2"
	"github.com/sahilm/fuzzy"
	"github.com/tailscale/hujson"
	"gopkg.in/yaml.v3"
)

type DeckDiceEFormat struct {
	Title      []string `json:"_title"`
	Author     []string `json:"_author"`
	Date       []string `json:"_date"`
	UpdateDate []string `json:"_updateDate"`
	Brief      []string `json:"_brief"`
	Version    []string `json:"_version"`
	License    []string `json:"_license"`
	// Export  []string `json:"_export"` // 导出项，类似command
	// Keys  []string `json:"_keys"` // 导出项，类似command
	// 一组牌        []string `json:"一组牌"`

	// 更新支持字段
	UpdateUrls []string `json:"_updateUrls"`
	Etag       []string `json:"_etag"`
}

type DeckSinaNyaFormat struct {
	Name    string   `json:"name" yaml:"name"`
	Author  string   `json:"author" yaml:"author"`
	Version int      `json:"version" yaml:"version"`
	Command string   `json:"command" yaml:"command"`
	License string   `json:"license" yaml:"license"`
	Desc    string   `json:"desc" yaml:"desc"`
	Info    []string `json:"info" yaml:"info"`
	Default []string `json:"default" yaml:"default"`
	// 一组牌        []string `json:"一组牌"`

	// 更新支持字段
	UpdateUrls []string `json:"update_urls"`
	Etag       string   `json:"etag"`
}

type SealMeta struct {
	Title         string    `toml:"title"`
	Author        string    `toml:"author"`
	Authors       []string  `toml:"authors"`
	Version       string    `toml:"version"`
	License       string    `toml:"license"`
	Date          time.Time `toml:"date"`
	UpdateDate    time.Time `toml:"update_date"`
	Desc          string    `toml:"desc"`
	FormatVersion int64     `toml:"format_version"`

	UpdateUrls []string `toml:"update_urls"`
	Etag       string   `toml:"etag"`
}

type SealComplexSingleDeck struct {
	Export  bool     `toml:"export" mapstructure:"export"`
	Visible bool     `toml:"visible" mapstructure:"visible"`
	Aliases []string `toml:"aliases" mapstructure:"aliases"`
	Replace bool     `toml:"replace" mapstructure:"replace"` // 是否放回

	// 文本牌组项
	Options []string `toml:"options" mapstructure:"options"`

	// 云牌组项
	CloudExtra  bool     `toml:"cloud_extra" mapstructure:"cloud_extra"`
	Distinct    bool     `toml:"distinct" mapstructure:"distinct"`
	OptionsUrls []string `toml:"options_urls" mapstructure:"options_urls"`
}

type DeckSealFormat struct {
	Meta  SealMeta            `toml:"meta"`
	Decks map[string][]string `toml:"decks"`
}

type CloudDeckItemInfo struct {
	Distinct    bool
	OptionsUrls []string
}

type DeckInfo struct {
	Enable             bool                          `json:"enable" yaml:"enable"`
	ErrText            string                        `json:"errText" yaml:"errText"`
	Filename           string                        `json:"filename" yaml:"filename"`
	Format             string                        `json:"format" yaml:"format"`               // 几种：“SinaNya” ”Dice!“ "Seal"
	FormatVersion      int64                         `json:"formatVersion" yaml:"formatVersion"` // 格式版本，默认都是1
	FileFormat         string                        `json:"fileFormat" yaml:"-" `               // json / yaml / toml / jsonc
	Name               string                        `json:"name" yaml:"name"`
	Version            string                        `json:"version" yaml:"-"`
	Author             string                        `json:"author" yaml:"-"`
	License            string                        `json:"license" yaml:"-"` // 许可协议，如cc-by-nc等
	Command            map[string]bool               `json:"command" yaml:"-"` // 牌堆命令名
	DeckItems          map[string][]string           `yaml:"-" json:"-"`
	Date               string                        `json:"date" yaml:"-" `
	UpdateDate         string                        `json:"updateDate" yaml:"-" `
	Desc               string                        `yaml:"-" json:"desc"`
	Info               []string                      `yaml:"-" json:"-"`
	RawData            *map[string][]string          `yaml:"-" json:"-"`
	UpdateUrls         []string                      `yaml:"updateUrls" json:"updateUrls"`
	Etag               string                        `yaml:"etag" json:"etag"`
	Cloud              bool                          `yaml:"cloud" json:"cloud"` // 含有云端内容
	CloudDeckItemInfos map[string]*CloudDeckItemInfo `yaml:"-" json:"-"`
}

func tryParseDiceE(content []byte, deckInfo *DeckInfo, jsoncDirectly bool) error {
	// 移除注释
	standardJson, isRFC, err := standardizeJson(content)
	if err != nil {
		return err
	}

	jsonData := map[string][]string{}
	rawJsonData := map[string]any{}
	err = json.Unmarshal(standardJson, &rawJsonData)
	if err != nil {
		return err
	}
	for k, value := range rawJsonData {
		if k == "$schema" {
			continue
		} else if k == "helpdoc" {
			if _, ok := value.(map[string]any); ok {
				return errors.New("该文件疑似为帮助文档，而不是牌堆文件")
			}
		}
		if val, ok := value.([]any); ok {
			v := make([]string, 0, len(val))
			for _, elem := range val {
				if vv, ok := elem.(string); ok {
					v = append(v, vv)
				}
			}
			jsonData[k] = v
		}
	}

	jsonData2 := DeckDiceEFormat{}
	err = json.Unmarshal(standardJson, &jsonData2)
	if err != nil {
		return err
	}

	// 存在 _export
	exports, exportsExists := jsonData["_export"]
	if !exportsExists {
		exports, exportsExists = jsonData["_exports"]
	}
	if exportsExists {
		for _, i := range exports {
			deckInfo.Command[i] = true
		}
	}

	// 存在 _keys，仅显示keys但都能抽
	keysInfo, keysExists := jsonData["_keys"]
	if keysExists { //nolint:nestif
		for _, i := range keysInfo {
			deckInfo.Command[i] = true
		}

		for k, v := range jsonData {
			deckInfo.DeckItems[k] = v

			// 不存在 _export
			if !exportsExists {
				if !strings.HasPrefix(k, "_") {
					_, exists := deckInfo.Command[k]
					if !exists {
						deckInfo.Command[k] = false
					}
				}
			}
		}
	} else {
		// 不存在 _keys 默认为全部显示
		for k, v := range jsonData {
			deckInfo.DeckItems[k] = v

			// 不存在 _export
			if !exportsExists {
				if !strings.HasPrefix(k, "_") {
					deckInfo.Command[k] = true
				}
			}
		}
	}

	deckInfo.Name = strings.Join(jsonData2.Title, " / ")
	deckInfo.Author = strings.Join(jsonData2.Author, " / ")
	deckInfo.Version = strings.Join(jsonData2.Version, " / ")
	deckInfo.License = strings.Join(jsonData2.License, " / ")
	deckInfo.Date = strings.Join(jsonData2.Date, " / ")
	deckInfo.UpdateDate = strings.Join(jsonData2.UpdateDate, " / ")
	deckInfo.Desc = strings.Join(jsonData2.Brief, "\n")
	deckInfo.Format = "Dice!"
	deckInfo.FormatVersion = 1
	if !jsoncDirectly && isRFC {
		deckInfo.FileFormat = "json"
	} else {
		deckInfo.FileFormat = "jsonc"
	}
	deckInfo.Enable = true
	deckInfo.UpdateUrls = jsonData2.UpdateUrls
	if len(jsonData2.Etag) > 0 {
		deckInfo.Etag = jsonData2.Etag[0]
	}
	deckInfo.RawData = &jsonData
	return nil
}

func standardizeJson(src []byte) (converted []byte, isRFC bool, err error) {
	jsonValue, err := hujson.Parse(src)
	if err != nil {
		return nil, false, err
	}
	isRFC = jsonValue.IsStandard()
	if !isRFC {
		jsonValue.Standardize()
	}
	return jsonValue.Pack(), isRFC, nil
}

func tryParseSinaNya(content []byte, deckInfo *DeckInfo) error {
	yamlData := map[string]interface{}{}
	err := yaml.Unmarshal(content, &yamlData)
	if err != nil {
		return err
	}
	yamlData2 := DeckSinaNyaFormat{}
	err = yaml.Unmarshal(content, &yamlData2)
	if err != nil {
		return err
	}

	yamlDataFix := map[string][]string{}
	for k, v := range yamlData {
		vs1, ok := v.([]interface{})
		if ok {
			vs2 := make([]string, len(vs1))
			for i, v := range vs1 {
				vs2[i], _ = v.(string)
			}

			yamlDataFix[k] = vs2
		}
	}

	if yamlData2.Default != nil {
		deckInfo.Command[yamlData2.Command] = true
		deckInfo.DeckItems[yamlData2.Command] = yamlData2.Default
		for k, v := range yamlDataFix {
			deckInfo.DeckItems[k] = v
		}
	} else {
		for k, v := range yamlDataFix {
			deckInfo.DeckItems[k] = v
			deckInfo.Command[k] = true
		}
	}

	deckInfo.Name = yamlData2.Name
	deckInfo.Author = yamlData2.Author
	deckInfo.Version = strconv.Itoa(yamlData2.Version)
	deckInfo.License = yamlData2.License
	deckInfo.Desc = yamlData2.Desc
	deckInfo.Info = yamlData2.Info
	deckInfo.RawData = &yamlDataFix
	deckInfo.Format = "SinaNya"
	deckInfo.FormatVersion = 1
	deckInfo.FileFormat = "yaml"
	deckInfo.Enable = true
	deckInfo.UpdateUrls = yamlData2.UpdateUrls
	deckInfo.Etag = yamlData2.Etag
	return nil
}

func tryParseSeal(content []byte, deckInfo *DeckInfo) error {
	tomlData := map[string]interface{}{}
	err := toml.Unmarshal(content, &tomlData)
	if err != nil {
		return err
	}
	deckData := DeckSealFormat{}
	err = toml.Unmarshal(content, &deckData)
	if err != nil {
		return err
	}

	tomlDataFix := map[string][]string{}

	// 简单牌组
	for name, deckItems := range deckData.Decks {
		deckInfo.DeckItems[name] = deckItems
		tomlDataFix[name] = deckItems
		if strings.HasPrefix(name, "__") {
			continue
		} else if strings.HasPrefix(name, "_") {
			deckInfo.Command[name] = false
		} else {
			deckInfo.Command[name] = true
		}
	}

	// 复杂牌组
	for k, v := range tomlData {
		if k == "" || k == "meta" || k == "decks" {
			continue
		}

		item := SealComplexSingleDeck{}
		itemData, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		err := mapstructure.Decode(itemData, &item)
		if err == nil {
			deckItemName := k
			deckInfo.DeckItems[deckItemName] = item.Options
			tomlDataFix[deckItemName] = item.Options
			if !item.Export {
				continue
			} else if !item.Visible {
				deckInfo.Command[deckItemName] = false
			} else {
				deckInfo.Command[deckItemName] = true
			}

			// 别名
			fakeData := []string{"{" + deckItemName + "}"}
			for _, alias := range item.Aliases {
				deckInfo.DeckItems[alias] = fakeData
				tomlDataFix[alias] = fakeData
				if !item.Export {
					continue
				} else if !item.Visible {
					deckInfo.Command[alias] = false
				} else {
					deckInfo.Command[alias] = true
				}
			}

			// 云牌组项
			if item.CloudExtra {
				deckInfo.Cloud = true
				deckInfo.CloudDeckItemInfos[deckItemName] = &CloudDeckItemInfo{
					Distinct:    item.Distinct,
					OptionsUrls: item.OptionsUrls,
				}
			}
		}
	}

	meta := deckData.Meta
	deckInfo.Name = meta.Title
	var author string
	if meta.Authors != nil {
		author = strings.Join(meta.Authors, " / ")
	} else {
		author = meta.Author
	}
	if meta.FormatVersion == 0 {
		meta.FormatVersion = 1
	}

	deckInfo.Author = author
	deckInfo.Version = meta.Version
	deckInfo.License = meta.License
	deckInfo.Date = meta.Date.Format("2006-01-02")
	deckInfo.UpdateDate = meta.UpdateDate.Format("2006-01-02")
	deckInfo.Desc = meta.Desc
	deckInfo.Format = "Seal"
	deckInfo.FormatVersion = meta.FormatVersion
	deckInfo.FileFormat = "toml"
	deckInfo.Enable = true
	deckInfo.UpdateUrls = meta.UpdateUrls
	deckInfo.Etag = meta.Etag
	deckInfo.RawData = &tomlDataFix
	return nil
}

func isPrefixWithUtf8Bom(buf []byte) bool {
	return len(buf) >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF
}

func DeckTryParse(d *Dice, fn string) {
	content, err := os.ReadFile(fn)
	if err != nil {
		d.Logger.Infof("牌堆文件“%s”加载失败", fn)
		return
	}
	deckInfo := new(DeckInfo)
	if deckInfo.DeckItems == nil {
		deckInfo.DeckItems = map[string][]string{}
	}
	if deckInfo.Command == nil {
		deckInfo.Command = map[string]bool{}
	}
	if deckInfo.CloudDeckItemInfos == nil {
		deckInfo.CloudDeckItemInfos = map[string]*CloudDeckItemInfo{}
	}
	_ = parseDeck(d, fn, content, deckInfo)
	deckInfo.Filename = fn

	if deckInfo.Name == "" {
		deckInfo.Name = filepath.Base(fn)
	}

	d.DeckList = append(d.DeckList, deckInfo)
	d.MarkModified()
}

func parseDeck(d *Dice, fn string, content []byte, deckInfo *DeckInfo) bool {
	if isPrefixWithUtf8Bom(content) {
		content = content[3:]
	}
	ext := strings.ToLower(path.Ext(fn))
	var err error

	switch ext {
	case ".json":
		err = tryParseDiceE(content, deckInfo, false)
	case ".jsonc":
		err = tryParseDiceE(content, deckInfo, true)
	case ".yaml", ".yml":
		err = tryParseSinaNya(content, deckInfo)
	case ".toml":
		err = tryParseSeal(content, deckInfo)
	default:
		d.Logger.Infof("牌堆文件“%s”是未知格式，尝试以json和yaml格式解析", fn)
		if tryParseDiceE(content, deckInfo, false) != nil {
			err = tryParseSinaNya(content, deckInfo)
		}
	}

	if err != nil {
		d.Logger.Errorf("牌堆文件“%s”解析失败 %v", fn, err)
		deckInfo.Enable = false
		deckInfo.ErrText = err.Error()
		return false
	}
	return true
}

// DecksDetect 检查牌堆
func DecksDetect(d *Dice) {
	// 先进行zip解压
	_ = filepath.Walk("data/decks", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && strings.EqualFold(info.Name(), "assets") {
			return fs.SkipDir
		}
		if info.IsDir() && strings.EqualFold(info.Name(), "images") {
			return fs.SkipDir
		}

		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), ".deck") {
				dest := filepath.Join(filepath.Dir(path), "_"+info.Name())
				if _, err := os.Stat(dest); err != nil {
					d.Logger.Info("检测到可能是新的压缩牌堆文件，准备自动解压:", info.Name())
					if isDeckFile(path) {
						_ = unzipSource(path, dest)
					} else {
						d.Logger.Info("目标并非压缩牌堆文件:", info.Name())
					}
				}
			}
		}

		return nil
	})

	_ = filepath.Walk("data/decks", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && strings.EqualFold(info.Name(), "assets") {
			return fs.SkipDir
		}
		if info.IsDir() && strings.EqualFold(info.Name(), "images") {
			return fs.SkipDir
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), ".deck") {
			return nil
		}
		if info.Name() == "info.yaml" {
			return nil
		}

		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".json" || ext == ".jsonc" || ext == ".yml" || ext == ".yaml" || ext == ".toml" || ext == "" {
				DeckTryParse(d, path)
			}
		}
		return nil
	})
}

func DeckDelete(_ *Dice, deck *DeckInfo) {
	dirpath := filepath.Dir(deck.Filename)
	dirname := filepath.Base(dirpath)

	if strings.HasPrefix(dirname, "_") && strings.HasSuffix(dirname, ".deck") {
		// 可能是zip解压出来的，那么删除目录和压缩包
		_ = os.RemoveAll(dirpath)
		zipFilename := filepath.Join(filepath.Dir(dirpath), dirname[1:])
		_ = os.Remove(zipFilename)
	} else {
		_ = os.Remove(deck.Filename)
	}
}

type DeckCommandListItem struct {
	command string
	deck    *DeckInfo
	visible bool // 是否可见: 指定了_key列表的牌堆, 没有在_key中的visible=false
}

type DeckCommandListItems []*DeckCommandListItem

func (e DeckCommandListItems) String(i int) string {
	// 如果可见, 返回牌组名; 否则, 返回空字符串以避免模糊匹配命中
	if e[i].visible {
		return e[i].command
	}
	return ""
}

func (e DeckCommandListItems) Len() int {
	return len(e)
}

func DeckReload(d *Dice) {
	if d.IsDeckLoading {
		return
	}
	d.IsDeckLoading = true
	d.DeckList = d.DeckList[:0]
	d.Logger.Infof("从此目录加载牌堆: %s", "data/decks")
	DecksDetect(d)
	d.Logger.Infof("加载完成，现有牌堆 %d 个", len(d.DeckList))
	d.IsDeckLoading = false
	d.MarkModified()

	lst := DeckCommandListItems{}
	for _, i := range d.DeckList {
		for k, v := range i.Command {
			lst = append(lst, &DeckCommandListItem{
				command: k,
				deck:    i,
				visible: v,
			})
		}
	}
	d.deckCommandItemsList = lst
}

func deckDraw(ctx *MsgContext, deckName string, shufflePool bool) (bool, string, error) {
	for _, i := range ctx.Dice.DeckList {
		if i.Enable {
			_, deckExists := i.Command[deckName]
			if deckExists {
				a, b := executeDeck(ctx, i, deckName, shufflePool)
				return true, a, b
			}
		}
	}
	return false, "", nil
}

func RegisterBuiltinExtDeck(d *Dice) {
	helpDraw := "" +
		".draw help // 显示本帮助\n" +
		".draw keys // 查看可抽取的牌组列表\n" +
		".draw keys <牌堆> // 查看特定牌堆可抽取的牌组列表\n" +
		".draw search <牌组名称> // 搜索相关牌组\n" +
		".draw reload // 从硬盘重新装载牌堆，仅Master可用\n" +
		".draw list // 查看载入的牌堆文件\n" +
		".draw <牌组名称> // 进行抽牌"

	cmdDraw := &CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "draw",
		ShortHelp:               helpDraw,
		Help:                    "抽牌命令: \n" + helpDraw,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if d.IsDeckLoading {
				ReplyToSender(ctx, msg, "牌堆尚未就绪，可能正在重新装载")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			cmdArgs.ChopPrefixToArgsWith("list", "help", "reload", "search", "keys", "desc")
			deckName := cmdArgs.GetArgN(1)

			if deckName == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			if strings.EqualFold(deckName, "list") { //nolint:nestif
				text := "载入并开启的牌堆:\n"
				for _, i := range ctx.Dice.DeckList {
					if i.Enable {
						author := fmt.Sprintf(" 作者:%s", i.Author)
						version := fmt.Sprintf(" 版本:%s", i.Version)
						count := 0
						for _, vis := range i.Command {
							if vis {
								count++
							}
						}
						text += fmt.Sprintf("- %s 格式: %s%s%s 牌组数量: %d\n", i.Name, i.Format, author, version, count)
					}
				}
				ReplyToSender(ctx, msg, text)
			} else if strings.EqualFold(deckName, "desc") {
				// 查看详情
				text := cmdArgs.GetArgN(2)
				matches := fuzzy.FindFrom(text, d.deckCommandItemsList)
				if len(matches) > 0 {
					text := "牌堆信息:\n"
					i := ctx.Dice.deckCommandItemsList[matches[0].Index].deck
					author := fmt.Sprintf("作者: %s", i.Author)
					version := fmt.Sprintf("版本: %s", i.Version)
					var cmds []string
					for j, vis := range i.Command {
						if vis {
							cmds = append(cmds, j)
						}
					}
					text += fmt.Sprintf("牌堆: %s\n格式: %s\n%s\n%s\n牌组数量: %d\n", i.Name, i.Format, author, version, len(cmds))
					if i.Date != "" {
						time := fmt.Sprintf("时间: %s\n", i.Date)
						text += time
					}
					if i.UpdateDate != "" {
						time := fmt.Sprintf("更新时间: %s\n", i.UpdateDate)
						text += time
					}

					text += "牌组: " + strings.Join(cmds, "/")
					ReplyToSender(ctx, msg, text)
				} else {
					ReplyToSender(ctx, msg, "此关键字未找到牌堆")
				}
			} else if strings.EqualFold(deckName, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			} else if strings.EqualFold(deckName, "keys") {
				specified := cmdArgs.GetArgN(2)
				text := "牌组关键字列表:\n"
				var keys []string
				for _, i := range ctx.Dice.DeckList {
					if i.Enable {
						if strings.Contains(i.Name, specified) {
							for j, isShow := range i.Command {
								if isShow {
									keys = append(keys, j)
								}
							}
						}
					}
				}
				VarSetValueStr(ctx, "$t原始列表", strings.Join(keys, "/"))

				keysStr := DiceFormatTmpl(ctx, "其它:抽牌_列表")
				if keysStr == "" {
					text += DiceFormatTmpl(ctx, "其它:抽牌_列表_没有牌组")
				} else {
					text += keysStr
				}
				ReplyToSender(ctx, msg, text)
			} else if strings.EqualFold(deckName, "reload") {
				if ctx.PrivilegeLevel < 100 {
					ReplyToSender(ctx, msg, "你不具备Master权限")
				} else {
					if ctx.Dice.Parent.JustForTest {
						ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					DeckReload(d)
					ReplyToSender(ctx, msg, "牌堆已经重新装载")
				}
			} else if strings.EqualFold(deckName, "search") {
				text := cmdArgs.GetArgN(2)
				if text != "" {
					matches := fuzzy.FindFrom(text, d.deckCommandItemsList)

					right := len(matches)
					if right == 0 {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "其它:抽牌_找不到牌组"))
					} else {
						if right > 10 {
							right = 3
						}
						reply := "找到以下牌组:\n"
						for _, i := range matches[:right] {
							reply += "- " + i.Str + "\n"
						}
						ReplyToSender(ctx, msg, reply)
					}
				} else {
					ReplyToSender(ctx, msg, "请给出要搜索的关键字")
				}
			} else {
				exists, result, err := deckDraw(ctx, deckName, true)
				if err != nil {
					result = fmt.Sprintf("<%s>", err.Error())
				}
				VarSetValueStr(ctx, "$t牌组", deckName)

				if exists {
					results := []string{result}

					// 多抽限额为5
					var times int
					if t := cmdArgs.GetArgN(2); t != "" {
						times64, _ := strconv.ParseInt(t, 10, 64)
						times = int(times64)
					}
					if cmdArgs.SpecialExecuteTimes > times {
						times = cmdArgs.SpecialExecuteTimes
					}

					// 信任骰主设置的执行次数上限
					// if times > 5 {
					// 	times = 5
					// }
					if times > int(ctx.Dice.MaxExecuteTime) {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰点_轮数过多警告"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					for i := 1; i < times; i++ {
						_, r2, errDraw := deckDraw(ctx, deckName, true)
						if errDraw != nil {
							r2 = fmt.Sprintf("<%s>", errDraw.Error())
						}
						results = append(results, r2)
					}

					sep := DiceFormatTmpl(ctx, "其它:抽牌_分隔符")
					result = strings.Join(results, sep)
					prefix := DiceFormatTmpl(ctx, "其它:抽牌_结果前缀")
					ReplyToSender(ctx, msg, prefix+result)
				} else {
					matches := fuzzy.FindFrom(deckName, d.deckCommandItemsList)
					right := len(matches)
					if right > 10 {
						right = 4
					}
					if right > 0 {
						text := DiceFormatTmpl(ctx, "其它:抽牌_找不到牌组_存在类似")
						text += "\n"
						for _, i := range matches[:right] {
							text += "- " + i.Str + "\n"
						}
						ReplyToSender(ctx, msg, text)
					} else {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "其它:抽牌_找不到牌组"))
					}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	theExt := &ExtInfo{
		Name:       "deck", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Author:     "木落",
		Brief:      "牌堆扩展，提供.deck指令支持，兼容Dice!和塔系牌堆",
		Official:   true,
		AutoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		OnLoad: func() {
			DeckReload(d)
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"draw": cmdDraw,
			"deck": cmdDraw,
		},
	}

	d.RegisterExtension(theExt)
}

func deckStringFormat(ctx *MsgContext, deckInfo *DeckInfo, s string) (string, error) {
	//  ***n***
	// 规则参考: https://sinanya.com/#/MakeFile
	// 1. 提取 {}
	re := regexp.MustCompile(`{[$%]?.+?}`)
	s = CompatibleReplace(ctx, s)
	m := re.FindAllStringIndex(s, -1)

	for _i := len(m) - 1; _i >= 0; _i-- {
		i := m[_i]

		var text string
		var err error

		deckName := s[i[0]:i[1]]

		// 特殊变量检查
		if deckName == "{player}" {
			if ctx.Player != nil {
				text = ctx.Player.Name
			} else {
				text = "%未知用户%"
			}
			s = s[:i[0]] + text + s[i[1]:]
			continue
		}
		if deckName == "{self}" {
			text = DiceFormatTmpl(ctx, "核心:骰子名字")
			s = s[:i[0]] + text + s[i[1]:]
			continue
		}

		sign := deckName[1]
		signLength := 0
		hasSign := sign == '$' || sign == '%'
		if hasSign {
			signLength++
		}

		deckName = deckName[1+signLength : len(deckName)-1]

		deck := deckInfo.DeckItems[deckName]

		if deck == nil {
			text = "<%未知牌组-" + deckName + "%>"
		} else {
			useShufflePool := sign != '%'
			if deckInfo.Format == "SinaNya" {
				useShufflePool = !useShufflePool
			}

			text, err = executeDeck(ctx, deckInfo, deckName, useShufflePool)
			if err != nil {
				if err.Error() == "超出遍历次数" {
					text = "<%抽取错误-可能死循环-" + deckName + "%>"
					// 触发错误，回滚文本。避免出现:
					//  draw keys能看到我 draw keys能看到我 draw keys能看到我 draw keys能看到我 <%抽取错误-超出遍历次数-导出牌组%>
					return text, err
				}
				text = "<%抽取错误-" + deckName + "%>"
			}
		}

		s = s[:i[0]] + text + s[i[1]:]
	}

	s = strings.ReplaceAll(s, "【name】", "{$t玩家}")
	s = strings.ReplaceAll(s, "[name]", "{$t玩家}")

	re = regexp.MustCompile(`\[.+?]`)
	m = re.FindAllStringIndex(s, -1)

	cqSolve := func(cq *CQCommand) {
		fn, exists := cq.Args["file"]
		if exists {
			if strings.HasPrefix(fn, "./") {
				pathPrefix, err := filepath.Rel(".", filepath.Dir(deckInfo.Filename))
				if err == nil {
					fn = filepath.Join(pathPrefix, fn[2:])
					fn = strings.ReplaceAll(fn, `\`, "/")
					cq.Args["file"] = fn
				}
			}
		}
	}

	imgSolve := func(text string) string {
		reImg := regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`) // [img:] 或 [图:]
		m := reImg.FindStringSubmatch(text)
		if m != nil {
			fn := m[2]
			if strings.HasPrefix(fn, "./") {
				pathPrefix, err := filepath.Rel(".", filepath.Dir(deckInfo.Filename))
				if err == nil {
					fn = filepath.Join(pathPrefix, fn[2:])
					fn = strings.ReplaceAll(fn, `\`, "/")
					return "[" + m[1] + ":" + fn + "]"
				}
			}
		}
		return text
	}

	for _i := len(m) - 1; _i >= 0; _i-- {
		i := m[_i]

		text := s[i[0]:i[1]]
		if strings.HasPrefix(text, "[CQ:") {
			continue
		}

		if strings.HasPrefix(text, "[图:") ||
			strings.HasPrefix(text, "[img:") ||
			strings.HasPrefix(text, "[文本:") ||
			strings.HasPrefix(text, "[语音:") {
			continue
		}

		runeText, rest := extractExecuteContent(s[i[0]:])
		text = "{" + runeText + "}"
		if runeText == "" {
			text = ""
		}
		s = s[:i[0]] + text + rest
		// fmt.Println("!!!!", text, "|", rest)
		// text = "{" + text[1:len(text)-1] + "}"
		// s = s[:i[0]] + text + s[i[1]:]
	}

	s = CQRewrite(s, cqSolve)
	s = ImageRewrite(s, imgSolve)

	s = strings.ReplaceAll(s, "\n", `\n`)
	return DiceFormat(ctx, s), nil
}

func executeDeck(ctx *MsgContext, deckInfo *DeckInfo, deckName string, shufflePool bool) (string, error) {
	var key string
	ctx.deckDepth++
	if ctx.deckDepth > 20000 {
		return "", errors.New("超出遍历次数")
	}

	if shufflePool {
		var pool *ShuffleRandomPool
		if ctx.DeckPools == nil {
			ctx.DeckPools = map[*DeckInfo]map[string]*ShuffleRandomPool{}
		}

		if ctx.DeckPools[deckInfo] == nil {
			ctx.DeckPools[deckInfo] = map[string]*ShuffleRandomPool{}
		}

		deckGroup := getDeckGroup(deckInfo, deckName)
		if len(deckGroup) == 0 {
			return "", errors.New("牌组为空，请检查格式是否正确")
		}
		if ctx.DeckPools[deckInfo][deckName] == nil {
			ctx.DeckPools[deckInfo][deckName] = DeckToShuffleRandomPool(deckGroup)
		}

		if len(ctx.DeckPools[deckInfo][deckName].data) == 0 {
			ctx.DeckPools[deckInfo][deckName] = DeckToShuffleRandomPool(deckGroup)
		}

		pool = ctx.DeckPools[deckInfo][deckName]
		if pool == nil {
			return "", errors.New("牌组为空，可能尚未加载完成")
		}
		key = pool.Pick().(string)
	} else {
		deckGroup := getDeckGroup(deckInfo, deckName)
		if len(deckGroup) == 0 {
			return "", errors.New("牌组为空，请检查格式是否正确")
		}
		pool := DeckToRandomPool(deckGroup)
		if pool == nil {
			return "", errors.New("牌组为空，可能尚未加载完成")
		}
		key = pool.Pick().(string)
	}
	cmd, err := deckStringFormat(ctx, deckInfo, key)
	return cmd, err
}

func getDeckGroup(deckInfo *DeckInfo, deckName string) (deckGroup []string) {
	deckGroup = deckInfo.DeckItems[deckName]
	if !deckInfo.Cloud {
		return
	}

	// 含有云端内容时，查看是否需要补充
	cloudInfo, ok := deckInfo.CloudDeckItemInfos[deckName]
	if !ok {
		return
	}

	statusCode, newData, err := GetCloudContent(cloudInfo.OptionsUrls, "")
	if err != nil || statusCode != http.StatusOK {
		return
	}
	cloudItems := make([]string, 0)
	err = json.Unmarshal(newData, &cloudItems)
	if err != nil {
		return
	}
	deckGroup = append(deckGroup, cloudItems...)
	if cloudInfo.Distinct {
		// 内容去重
		tempSet := map[string]bool{}
		for _, item := range deckGroup {
			tempSet[item] = true
		}
		temp := make([]string, 0, len(tempSet))
		for item := range tempSet {
			temp = append(temp, item)
		}
		deckGroup = temp
	}
	return
}

func extractWeight(s string) (uint, string) {
	weight := int64(1)
	re := regexp.MustCompile(`^::(\d+)::`)
	m := re.FindStringSubmatch(s)
	if m != nil {
		weight, _ = strconv.ParseInt(m[1], 10, 64)
		s = s[len(m[0]):]
	}
	return uint(weight), s
}

func DeckToRandomPool(deck []string) *wr.Chooser {
	choices := []wr.Choice{}
	for _, i := range deck {
		weight, text := extractWeight(i)
		choices = append(choices, wr.Choice{Item: text, Weight: weight})
	}
	randomPool, _ := wr.NewChooser(choices...)
	return randomPool
}

// 临时乱写的
type ShuffleRandomPool struct {
	data   []wr.Choice
	totals []int
	max    int
}

func NewChooser(choices ...wr.Choice) (*ShuffleRandomPool, error) {
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})

	totals := make([]int, len(choices))
	runningTotal := 0
	for i, c := range choices {
		weight := int(c.Weight)
		// if (maxInt - runningTotal) <= weight {
		// 	return nil, errWeightOverflow
		// }
		runningTotal += weight
		totals[i] = runningTotal
	}

	if runningTotal < 1 {
		return nil, errors.New("zero Choices with Weight >= 1")
	}

	return &ShuffleRandomPool{data: choices, totals: totals, max: runningTotal}, nil
}

// Pick returns a single weighted random Choice.Item from the Chooser.
//
// Utilizes global rand as the source of randomness.
func (c *ShuffleRandomPool) Pick() interface{} {
	r := rand.Intn(c.max) + 1
	i := searchInts(c.totals, r)

	theOne := c.data[i]
	c.max -= int(theOne.Weight)
	c.totals = append(c.totals[:i], c.totals[i+1:]...)
	c.data = append(c.data[:i], c.data[i+1:]...)
	return theOne.Item
}

func searchInts(a []int, x int) int {
	// Possible further future optimization for searchInts via SIMD if we want
	// to write some Go assembly code: http://0x80.pl/articles/simd-search.html
	i, j := 0, len(a)
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		if a[h] < x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

func DeckToShuffleRandomPool(deck []string) *ShuffleRandomPool {
	var choices []wr.Choice
	for _, i := range deck {
		weight, text := extractWeight(i)
		choices = append(choices, wr.Choice{Item: text, Weight: weight})
	}
	randomPool, _ := NewChooser(choices...)
	return randomPool
}

// extractExecuteContent 抽出需要执行代码的部分，支持嵌套，返回的第一项为匹配内容。第二项为剩余文本
func extractExecuteContent(s string) (string, string) {
	start := strings.Index(s, "[")
	if start == -1 {
		return "", ""
	}
	end := start
	count := 1
	for i := start + 1; i < len(s); i++ {
		switch s[i] {
		case '[':
			count++
		case ']':
			count--
		}
		if count == 0 {
			end = i
			break
		}
	}
	if end == 0 {
		return "", s
	}
	return s[start+1 : end], s[end+1:]
}

func (d *Dice) DeckCheckUpdate(deckInfo *DeckInfo) (string, string, string, error) {
	if len(deckInfo.UpdateUrls) == 0 {
		return "", "", "", fmt.Errorf("牌堆未提供更新链接")
	}

	statusCode, newData, err := GetCloudContent(deckInfo.UpdateUrls, deckInfo.Etag)
	if err != nil {
		return "", "", "", err
	}
	if statusCode == http.StatusNotModified {
		return "", "", "", fmt.Errorf("牌堆没有更新")
	}
	if statusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("未获取到牌堆更新")
	}

	oldData, err := os.ReadFile(deckInfo.Filename)
	if err != nil {
		return "", "", "", err
	}

	// 内容预处理
	if isPrefixWithUtf8Bom(oldData) {
		oldData = oldData[3:]
	}
	oldDeck := strings.ReplaceAll(string(oldData), "\r\n", "\n")
	if isPrefixWithUtf8Bom(newData) {
		newData = newData[3:]
	}
	newDeck := strings.ReplaceAll(string(newData), "\r\n", "\n")

	temp, err := os.CreateTemp("", "new-*-"+filepath.Base(deckInfo.Filename))
	if err != nil {
		return "", "", "", err
	}
	defer func(temp *os.File) {
		_ = temp.Close()
	}(temp)

	_, err = temp.WriteString(newDeck)
	if err != nil {
		return "", "", "", err
	}
	return oldDeck, newDeck, temp.Name(), nil
}

func (d *Dice) DeckUpdate(deckInfo *DeckInfo, tempFileName string) error {
	newData, err := os.ReadFile(tempFileName)
	_ = os.Remove(tempFileName)
	if err != nil {
		return err
	}
	if len(newData) == 0 {
		return fmt.Errorf("new data is empty")
	}
	// 更新牌堆
	ok := parseDeck(d, tempFileName, newData, deckInfo)
	if !ok {
		d.Logger.Errorf("牌堆“%s”更新失败，无法解析获取到的牌堆数据", deckInfo.Name)
		return fmt.Errorf("无法解析获取到的牌堆数据")
	}

	err = os.WriteFile(deckInfo.Filename, newData, 0755)
	if err != nil {
		d.Logger.Errorf("牌堆“%s”更新时保存文件出错，%s", deckInfo.Name, err.Error())
		return err
	}
	d.Logger.Infof("牌堆“%s”更新成功", deckInfo.Name)
	return nil
}
