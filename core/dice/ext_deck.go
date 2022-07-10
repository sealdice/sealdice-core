// 抽牌模块
// Dice牌堆参考资料 https://forum.kokona.tech/d/167-json-pai-dui-bian-xie-cong-ru-men-dao-jin-jie

package dice

import (
	"encoding/json"
	"fmt"
	wr "github.com/mroth/weightedrand"
	"github.com/sahilm/fuzzy"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type DeckDiceEFormat struct {
	Title   []string `json:"_title"`
	Author  []string `json:"_author"`
	Date    []string `json:"_date"`
	Version []string `json:"_version"`
	//Export  []string `json:"_export"` // 导出项，类似command
	//一组牌        []string `json:"一组牌"`
}

type DeckSinaNyaFormat struct {
	Name    string   `json:"name" yaml:"name"`
	Author  string   `json:"author" yaml:"author"`
	Version int      `json:"version" yaml:"version"`
	Command string   `json:"command" yaml:"command"`
	Desc    string   `json:"desc" yaml:"desc"`
	Info    []string `json:"info" yaml:"info"`
	Default []string `json:"default" yaml:"default"`
	//一组牌        []string `json:"一组牌"`
}

type DeckInfo struct {
	Enable        bool                 `json:"enable" yaml:"enable"`
	Filename      string               `json:"filename" yaml:"filename"`
	Format        string               `json:"format" yaml:"format"`               // 几种：“SinaNya” ”Dice!“
	FormatVersion int64                `json:"formatVersion" yaml:"formatVersion"` // 格式版本，默认都是1
	FileFormat    string               `json:"fileFormat" yaml:"-" `               // json / yaml
	Name          string               `json:"name" yaml:"name"`
	Version       string               `json:"version" yaml:"-"`
	Author        string               `json:"author" yaml:"-"`
	Command       map[string]bool      `json:"command" yaml:"-"` // 牌堆命令名
	DeckItems     map[string][]string  `yaml:"-" json:"-"`
	Date          string               `json:"date" yaml:"-" `
	Desc          string               `yaml:"-" json:"desc"`
	Info          []string             `yaml:"-" json:"-"`
	rawData       *map[string][]string `yaml:"-" json:"-"`
}

type DeckInfoCommandList []string

func (e DeckInfoCommandList) String(i int) string {
	return e[i]
}

func (e DeckInfoCommandList) Len() int {
	return len(e)
}

func tryParseDiceE(d *Dice, content []byte, deckInfo *DeckInfo) bool {
	jsonData := map[string][]string{}
	err := json.Unmarshal(content, &jsonData)
	if err != nil {
		//fmt.Println("牌堆解析:",  err)
		return false
	}
	jsonData2 := DeckDiceEFormat{}
	err = json.Unmarshal(content, &jsonData2)
	if err != nil {
		return false
	}

	// 存在 _export
	exports, exists := jsonData["_export"]
	if exists {
		for _, i := range exports {
			deckInfo.Command[i] = true
		}
	}

	for k, v := range jsonData {
		deckInfo.DeckItems[k] = v

		// 不存在 _export
		if !exists {
			if !strings.HasPrefix(k, "_") {
				deckInfo.Command[k] = true
			}
		}
	}

	deckInfo.Name = strings.Join(jsonData2.Title, " / ")
	deckInfo.Author = strings.Join(jsonData2.Author, " / ")
	deckInfo.Version = strings.Join(jsonData2.Version, " / ")
	deckInfo.Date = strings.Join(jsonData2.Date, " / ")
	deckInfo.Format = "Dice!"
	deckInfo.FormatVersion = 1
	deckInfo.Enable = true
	deckInfo.rawData = &jsonData
	return true
}

func tryParseSinaNya(d *Dice, content []byte, deckInfo *DeckInfo) bool {
	jsonData := map[string]interface{}{}
	err := yaml.Unmarshal(content, &jsonData)
	if err != nil {
		return false
	}
	jsonData2 := DeckSinaNyaFormat{}
	err = yaml.Unmarshal(content, &jsonData2)
	if err != nil {
		return false
	}

	jsonDataFix := map[string][]string{}
	for k, v := range jsonData {
		vs1, ok := v.([]interface{})
		if ok {
			vs2 := make([]string, len(vs1))
			for i, v := range vs1 {
				vs2[i], _ = v.(string)
			}

			jsonDataFix[k] = vs2
		}
	}

	if jsonData2.Default != nil {
		deckInfo.Command[jsonData2.Command] = true
		deckInfo.DeckItems[jsonData2.Command] = jsonData2.Default
		for k, v := range jsonDataFix {
			deckInfo.DeckItems[k] = v
		}
	} else {
		for k, v := range jsonDataFix {
			deckInfo.DeckItems[k] = v
			deckInfo.Command[k] = true
		}
	}

	deckInfo.Name = jsonData2.Name
	deckInfo.Author = jsonData2.Author
	deckInfo.Version = strconv.Itoa(jsonData2.Version)
	deckInfo.Desc = jsonData2.Desc
	deckInfo.Info = jsonData2.Info
	deckInfo.rawData = &jsonDataFix
	deckInfo.Format = "SinaNya"
	deckInfo.FormatVersion = 1
	deckInfo.Enable = true
	return true
}

func isPrefixWithUtf8Bom(buf []byte) bool {
	return len(buf) >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF
}

func DeckTryParse(d *Dice, fn string) {
	content, err := ioutil.ReadFile(fn)
	if err != nil {
		d.Logger.Infof("牌堆文件“%s”加载失败", fn)
		return
	}
	if isPrefixWithUtf8Bom(content) {
		content = content[3:]
	}

	deckInfo := new(DeckInfo)
	if deckInfo.DeckItems == nil {
		deckInfo.DeckItems = map[string][]string{}
	}
	if deckInfo.Command == nil {
		deckInfo.Command = map[string]bool{}
	}

	if !tryParseDiceE(d, content, deckInfo) {
		if !tryParseSinaNya(d, content, deckInfo) {
			d.Logger.Infof("牌堆文件“%s”解析失败", fn)
			return
		}
	}
	deckInfo.Filename = fn

	if deckInfo.Name == "" {
		deckInfo.Name = filepath.Base(fn)
	}

	d.DeckList = append(d.DeckList, deckInfo)
}

// DecksDetect 检查牌堆
func DecksDetect(d *Dice) {
	// 先进行zip解压
	filepath.Walk("data/decks", func(path string, info fs.FileInfo, err error) error {
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

	filepath.Walk("data/decks", func(path string, info fs.FileInfo, err error) error {
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
			if ext == ".json" || ext == ".yaml" || ext == "" {
				DeckTryParse(d, path)
			}
		}
		return nil
	})
}

func DeckDelete(d *Dice, deck *DeckInfo) {
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
}

func RegisterBuiltinExtDeck(d *Dice) {
	helpDraw := "" +
		".draw help // 显示本帮助\n" +
		".draw list // 查看载入的牌堆文件\n" +
		".draw keys // 查看可抽取的牌组列表(容易很长，不建议用)\n" +
		".draw keys <牌堆> // 查看特定牌堆可抽取的牌组列表\n" +
		".draw search <牌组名称> // 搜索相关牌组\n" +
		".draw reload // 从硬盘重新装载牌堆，仅Master可用\n" +
		".draw <牌组名称> // 进行抽牌"

	cmdDraw := &CmdItemInfo{
		Name:     "draw",
		Help:     helpDraw,
		LongHelp: "抽牌命令: \n" + helpDraw,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				if d.IsDeckLoading {
					ReplyToSender(ctx, msg, "牌堆尚未就绪，可能正在重新装载")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				cmdArgs.ChopPrefixToArgsWith("list", "help", "reload", "search", "keys")
				deckName, exists := cmdArgs.GetArgN(1)

				if exists {
					if strings.EqualFold(deckName, "list") {
						text := "载入并开启的牌堆:\n"
						for _, i := range ctx.Dice.DeckList {
							if i.Enable {
								author := fmt.Sprintf(" 作者:%s", i.Author)
								version := fmt.Sprintf(" 版本:%s", i.Version)
								text += fmt.Sprintf("- %s 格式: %s%s%s 牌组数量: %d\n", i.Name, i.Format, author, version, len(i.Command))
							}
						}
						ReplyToSender(ctx, msg, text)
					} else if strings.EqualFold(deckName, "help") {
						return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
					} else if strings.EqualFold(deckName, "keys") {
						specified, _ := cmdArgs.GetArgN(2)
						text := "牌组关键字列表:\n"
						keys := ""
						for _, i := range ctx.Dice.DeckList {
							if i.Enable {
								if strings.Contains(i.Name, specified) {
									for j := range i.Command {
										keys += j + "/"
									}
								}
							}
						}
						if keys == "" {
							text += DiceFormatTmpl(ctx, "其它:抽牌_列表_没有牌组")
						} else {
							text += keys[:len(keys)-1]
						}
						ReplyToSender(ctx, msg, text)
					} else if strings.EqualFold(deckName, "reload") {
						if ctx.PrivilegeLevel < 100 {
							ReplyToSender(ctx, msg, fmt.Sprintf("你不具备Master权限"))
						} else {
							if ctx.Dice.Parent.JustForTest {
								ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
								return CmdExecuteResult{Matched: true, Solved: true}
							}

							DeckReload(d)
							ReplyToSender(ctx, msg, "牌堆已经重新装载")
						}
					} else if strings.EqualFold(deckName, "search") {
						text, exists := cmdArgs.GetArgN(2)
						if exists {
							var lst DeckInfoCommandList
							for _, i := range ctx.Dice.DeckList {
								if i.Enable {
									for j, _ := range i.Command {
										lst = append(lst, j)
									}
								}
							}
							matches := fuzzy.FindFrom(text, lst)

							right := len(matches)
							if right > 10 {
								right = 3
							}

							text := "找到以下牌组:\n"
							for _, i := range matches[:right] {
								text += "- " + i.Str + "\n"
							}
							ReplyToSender(ctx, msg, text)
						} else {
							ReplyToSender(ctx, msg, "请给出要搜索的关键字")
						}
					} else {
						isDrew := false
						for _, i := range d.DeckList {
							if i.Enable {
								deckExists := i.Command[deckName]
								if deckExists {
									deck := i.DeckItems[deckName]
									result, _ := executeDeck(ctx, i, deck)
									ReplyToSender(ctx, msg, result)
									isDrew = true
								}
							}
						}
						if !isDrew {
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "其它:抽牌_找不到牌组"))
						}
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	theExt := &ExtInfo{
		Name:       "deck", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Author:     "木落",
		Brief:      "牌堆扩展，提供.deck指令支持，兼容Dice!和塔系牌堆",
		AutoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		OnLoad: func() {
			DeckReload(d)
		},
		GetDescText: func(i *ExtInfo) string {
			return GetExtensionDesc(i)
		},
		CmdMap: CmdMapCls{
			"draw": cmdDraw,
			"deck": cmdDraw,
		},
	}

	d.RegisterExtension(theExt)
}

func deckStringFormat(ctx *MsgContext, deckInfo *DeckInfo, s string) string {
	//  ***n***
	// 参考: https://sinanya.com/#/MakeFile
	// 1. 提取 {}
	re := regexp.MustCompile(`{[$%]?.+?}`)
	s = strings.ReplaceAll(s, "#{SPLIT}", "###SPLIT###")
	s = strings.ReplaceAll(s, "{FormFeed}", "###SPLIT###")
	s = strings.ReplaceAll(s, "{formfeed}", "###SPLIT###")
	m := re.FindAllStringIndex(s, -1)

	for _i := len(m) - 1; _i >= 0; _i-- {
		i := m[_i]

		var text string
		var err error

		deckName := s[i[0]:i[1]]

		// 特殊变量检查
		if deckName == "{player}" {
			var text string
			if ctx.Player != nil {
				text = ctx.Player.Name
			} else {
				text = "%未知用户%"
			}
			s = s[:i[0]] + text + s[i[1]:]
			continue
		}
		if deckName == "{self}" {
			var text string
			text = DiceFormatTmpl(ctx, "核心:骰子名字")
			s = s[:i[0]] + text + s[i[1]:]
			continue
		}

		sign := deckName[1]
		signLength := 0
		hasSign := sign == '$' || sign == '%'
		if hasSign {
			signLength += 1
		}

		deckName = deckName[1+signLength : len(deckName)-1]

		deck := deckInfo.DeckItems[deckName]
		if deck == nil {
			text = "<%未知牌组-" + deckName + "%>"
		} else {
			text, err = executeDeck(ctx, deckInfo, deck)
			if err != nil {
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
		re := regexp.MustCompile(`\[(img|图|文本|text|语音|voice):(.+?)]`) // [img:] 或 [图:]
		m := re.FindStringSubmatch(text)
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

		text = "{" + text[1:len(text)-1] + "}"
		s = s[:i[0]] + text + s[i[1]:]
	}

	s = CQRewrite(s, cqSolve)
	s = ImageRewrite(s, imgSolve)

	s = strings.ReplaceAll(s, "\n", `\n`)
	return DiceFormat(ctx, s)
}

func executeDeck(ctx *MsgContext, deckInfo *DeckInfo, deckGroup []string) (string, error) {
	pool := DeckToRandomPool(deckGroup)
	cmd := deckStringFormat(ctx, deckInfo, pool.Pick().(string))
	return cmd, nil
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
