package dice

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	nanoid "github.com/matoous/go-nanoid/v2"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/sahilm/fuzzy"
	"github.com/xuri/excelize/v2"
)

// 分词器封存了，看起来不太需要
// _ "github.com/leopku/bleve-gse-tokenizer/v2"

const HelpBuiltinGroup = "builtin"

const (
	Unload int = iota
	Loaded
	LoadError
)

type HelpDoc struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Group      string `json:"group"`
	Type       string `json:"type"`
	IsDir      bool   `json:"isDir"`
	LoadStatus int    `json:"loadStatus"`
	Deleted    bool   `json:"deleted"`

	Children []*HelpDoc `json:"children"`
}

type HelpTextItem struct {
	Group       string
	From        string
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
	CurID       uint64
	Index       bleve.Index
	TextMap     map[string]*HelpTextItem
	Parent      *DiceManager
	EngineType  int
	batch       *bleve.Batch
	batchNum    int
	LoadingFn   string
	HelpDocTree []*HelpDoc
}

func (m *HelpManager) GetNextID() string {
	m.CurID++
	return strconv.FormatUint(m.CurID, 10)
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

	// not bleve
	if m.EngineType != 0 {
		return
	}

	// 删除旧版本的
	indexDir := "./data/_index"
	_ = os.RemoveAll(indexDir)

	mapping := bleve.NewIndexMapping()
	indexDir = "./_help_cache"
	_ = os.RemoveAll(indexDir)

	// if m.Parent.UseDictForTokenizer {
	// 这些代码封存，看起来不怎么需要
	// if err := mapping.AddCustomTokenizer("gse", map[string]interface{}{
	// 	"type":       "gse",
	// 	"user_dicts": "./data/dict/zh/dict.txt", // <-- MUST specified, otherwise panic would occurred.
	// }); err != nil {
	// 	panic(err)
	// }
	// if err := mapping.AddCustomAnalyzer("gse", map[string]interface{}{
	// 	"type":      "gse",
	// 	"tokenizer": "gse",
	// }); err != nil {
	// 	panic(err)
	// }
	// mapping.DefaultAnalyzer = "gse"
	// }

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	docMapping.AddFieldMappingsAt("content", bleve.NewTextFieldMapping())
	docMapping.AddFieldMappingsAt("package", bleve.NewTextFieldMapping())

	mapping.AddDocumentMapping("helpdoc", docMapping)
	mapping.TypeField = "_type" // 此为默认值，可修改

	index, err := bleve.New(indexDir, mapping)
	if err != nil {
		panic(err)
	}

	m.Index = index
}

func (m *HelpManager) Close() {
	if m.EngineType == 0 {
		if m.Index != nil {
			_ = m.Index.Close()
			m.Index = nil

			_ = os.RemoveAll("./_help_cache")
		}
	}
}

func (m *HelpManager) Load() {
	m.loadSearchEngine()

	_ = m.AddItem(HelpTextItem{
		Group: HelpBuiltinGroup,
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

	_ = m.AddItem(HelpTextItem{
		Group: HelpBuiltinGroup,
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

	_ = m.AddItem(HelpTextItem{
		Group: HelpBuiltinGroup,
		Title: "跑团",
		Content: `.help 跑团：
.st 力量50 //载入技能/属性
.coc // coc7版人物做成
.dnd // dnd5版任务做成
.pc new <角色名> // 创建角色并自动绑卡，无角色名则为当前
.pc tag <角色名> // 当前群绑卡/解除绑卡(不填角色名)
.pc save <角色名> // 保存角色[不绑卡时需要手动保存]，无角色名则为当前
.pc load <角色名> // 加载角色[不绑卡]，无角色名则为当前
.pc list //列出当前角色
.pc del <角色名> //删除角色
.setcoc 2 //设置为coc2版房规
.nn 张三 //将自己的角色名设置为张三
`,
		PackageName: "帮助",
	})

	m.HelpDocTree = make([]*HelpDoc, 0)
	entries, err := os.ReadDir("data/helpdoc")
	if err != nil {
		fmt.Println("unable to read helpdoc folder: ", err.Error())
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		var child HelpDoc
		child.Key = generateHelpDocKey()
		child.Name = entry.Name()
		child.Path = path.Join("data/helpdoc", entry.Name())
		child.IsDir = entry.IsDir()
		if child.IsDir {
			child.Group = entry.Name()
			child.Type = "dir"
			child.Children = make([]*HelpDoc, 0)
		} else {
			child.Group = "default"
			child.Type = filepath.Ext(child.Path)
		}
		buildHelpDocTree(&child, func(d *HelpDoc) {
			if !d.IsDir {
				ok := m.loadHelpDoc(d.Group, d.Path)
				if ok {
					d.LoadStatus = Loaded
				} else {
					d.LoadStatus = LoadError
				}
			}
		})
		m.HelpDocTree = append(m.HelpDocTree, &child)
	}
	_ = m.AddItemApply()
}

func (m *HelpManager) loadHelpDoc(group string, path string) bool {
	fileExt := filepath.Ext(path)

	switch fileExt {
	case ".json":
		m.LoadingFn = path
		data := HelpDocFormat{}
		pack, err := os.ReadFile(path)
		if err == nil {
			err = json.Unmarshal(pack, &data)
			if err == nil {
				for k, v := range data.Helpdoc {
					_ = m.AddItem(HelpTextItem{
						Group:       group,
						From:        path,
						Title:       k,
						Content:     v,
						PackageName: data.Mod,
					})
				}
			}
		}
		return true
	case ".xlsx":
		// 梨骰帮助文件
		m.LoadingFn = path
		f, err := excelize.OpenFile(path)
		if err != nil {
			fmt.Println(err)
			break
		}

		for index, s := range f.GetSheetList() {
			rows, err := f.GetRows(s)
			if err == nil {
				for i, row := range rows {
					if i == 0 {
						err := validateXlsxHeaders(row)
						if err == nil {
							// 跳过第一行
							continue
						} else {
							fmt.Printf("%s sheet %d: %s\n", path, index, err)
							break
						}
					}
					if len(row) < 3 {
						continue
					}
					key := row[0]
					synonym := row[1]
					content := row[2]

					if synonym != "" {
						key += "/" + synonym
					}

					_ = m.AddItem(HelpTextItem{
						Group:       group,
						From:        path,
						Title:       key,
						Content:     content,
						PackageName: s,
					})
				}
			}
		}

		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
		return true
	}
	return false
}

// validateXlsxHeaders 验证 xlsx 格式 helpdoc 的表头是否是 Key Synonym Content Description Catalogue Tag
func validateXlsxHeaders(headers []string) error {
	if len(headers) < 3 {
		return fmt.Errorf("helpdoc格式错误，缺少必须列")
	}
	if headers[0] != "Key" {
		return fmt.Errorf("helpdoc表头格式错误，第一列表头必须是Key")
	}
	if headers[1] != "Synonym" {
		return fmt.Errorf("helpdoc表头格式错误，第二列表头必须是Synonym")
	}
	if headers[2] != "Content" {
		return fmt.Errorf("helpdoc表头格式错误，第三列表头必须是Content")
	}

	// 其它表头校验
	if len(headers) > 3 && headers[3] != "Description" {
		return fmt.Errorf("helpdoc表头格式错误，第四列表头必须是Description")
	}
	if len(headers) > 4 && headers[4] != "Catalogue" {
		return fmt.Errorf("helpdoc表头格式错误，第五列表头必须是Catalogue")
	}
	if len(headers) > 5 && headers[6] != "Tag" {
		return fmt.Errorf("helpdoc表头格式错误，第六列表头必须是Tag")
	}
	return nil
}

func (dm *DiceManager) AddHelpWithDice(dice *Dice) {
	m := dm.Help

	addCmdMap := func(packageName string, cmdMap CmdMapCls) {
		for k, v := range cmdMap {
			content := v.Help
			if content == "" {
				content = v.ShortHelp
			}
			_ = m.AddItem(HelpTextItem{
				Group:       HelpBuiltinGroup,
				Title:       k,
				Content:     content,
				PackageName: packageName,
			})
		}
	}

	addCmdMap("核心指令", dice.CmdMap)
	for _, i := range dice.ExtList {
		_ = m.AddItem(HelpTextItem{
			Group:       HelpBuiltinGroup,
			Title:       i.Name,
			Content:     i.GetDescText(i),
			PackageName: "扩展模块",
		})
		addCmdMap(i.Name, i.CmdMap)
	}
	_ = m.AddItemApply()
}

func (m *HelpManager) AddItem(item HelpTextItem) error {
	data := map[string]string{
		"group":   item.Group,
		"from":    item.From,
		"title":   item.Title,
		"content": item.Content,
		"package": item.PackageName,
		"_type":   "helpdoc",
	}

	id := m.GetNextID()
	m.TextMap[id] = &item

	if m.EngineType == 0 {
		if m.batch == nil {
			m.batch = m.Index.NewBatch()
		}
		if m.batchNum >= 50 {
			err := m.Index.Batch(m.batch)
			if err != nil {
				return err
			}
			m.batch.Reset()
			m.batchNum = 0
		}

		m.batchNum++
		return m.batch.Index(id, data)
	}
	return nil
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

func (m *HelpManager) searchBleve(ctx *MsgContext, text string, titleOnly bool, pageSize, pageNum int, group string) (*bleve.SearchResult, int, int, int, error) {
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

	andQuery := bleve.NewConjunctionQuery(titleOrContent)

	// 限制查询组
	for _, i := range ctx.Group.HelpPackages {
		queryPack := query.NewMatchPhraseQuery(i)
		queryPack.SetField("package")
		andQuery.AddQuery(queryPack)
	}

	// 查询指定文档组
	if group != "" {
		queryPack := query.NewMatchPhraseQuery(group)
		queryPack.SetField("group")
		andQuery.AddQuery(queryPack)
	}

	req := bleve.NewSearchRequestOptions(andQuery, pageSize, (pageNum-1)*pageSize, false)

	index := m.Index
	res, err := index.Search(req)
	if err != nil {
		return res, 0, 0, 0, err
	}

	total := int(res.Total)
	pageStart := (pageNum - 1) * pageSize
	pageEnd := pageStart + len(res.Hits)
	return res, total, pageStart, pageEnd, nil
}

func (m *HelpManager) Search(ctx *MsgContext, text string, titleOnly bool, pageSize, pageNum int, group string) (res *bleve.SearchResult, total, pageStart, pageEnd int, err error) {
	if pageSize <= 0 || pageNum <= 0 {
		// 为了使Search的结果完全忠实于分页参数, 而不产生有结果但与分页不相符的情况
		return nil, 0, 0, 0, fmt.Errorf("分页参数错误")
	}

	if m.EngineType == 0 {
		return m.searchBleve(ctx, text, titleOnly, pageSize, pageNum, group)
	}

	// 不是很好的做法，待优化
	items := HelpTextItems{}
	var idLst []string

	for id, v := range m.TextMap {
		items = append(items, v)
		idLst = append(idLst, id)
	}

	hits := search.DocumentMatchCollection{}
	matches := fuzzy.FindFrom(text, items)

	total = len(matches)
	pageStart = (pageNum - 1) * pageSize
	pageEnd = pageNum * pageSize

	if pageStart < total {
		if pageEnd > total {
			pageEnd = total
		}

		for _, i := range matches[pageStart:pageEnd] {
			hits = append(hits, &search.DocumentMatch{
				ID:    idLst[i.Index],
				Score: float64(i.Score),
			})
		}
	} else {
		// 分页超出范围, 返回空结果
		pageStart = -1
		pageEnd = -1
	}

	return &bleve.SearchResult{
		Status:  nil,
		Request: nil,
		Hits:    hits,
		Total:   uint64(total),
	}, total, pageStart, pageEnd, nil
}

func (m *HelpManager) GetSuffixText() string {
	switch m.EngineType {
	case 0:
		return "(本次搜索由全文搜索完成)"
	default:
		return "(本次搜索由快速文档查找完成)"
	}
}

func (m *HelpManager) GetPrefixText() string {
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

func generateHelpDocKey() string {
	key, _ := nanoid.Generate("0123456789abcdef", 16)
	return key
}

func buildHelpDocTree(node *HelpDoc, fn func(d *HelpDoc)) {
	p, err := os.Stat(node.Path)
	if err != nil {
		return
	}

	fn(node)

	if !p.IsDir() {
		return
	}

	subs, err := os.ReadDir(node.Path)
	if err != nil {
		return
	}

	for _, sub := range subs {
		if strings.HasPrefix(sub.Name(), ".") {
			continue
		}
		var child HelpDoc
		child.Key = generateHelpDocKey()
		child.Name = sub.Name()
		child.Path = path.Join(node.Path, sub.Name())
		child.Group = node.Group
		child.IsDir = sub.IsDir()
		if sub.IsDir() {
			child.Type = "dir"
			child.Children = make([]*HelpDoc, 0)
		} else {
			child.Type = filepath.Ext(sub.Name())
		}

		fn(&child)
		if sub.IsDir() {
			buildHelpDocTree(&child, fn)
		}
		node.Children = append(node.Children, &child)
	}
}

func (m *HelpManager) UploadHelpDoc(src io.Reader, group string, name string) error {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	group = strings.ReplaceAll(group, "/", "_")
	group = strings.ReplaceAll(group, "\\", "_")
	if group == "default" {
		// 默认组直接上传到helpdoc文件夹中
		group = ""
	}

	dirPath := filepath.Join("./data/helpdoc", group)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dirPath, name)
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	var groupExists bool
	for _, groupDir := range m.HelpDocTree {
		if groupDir.Name == group {
			groupExists = true
			groupDir.Deleted = false

			var fileExists bool
			for _, child := range groupDir.Children {
				if child.Name == name && filepath.Clean(child.Path) == filepath.Clean(filePath) && !child.Deleted {
					if child.LoadStatus == Unload {
						child.Deleted = false
						fileExists = true
					} else {
						child.Deleted = true
					}
				}
			}
			if !fileExists {
				groupDir.Children = append(groupDir.Children, &HelpDoc{
					Key:   generateHelpDocKey(),
					Name:  name,
					Path:  filePath,
					Group: group,
					Type:  filepath.Ext(filePath),
				})
			}
			break
		}
	}
	if !groupExists {
		if group != "" {
			newGroupDir := HelpDoc{
				Key:      generateHelpDocKey(),
				Name:     group,
				Path:     dirPath,
				Group:    group,
				Type:     "dir",
				IsDir:    true,
				Children: make([]*HelpDoc, 0),
			}
			newGroupDir.Children = append(newGroupDir.Children, &HelpDoc{
				Key:   generateHelpDocKey(),
				Name:  name,
				Path:  filePath,
				Group: group,
				Type:  filepath.Ext(filePath),
			})
			m.HelpDocTree = append(m.HelpDocTree, &newGroupDir)
		} else {
			m.HelpDocTree = append(m.HelpDocTree, &HelpDoc{
				Key:   generateHelpDocKey(),
				Name:  name,
				Path:  filePath,
				Group: "default",
				Type:  filepath.Ext(filePath),
			})
		}
	}

	return nil
}

func (m *HelpManager) DeleteHelpDoc(keys []string) error {
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	for _, node := range m.HelpDocTree {
		err := traverseHelpDocTree(node, func(d *HelpDoc) error {
			if !d.IsDir {
				_, ok := keySet[d.Key]
				if ok {
					_, err := os.Stat(d.Path)
					if !os.IsNotExist(err) {
						err := os.Remove(d.Path)
						if err != nil {
							return err
						}
					}
					d.Deleted = true
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		_, ok := keySet[node.Key]
		if ok {
			_, err := os.Stat(node.Path)
			if !os.IsNotExist(err) {
				err := os.RemoveAll(node.Path)
				if err != nil {
					return err
				}
			}
			node.Deleted = true
		}
	}
	return nil
}

func traverseHelpDocTree(root *HelpDoc, fn func(node *HelpDoc) error) error {
	if root == nil {
		return nil
	}
	err := fn(root)
	if err != nil {
		return err
	}

	if len(root.Children) == 0 {
		return nil
	}

	for _, child := range root.Children {
		err := traverseHelpDocTree(child, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

type HelpTextVo struct {
	ID          int    `json:"id"`
	Group       string `json:"group"`
	From        string `json:"from"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	PackageName string `json:"packageName"`
	KeyWords    string `json:"keyWords"`
}

type HelpTextVos []HelpTextVo

func (h HelpTextVos) Len() int {
	return len(h)
}

func (h HelpTextVos) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h HelpTextVos) Less(i, j int) bool {
	return h[i].ID < h[j].ID
}

func (m *HelpManager) GetHelpItemPage(pageNum, pageSize int, id, group, from, title string) (int, HelpTextVos) {
	if pageNum <= 0 || pageSize <= 0 {
		return 0, HelpTextVos{}
	}

	if id != "" {
		item := m.TextMap[id]
		if item != nil &&
			strings.Contains(item.Group, group) &&
			strings.Contains(item.From, from) &&
			strings.Contains(item.Title, title) {
			vo := HelpTextVo{
				Group:       item.Group,
				From:        item.From,
				Title:       item.Title,
				Content:     item.Content,
				PackageName: item.PackageName,
				KeyWords:    item.KeyWords,
			}
			vo.ID, _ = strconv.Atoi(id)
			return 1, HelpTextVos{vo}
		}
		return 0, HelpTextVos{}
	}
	temp := make(HelpTextVos, 0, len(m.TextMap))
	for i, item := range m.TextMap {
		if strings.Contains(item.Group, group) &&
			strings.Contains(item.From, from) &&
			strings.Contains(item.Title, title) {
			vo := HelpTextVo{
				Group:       item.Group,
				From:        item.From,
				Title:       item.Title,
				Content:     item.Content,
				PackageName: item.PackageName,
				KeyWords:    item.KeyWords,
			}
			vo.ID, _ = strconv.Atoi(i)
			temp = append(temp, vo)
		}
	}
	sort.Sort(temp)

	start := (pageNum - 1) * pageSize
	end := start + pageSize
	total := len(temp)
	if start >= total {
		return total, HelpTextVos{}
	} else if end < total {
		return total, temp[start:end]
	}
	return total, temp[start:]
}
