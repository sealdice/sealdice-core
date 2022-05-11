package dice

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/sahilm/fuzzy"
	"github.com/xuri/excelize/v2"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

// 分词器封存了，看起来不太需要
//_ "github.com/leopku/bleve-gse-tokenizer/v2"

type HelpTextItem struct {
	Title       string
	Content     string
	PackageName string
	KeyWords    string
	RelatedExt  []string
}

type HelpTextItems []*HelpTextItem

func (e HelpTextItems) String(i int) string {
	return e[i].Title
}

func (e HelpTextItems) Len() int {
	return len(e)
}

type HelpManager struct {
	CurId      uint64
	Index      bleve.Index
	TextMap    map[string]*HelpTextItem
	Parent     *DiceManager
	EngineType int
	batch      *bleve.Batch
	batchNum   int
}

func (m *HelpManager) GetNextId() string {
	m.CurId += 1
	return strconv.FormatUint(m.CurId, 10)
}

type HelpDocFormat struct {
	Mod     string            `json:"mod"`
	Author  string            `json:"author"`
	Brief   string            `json:"brief"`
	Comment string            `json:"comment"`
	Helpdoc map[string]string `json:"helpdoc"`
}

func (m *HelpManager) loadSearchEngine() {
	if runtime.GOARCH == "arm64" {
		m.EngineType = 1 // 默认0，bleve
	}

	m.TextMap = map[string]*HelpTextItem{}

	switch m.EngineType {
	case 0: // 默认，bleve
		INDEX_DIR := "./data/_index"

		mapping := bleve.NewIndexMapping()
		os.RemoveAll(INDEX_DIR)

		if m.Parent.UseDictForTokenizer {
			//这些代码封存，看起来不怎么需要
			//if err := mapping.AddCustomTokenizer("gse", map[string]interface{}{
			//	"type":       "gse",
			//	"user_dicts": "./data/dict/zh/dict.txt", // <-- MUST specified, otherwise panic would occurred.
			//}); err != nil {
			//	panic(err)
			//}
			//if err := mapping.AddCustomAnalyzer("gse", map[string]interface{}{
			//	"type":      "gse",
			//	"tokenizer": "gse",
			//}); err != nil {
			//	panic(err)
			//}
			//mapping.DefaultAnalyzer = "gse"
		}

		docMapping := bleve.NewDocumentMapping()
		docMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
		docMapping.AddFieldMappingsAt("content", bleve.NewTextFieldMapping())
		docMapping.AddFieldMappingsAt("package", bleve.NewTextFieldMapping())

		mapping.AddDocumentMapping("helpdoc", docMapping)
		mapping.TypeField = "_type" // 此为默认值，可修改

		index, err := bleve.New(INDEX_DIR, mapping)
		if err != nil {
			panic(err)
		}

		m.Index = index
	}
}

func (m *HelpManager) Close() {
	if m.EngineType == 0 {
		m.Index.Close()
	}
}

func (m *HelpManager) Load() {
	m.loadSearchEngine()

	m.AddItem(HelpTextItem{
		Title:       "First Text",
		Content:     "In view, a humble vaudevillian veteran cast vicariously as both victim and villain vicissitudes of fate.",
		PackageName: "测试",
	})

	m.AddItem(HelpTextItem{
		Title:       "测试词条",
		Content:     "他在命运的沉浮中随波逐流, 扮演着受害与加害者的双重角色",
		PackageName: "测试",
	})

	m.AddItem(HelpTextItem{
		Title: "骰点",
		Content: `.help 骰点：
 .r  //丢一个100面骰
.r d10 //丢一个10面骰(数字可改)
.r 3d6 //丢3个6面骰(数字可改)
.ra 侦查 //侦查技能检定
.ra 侦查+10 //技能临时加值检定
.ra 3#p 射击 // 连续射击三次`,
		PackageName: "帮助",
	})

	m.AddItem(HelpTextItem{
		Title: "娱乐",
		Content: `.gugu // 随机召唤一只鸽子
.jrrp 今日人品
`,
		PackageName: "帮助",
	})

	m.AddItem(HelpTextItem{
		Title: "扩展",
		Content: `.help 扩展：
扩展功能可以让你开关部分指令。
例如你希望你的骰子是纯TRPG骰，那么可以通过.ext xxx off关闭一系列娱乐模块。
或者目前正在进行dnd5e游戏，你可以通过如下指令开关dnd特化扩展。COC亦然。
注意一点，不同扩展允许存在同名指令，例如dnd和coc都有st和rc，但他们本质上不是同一个指令，并不通用，还请注意。

.ext coc7 on // 打开coc7版扩展
.ext dnd5e off // 关闭dnd5版扩展

.ext dnd5e on // 打开dnd5版扩展
.ext coc7 off // 关闭coc7版扩展
`,
		PackageName: "帮助",
	})

	m.AddItem(HelpTextItem{
		Title: "日志",
		Content: `.help 日志：
.log new //新建记录
.log on //开始记录
.log off //暂停纪录
.log end //结束记录并导出
`,
		PackageName: "帮助",
	})

	m.AddItem(HelpTextItem{
		Title: "跑团",
		Content: `.help 跑团：
.st 力量50 //载入技能/属性
.coc // coc7版人物做成
.dnd // dnd5版任务做成
.ch save <角色名> //保存角色，无角色名则为当前
.ch load <角色名> //加载角色，无角色名则为当前
.ch list //列出当前角色
.ch del <角色名> //删除角色
.setcoc 2 //设置为coc2版房规
.nn 张三 //将自己的角色名设置为张三
`,
		PackageName: "帮助",
	})

	m.AddItem(HelpTextItem{
		Title: "骰主",
		Content: `.botlist add @A @B @C // 标记群内其他机器人，以免发生误触和无限对话
.botlist del @A @B @C // 去除机器人标记
.botlist list // 查看当前列表
.master add me @A @B // 标记骰主
.send <留言> // 给骰主留言
`,
		PackageName: "帮助",
	})

	m.AddItem(HelpTextItem{
		Title: "其他",
		Content: `.find 克苏鲁星之眷族 //查找对应怪物资料
.find 70尺 法术 // 查找关联资料（仅在全文搜索开启时可用）
`,
		PackageName: "帮助",
	})

	//m.AddItem(HelpTextItem{
	//	Title:       "查询/find",
	//	Content:     "想要问什么呢？\n.查询 <数字ID> // 显示该ID的词条\n.查询 <任意文本> // 查询关联内容\n.查询 --rand // 随机词条",
	//	PackageName: "核心指令",
	//})

	filepath.WalkDir("data/helpdoc", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			fileExt := filepath.Ext(path)

			switch fileExt {
			case ".json":
				data := HelpDocFormat{}
				pack, err := ioutil.ReadFile(path)
				if err == nil {
					err = json.Unmarshal(pack, &data)
					if err == nil {
						for k, v := range data.Helpdoc {
							m.AddItem(HelpTextItem{
								Title:       k,
								Content:     v,
								PackageName: data.Mod,
							})
						}
					}
				}
			case ".xlsx":
				// 梨骰帮助文件
				f, err := excelize.OpenFile(path)
				if err != nil {
					fmt.Println(err)
					break
				}
				defer func() {
					// Close the spreadsheet.
					if err := f.Close(); err != nil {
						fmt.Println(err)
					}
				}()

				for _, s := range f.GetSheetList() {
					rows, err := f.GetRows(s)
					if err == nil {
						for _, row := range rows {
							//Key Synonym Content Description Catalogue Tag
							key := row[0]
							synonym := row[1]
							content := row[2]

							if synonym != "" {
								key += "/" + synonym
							}

							m.AddItem(HelpTextItem{
								Title:       key,
								Content:     content,
								PackageName: s,
							})
						}
					}
				}
			}
		}
		return nil
	})
	m.AddItemApply()
}

func (dm *DiceManager) AddHelpWithDice(dice *Dice) {
	//lst := map[*CmdItemInfo]bool{}
	//for _, v := range dice.CmdMap {
	//	lst[v] = true
	//}
	m := dm.Help

	addCmdMap := func(packageName string, cmdMap CmdMapCls) {
		for k, v := range cmdMap {
			content := v.LongHelp
			if content == "" {
				content = v.Help
			}
			m.AddItem(HelpTextItem{
				Title:       k,
				Content:     content,
				PackageName: packageName,
			})
		}
	}

	addCmdMap("核心指令", dice.CmdMap)
	for _, i := range dice.ExtList {
		m.AddItem(HelpTextItem{
			Title:       i.Name,
			Content:     i.GetDescText(i),
			PackageName: "扩展模块",
		})
		addCmdMap(i.Name, i.CmdMap)
	}
	m.AddItemApply()
}

func (m *HelpManager) AddItem(item HelpTextItem) error {
	data := map[string]string{
		"title":   item.Title,
		"content": item.Content,
		"package": item.PackageName,
		"_type":   "helpdoc",
	}

	id := m.GetNextId()
	m.TextMap[id] = &item

	if m.EngineType == 0 {
		if m.batch == nil {
			m.batch = m.Index.NewBatch()
		}
		if m.batchNum >= 50 {
			m.Index.Batch(m.batch)
			m.batch.Reset()
			m.batchNum = 0
		}

		m.batchNum += 1
		return m.batch.Index(id, data)
		//return m.Index.Index(id, data)
	} else {
		return nil
	}
}

func (m *HelpManager) AddItemApply() error {
	if m.batch != nil {
		err := m.Index.Batch(m.batch)
		m.batch.Reset()
		m.batch = nil
		return err
	}
	return nil
}

func (m *HelpManager) searchBleve(ctx *MsgContext, text string, titleOnly bool, num int) (*bleve.SearchResult, error) {
	// 在标题中查找
	queryTitle := query.NewMatchPhraseQuery(text)
	queryTitle.SetField("title")

	titleOrContent := bleve.NewDisjunctionQuery(queryTitle)

	// 在正文中查找
	if !titleOnly {
		for _, i := range reSpace.Split(text, -1) {
			queryContent := query.NewMatchPhraseQuery(i)
			queryContent.SetField("content")
			titleOrContent.AddQuery(queryContent)
		}
	}

	//queryContent := query.NewQueryStringQuery("-AAAAAAAA +vaudevillian")
	////queryContent.SetField("content")
	andQuery := bleve.NewConjunctionQuery(titleOrContent)

	// 限制查询组
	for _, i := range ctx.Group.HelpPackages {
		queryPack := query.NewMatchPhraseQuery(i)
		queryPack.SetField("package")
		andQuery.AddQuery(queryPack)
	}

	req := bleve.NewSearchRequest(andQuery)

	index := m.Index
	res, err := index.Search(req)

	if err == nil {
		res.Hits = res.Hits[:num]
	}

	return res, err
	//index.Close()
}

func (m *HelpManager) Search(ctx *MsgContext, text string, titleOnly bool, num int) (*bleve.SearchResult, error) {
	if num < 1 {
		num = 1
	}
	if num > 10 {
		num = 10
	}

	if m.EngineType == 0 {
		return m.searchBleve(ctx, text, titleOnly, num)
	} else {
		//for _, i := range ctx.Group.HelpPackages {
		//	//queryPack := query.NewMatchPhraseQuery(i)
		//	//queryPack.SetField("package")
		//}

		// 不是很好的做法，待优化
		items := HelpTextItems{}
		idLst := []string{}

		for id, v := range m.TextMap {
			items = append(items, v)
			idLst = append(idLst, id)
		}

		hits := search.DocumentMatchCollection{}

		matches := fuzzy.FindFrom(text, items)

		right := len(matches)

		if right > num {
			right = num
		}
		for _, i := range matches[:right] {
			hits = append(hits, &search.DocumentMatch{
				ID:    idLst[i.Index],
				Score: float64(i.Score),
			})
		}

		return &bleve.SearchResult{
			Status:  nil,
			Request: nil,
			Hits:    hits,
			Total:   uint64(len(hits)),
		}, nil
	}
}

func (m *HelpManager) GetSuffixText() string {
	switch m.EngineType {
	case 0:
		return "(本次搜索由全文搜索完成)"
	default:
		return "(本次搜索由快速文档查找完成)"
	}
}

func (m *HelpManager) GetSuffixText2() string {
	switch m.EngineType {
	case 0:
		return "[全文搜索]"
	default:
		return "[快速文档查找]"
	}
}

func (m *HelpManager) GetShowBestOffset() int {
	switch m.EngineType {
	case 0:
		return 1
	default:
		return 15
	}
}

func (m *HelpManager) GetContent(item *HelpTextItem, depth int) string {
	if depth > 7 {
		return "{递归层数过多，不予显示}"
	}
	txt := item.Content
	finalTxt := ""
	re := regexp.MustCompile(`\{[^}\n]+\}`)
	matched := re.FindAllStringSubmatchIndex(txt, -1)

	for _, i := range matched {
		left := i[0]
		right := i[1]

		skip := false
		if left != 0 {
			if txt[left-1] == '\\' {
				skip = true
			}
		}

		if !skip {
			finalTxt += txt[:left]
			name := txt[left+1 : right-1]
			matched := false
			// 注意: 效率不高
			for _, v := range m.TextMap {
				if v.Title == name {
					finalTxt += m.GetContent(v, depth+1)
					matched = true
					break
				}
			}
			if !matched {
				finalTxt += txt[left:right-1] + " - 未能找到" + "}"
			}
			finalTxt += txt[right:]
		}
	}

	if len(matched) == 0 {
		return txt
	}
	return finalTxt
}
