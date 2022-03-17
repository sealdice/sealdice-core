package dice

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/sahilm/fuzzy"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// 分词器封存了，看起来不太需要
//_ "github.com/leopku/bleve-gse-tokenizer/v2"

type HelpTextItem struct {
	Title       string
	Content     string
	PackageName string
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
		m.EngineType = 1
	} else {
		m.EngineType = 0 // 默认，bleve
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
		Title:       "help",
		Content:     "帮助指令，但是还完全没有写任何内容。这仅仅是一条测试。",
		PackageName: "核心指令",
	})

	m.AddItem(HelpTextItem{
		Title:       "查询/find",
		Content:     "想要问什么呢？\n.查询 <数字ID> // 显示该ID的词条\n.查询 <任意文本> // 查询关联内容\n.查询 --rand // 随机词条",
		PackageName: "核心指令",
	})

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
		return m.Index.Index(id, data)
	} else {
		return nil
	}
}

func (m *HelpManager) searchBleve(ctx *MsgContext, text string, titleOnly bool) (*bleve.SearchResult, error) {
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

	return res, err
	//index.Close()
}

func (m *HelpManager) Search(ctx *MsgContext, text string, titleOnly bool) (*bleve.SearchResult, error) {
	if m.EngineType == 0 {
		return m.searchBleve(ctx, text, titleOnly)
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
		if right > 10 {
			right = 10
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

func (m *HelpManager) GetShowBestOffset() int {
	switch m.EngineType {
	case 0:
		return 1
	default:
		return 15
	}
}

type DiceManager struct {
	Dice                []*Dice
	ServeAddress        string
	Help                *HelpManager
	UseDictForTokenizer bool
}

type DiceConfigs struct {
	DiceConfigs  []DiceConfig `yaml:"diceConfigs"`
	ServeAddress string       `yaml:"serveAddress"`
	WebUIAddress string       `yaml:"webUIAddress"`
}

func (dm *DiceManager) InitHelp() {
	os.MkdirAll("./data/helpdoc", 0644)
	dm.Help = new(HelpManager)
	dm.Help.Parent = dm
	dm.Help.Load()
}

func (dm *DiceManager) LoadDice() {
	os.MkdirAll("./data/images", 0644)
	os.MkdirAll("./data/decks", 0644)
	ioutil.WriteFile("./data/images/sealdice.png", ICON_PNG, 0644)

	data, err := ioutil.ReadFile("./data/dice.yaml")
	if err != nil {
		return
	}

	var dc DiceConfigs
	err = yaml.Unmarshal(data, &dc)
	if err != nil {
		fmt.Println("读取 data/dice.yaml 发生错误: 配置文件格式不正确")
		panic(err)
	}

	dm.ServeAddress = dc.ServeAddress
	for _, i := range dc.DiceConfigs {
		newDice := new(Dice)
		newDice.BaseConfig = i
		newDice.Parent = dm
		dm.Dice = append(dm.Dice, newDice)
	}
}

func (dm *DiceManager) Save() {
	var dc DiceConfigs
	dc.ServeAddress = dm.ServeAddress
	for _, i := range dm.Dice {
		dc.DiceConfigs = append(dc.DiceConfigs, i.BaseConfig)
	}

	data, err := yaml.Marshal(dc)
	if err == nil {
		ioutil.WriteFile("./data/dice.yaml", data, 0644)
	}
}

func (dm *DiceManager) InitDice() {
	for _, i := range dm.Dice {
		i.Init()
	}
}

func (dm *DiceManager) TryCreateDefault() {
	if dm.ServeAddress == "" {
		dm.ServeAddress = "0.0.0.0:3211"
	}

	if len(dm.Dice) == 0 {
		defaultDice := new(Dice)
		defaultDice.BaseConfig.Name = "default"
		defaultDice.BaseConfig.IsLogPrint = true
		dm.Dice = append(dm.Dice, defaultDice)
	}
}
