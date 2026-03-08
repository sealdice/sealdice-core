package docengine

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/simple"
	"github.com/blevesearch/bleve/v2/search/query"
	index "github.com/blevesearch/bleve_index_api"
	"github.com/oklog/ulid/v2"

	"sealdice-core/logger"
)

type BleveSearchEngine struct {
	Index     bleve.Index
	batch     *bleve.Batch
	batchSize int
	CurID     uint64
	// idList 按照字典序(ULID的特性)排序的文档ID列表，用于为用户提供“纯数字”的短ID
	idList []string
	// idToNumber 反向映射：ULID -> 数字ID(从1开始)，用于在搜索结果中展示短ID
	idToNumber map[string]int
}

var indexDir = "./data/_help_cache/_index"
var reSpace = regexp.MustCompile(`\s+`)

func (d *BleveSearchEngine) getNextID() string {
	return ulid.Make().String()
}

// NewEngine 创建并初始化 BleveSearchEngine
func NewBleveSearchEngine() (*BleveSearchEngine, error) {
	engine := &BleveSearchEngine{}
	err := engine.Init()
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func (d *BleveSearchEngine) GetSuffixText() string {
	return "(本次搜索由全文搜索完成)"
}

func (d *BleveSearchEngine) GetPrefixText() string {
	return "[全文搜索]"
}

func (d *BleveSearchEngine) GetShowBestOffset() int {
	return 1
}

func (d *BleveSearchEngine) Init() error {
	mapping := bleve.NewIndexMapping()
	docMapping := bleve.NewDocumentMapping()
	contentFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping := bleve.NewTextFieldMapping()
	keywordMapping := bleve.NewKeywordFieldMapping()
	// 试图：不区分大小写的搜索方案
	keywordMapping.Analyzer = simple.Name
	// 注意： 这里group,from,package都是keywordMapping
	// title既要做分词匹配，又要做精确匹配，需要特殊配置它
	// 下面这些GPT说的，如果不对，随便改。
	// 不需要分词，只需要支持模糊匹配（类似 SQL 中的 LIKE），那么 keyword 类型的字段 是最合适的选择。
	// keyword 类型的字段会将整个字段值作为一个整体存储，适合精确匹配和通配符匹配（如 NewWildcardQuery）。
	docMapping.AddFieldMappingsAt("group", keywordMapping)
	docMapping.AddFieldMappingsAt("from", keywordMapping)
	docMapping.AddFieldMappingsAt("title", titleFieldMapping)
	// Content才是真正的文档
	docMapping.AddFieldMappingsAt("content", contentFieldMapping)
	docMapping.AddFieldMappingsAt("package", keywordMapping)
	mapping.AddDocumentMapping("helpdoc", docMapping)
	mapping.TypeField = "_type"

	var i bleve.Index
	var err error

	i, err = bleve.Open(indexDir)
	if err != nil {
		i, err = bleve.New(indexDir, mapping)
		if err != nil {
			return err
		}
	}

	d.Index = i
	d.CurID = 0
	d.batch = d.Index.NewBatch()
	// 初始化数字ID映射（从已有索引构建）
	// 目的：为用户提供稳定的“纯数字且不太长”的ID
	_ = d.rebuildNumericIDMapping()
	return nil
}

func (d *BleveSearchEngine) Close() {
	if d.Index != nil {
		_ = d.Index.Close()
		d.Index = nil
	}
}

func (d *BleveSearchEngine) GetTotalID() uint64 {
	return d.CurID
}

// rebuildNumericIDMapping 重建 ULID -> 数字ID 的映射（内存态）
// 步骤：
// 1. 遍历全部文档ID
// 2. 基于ULID的字典序进行排序（ULID的字典序具备时间有序且唯一的属性）
// 3. 生成从1开始的数字ID映射
func (d *BleveSearchEngine) rebuildNumericIDMapping() error {
	if d.Index == nil {
		// 索引尚未就绪，直接返回
		return nil
	}
	req := bleve.NewSearchRequestOptions(bleve.NewMatchAllQuery(), 1000000, 0, false)
	req.Fields = []string{} // 不需要取字段，仅需要ID
	res, err := d.Index.Search(req)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(res.Hits))
	for _, hit := range res.Hits {
		ids = append(ids, hit.ID)
	}
	// 使用字典序进行排序：ULID的特性可保证唯一且时间有序
	sort.Strings(ids)
	d.idList = ids
	// 重建反查表
	d.idToNumber = make(map[string]int, len(ids))
	for i, id := range ids {
		// 数字ID从1开始，符合用户认知
		d.idToNumber[id] = i + 1
	}
	return nil
}

// getNumericIDByULID 将内部ULID转换为“纯数字ID（字符串）”
// 若当前映射不存在或未命中，尝试重建；仍未命中时返回ULID原值（极少数异常场景）
func (d *BleveSearchEngine) getNumericIDByULID(id string) string {
	if d.idToNumber == nil {
		_ = d.rebuildNumericIDMapping()
	}
	if num, ok := d.idToNumber[id]; ok {
		return strconv.Itoa(num)
	}
	// 可能发生于索引刚变化，内存映射尚未更新，尝试重建一次
	if err := d.rebuildNumericIDMapping(); err == nil {
		if num, ok := d.idToNumber[id]; ok {
			return strconv.Itoa(num)
		}
	}
	// 理论上不应走到这里；为保证功能可用，做降级回退
	return id
}

func (d *BleveSearchEngine) DeleteByFrom(path string) error {
	if d.Index == nil {
		return nil
	}
	q := bleve.NewMatchPhraseQuery(path)
	q.SetField("from")
	req := bleve.NewSearchRequestOptions(q, 10000, 0, false)
	req.Fields = []string{}
	res, err := d.Index.Search(req)
	if err != nil {
		return err
	}
	for _, hit := range res.Hits {
		if err := d.Index.Delete(hit.ID); err != nil {
			return err
		}
	}
	// 发生删除后重建映射，确保数字ID与当前索引一致
	_ = d.rebuildNumericIDMapping()
	return nil
}

func (d *BleveSearchEngine) DeleteByGroup(group string) error {
	if d.Index == nil {
		return nil
	}
	q := bleve.NewMatchPhraseQuery(group)
	q.SetField("group")
	req := bleve.NewSearchRequestOptions(q, 10000, 0, false)
	req.Fields = []string{}
	res, err := d.Index.Search(req)
	if err != nil {
		return err
	}
	for _, hit := range res.Hits {
		if err := d.Index.Delete(hit.ID); err != nil {
			return err
		}
	}
	// 发生删除后重建映射，确保数字ID与当前索引一致
	_ = d.rebuildNumericIDMapping()
	return nil
}

// AddItem 这里引用了dice，其实不妥，应该将它单独拆出来的。
func (d *BleveSearchEngine) AddItem(item HelpTextItem) (string, error) {
	// 如果batch为空，初始化一个batch
	if d.batch == nil {
		return "", errors.New("已通过end参数执行AddItemApply，不允许新增文档。请检查代码逻辑")
	}
	id := d.getNextID()
	data := map[string]string{
		"group":   item.Group,
		"from":    item.From,
		"title":   item.Title,
		"content": item.Content,
		"package": item.PackageName,
		"_type":   "helpdoc",
	}
	d.batchSize++
	// 五十一次执行
	if d.batchSize >= 50 {
		err := d.AddItemApply(false)
		d.batchSize = 0
		if err != nil {
			return "", err
		}
	}
	return id, d.batch.Index(id, data)
}

// AddItemApply 这里认为是真正执行插入文档的逻辑
// 由于现在已经将执行函数改为了可按文件执行，所以可以按文件进行Apply，这应当不会有太大的量级。
// end代表是否是最后一次执行，一般用在所有的数据都处理完之后，关闭逻辑的时候使用，如bleve batch重复利用后最后销毁
func (d *BleveSearchEngine) AddItemApply(end bool) error {
	if d.batch != nil {
		// 执行batch
		err := d.Index.Batch(d.batch)
		if err != nil {
			return err
		}
		d.batch.Reset()
		if end {
			d.batch = nil
		}
		// 每次提交后重建映射，确保新增文档可被数字ID正确引用
		_ = d.rebuildNumericIDMapping()
		return err
	}
	return nil
}

func (d *BleveSearchEngine) Search(helpPackages []string, text string, titleOnly bool, pageSize, pageNum int, group string) (*GeneralSearchResult, int, int, int, error) {
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
	for _, i := range helpPackages {
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
	// 设置要被返回的数据
	req.Fields = []string{"*"}
	res, err := d.Index.Search(req)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	var resultList = make(MatchCollection, 0)
	for _, hit := range res.Hits {
		result := MatchResult{
			// 对用户展示“纯数字且不长”的ID，通过内存映射实现 ULID -> 数字ID
			// 注意：该数字ID在索引变动（增删改）后会重新分配，从而导致旧数字ID失效，这是预期行为
			ID:     d.getNumericIDByULID(hit.ID),
			Fields: hit.Fields,
			Score:  hit.Score,
		}
		resultList = append(resultList, &result)
	}
	// 转换搜索格式
	responseResult := GeneralSearchResult{
		Hits:  resultList,
		Total: res.Total,
	}
	total := int(res.Total)
	pageStart := (pageNum - 1) * pageSize
	pageEnd := pageStart + len(res.Hits)
	return &responseResult, total, pageStart, pageEnd, nil
}

func (d *BleveSearchEngine) ListAllDocumentIDs() ([]string, error) {
	if d.Index == nil {
		return nil, nil
	}
	// 如果已有映射，直接返回按字典序排序好的列表；否则重建后返回
	if d.idList == nil {
		if err := d.rebuildNumericIDMapping(); err != nil {
			return nil, err
		}
	}
	// 返回一个拷贝，避免外部修改内部切片
	out := make([]string, len(d.idList))
	copy(out, d.idList)
	return out, nil
}

func (d *BleveSearchEngine) PaginateDocuments(pageSize, pageNum int, group, from, title string) (uint64, []*HelpTextItem, error) {
	var items []*HelpTextItem
	// 只有Keyword才支持NewTermQuery
	conjunctionQuery := bleve.NewConjunctionQuery()
	if group != "" {
		groupQuery := bleve.NewWildcardQuery(fmt.Sprintf("*%s*", group))
		groupQuery.SetField("group")
		conjunctionQuery.AddQuery(groupQuery)
	}
	if from != "" {
		fromQuery := bleve.NewWildcardQuery(fmt.Sprintf("*%s*", from))
		fromQuery.SetField("from")
		conjunctionQuery.AddQuery(fromQuery)
	}
	if title != "" {
		titleQuery := bleve.NewWildcardQuery(fmt.Sprintf("*%s*", title))
		titleQuery.SetField("title")
		conjunctionQuery.AddQuery(titleQuery)
	}

	// 计算分页参数
	fromInt := (pageNum - 1) * pageSize // 起始位置
	if fromInt < 0 {
		fromInt = 0
	}
	var searchRequest *bleve.SearchRequest
	// 创建查询请求，设置分页参数
	if group == "" && from == "" && title == "" {
		searchRequest = bleve.NewSearchRequestOptions(bleve.NewMatchAllQuery(), pageSize, fromInt, false)
	} else {
		searchRequest = bleve.NewSearchRequestOptions(conjunctionQuery, pageSize, fromInt, true)
	}
	searchRequest.Fields = []string{"*"} // 设置需要返回的字段

	// 执行查询
	searchResult, err := d.Index.Search(searchRequest)
	if err != nil {
		return 0, nil, err
	}

	// 处理结果
	for _, hit := range searchResult.Hits {
		fields := hit.Fields
		item := &HelpTextItem{
			Group:       fmt.Sprintf("%v", fields["group"]),
			From:        fmt.Sprintf("%v", fields["from"]),
			Title:       fmt.Sprintf("%v", fields["title"]),
			Content:     fmt.Sprintf("%v", fields["content"]),
			PackageName: fmt.Sprintf("%v", fields["package"]),
			KeyWords:    "",  // 暂时空值
			RelatedExt:  nil, // 暂时空值
		}
		items = append(items, item)
	}
	return searchResult.Total, items, nil
}

func (d *BleveSearchEngine) GetItemByID(id string) (*HelpTextItem, error) {
	log := logger.M()
	document, err := d.Index.Document(id)
	if err != nil {
		return nil, err
	}
	// 检查是否找到文档
	if document == nil {
		return nil, errors.New("未找到匹配的文档")
	}
	item := HelpTextItem{}
	// 看了看源码，意思就是这样访问文档内的所有fields
	document.VisitFields(func(field index.Field) {
		name := field.Name()
		value := string(field.Value())
		// 这里的代码有点抽象……
		switch name {
		case "group":
			item.Group = value
		case "from":
			item.From = value
		case "title":
			item.Title = value
		case "content":
			item.Content = value
		case "package":
			item.PackageName = value
			// 好像会碰到Type的参数？
		default:
			log.Debugf("这是个什么参数 %s", name)
		}
	})
	return &item, nil
}

// 精确查询title
func (d *BleveSearchEngine) GetHelpTextItemByTermTitle(title string) (*HelpTextItem, error) {
	newTermQuery := query.NewMatchQuery(title)
	newTermQuery.SetField("title") // 精确匹配title
	req := bleve.NewSearchRequest(newTermQuery)
	req.Fields = []string{"*"}
	res, err := d.Index.Search(req)
	if err != nil {
		return nil, err
	}
	// 取出结果
	if len(res.Hits) > 0 {
		fields := res.Hits[0].Fields
		return &HelpTextItem{
			Group:       fmt.Sprintf("%v", fields["group"]),
			From:        fmt.Sprintf("%v", fields["from"]),
			Title:       fmt.Sprintf("%v", fields["title"]),
			Content:     fmt.Sprintf("%v", fields["content"]),
			PackageName: fmt.Sprintf("%v", fields["package"]),
			// 这俩是什么东西？！
			KeyWords:   "",
			RelatedExt: nil,
		}, nil
	}
	return nil, errors.New("查询失败，未查询到数据")
}
