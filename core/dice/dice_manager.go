package dice

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	_ "github.com/leopku/bleve-gse-tokenizer/v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strconv"
)

type HelpTextItem struct {
	Title       string
	Content     string
	PackageName string
}

type HelpManager struct {
	CurId   uint64
	Index   bleve.Index
	TextMap map[string]*HelpTextItem
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

func (m *HelpManager) Load() {
	INDEX_DIR := "./data/_index"
	m.TextMap = map[string]*HelpTextItem{}

	mapping := bleve.NewIndexMapping()
	os.RemoveAll(INDEX_DIR)

	if err := mapping.AddCustomTokenizer("gse", map[string]interface{}{
		"type":       "gse",
		"user_dicts": "zh", // <-- MUST specified, otherwise panic would occurred.
	}); err != nil {
		panic(err)
	}
	if err := mapping.AddCustomAnalyzer("gse", map[string]interface{}{
		"type":      "gse",
		"tokenizer": "gse",
	}); err != nil {
		panic(err)
	}
	mapping.DefaultAnalyzer = "gse"

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	docMapping.AddFieldMappingsAt("content", bleve.NewTextFieldMapping())
	docMapping.AddFieldMappingsAt("package", bleve.NewTextFieldMapping())
	docMapping.DefaultAnalyzer = "gse"

	mapping.AddDocumentMapping("helpdoc", docMapping)
	mapping.TypeField = "_type" // 此为默认值，可修改

	index, err := bleve.New(INDEX_DIR, mapping)
	if err != nil {
		panic(err)
	}

	m.Index = index

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
		Title:       "查询/search",
		Content:     "想要问什么呢？\n.查询 <数字ID> // 显示该ID的词条\n.查询 <任意文本> // 查询关联内容\n.查询 --rand // 随机词条",
		PackageName: "核心指令",
	})

	pack, err := ioutil.ReadFile("data/蜜瓜包-怪物之锤查询.json")
	if err == nil {
		data := HelpDocFormat{}
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
	return m.Index.Index(id, data)
}

func (m *HelpManager) Search(ctx *MsgContext, text string, titleOnly bool) (*bleve.SearchResult, error) {
	// 在标题中查找
	queryTitle := query.NewMatchPhraseQuery(text)
	queryTitle.SetField("title")

	titleOrContent := bleve.NewDisjunctionQuery(queryTitle)

	// 在正文中查找
	if !titleOnly {
		queryContent := query.NewMatchPhraseQuery(text)
		queryTitle.SetField("title")
		titleOrContent.AddQuery(queryContent)
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

type DiceManager struct {
	Dice         []*Dice
	ServeAddress string
	Help         *HelpManager
}

type DiceConfigs struct {
	DiceConfigs  []DiceConfig `yaml:"diceConfigs"`
	ServeAddress string       `yaml:"serveAddress"`
	WebUIAddress string       `yaml:"webUIAddress"`
}

func (dm *DiceManager) InitHelp() {
	dm.Help = new(HelpManager)
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
