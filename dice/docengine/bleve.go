package docengine

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync/atomic"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	index "github.com/blevesearch/bleve_index_api"

	log "sealdice-core/utils/kratos"
)

type BleveSearchEngine struct {
	Index     bleve.Index
	batch     *bleve.Batch
	batchSize int
	CurID     uint64
}

var indexDir = "./data/_index"
var reSpace = regexp.MustCompile(`\s+`)

// getNextID 使用原子操作，避免并发问题
func (d *BleveSearchEngine) getNextID() string {
	// 使用原子操作安全递增 CurID
	nextID := atomic.AddUint64(&d.CurID, 1)
	return strconv.FormatUint(nextID, 10)
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
	keywordMapping := bleve.NewKeywordFieldMapping()
	// 注意： 这里group,from,title，package都是keywordMapping，这样就能进行精确搜索。
	docMapping.AddFieldMappingsAt("group", keywordMapping)
	docMapping.AddFieldMappingsAt("from", keywordMapping)
	docMapping.AddFieldMappingsAt("title", contentFieldMapping)
	// Content才是真正的文档
	docMapping.AddFieldMappingsAt("content", contentFieldMapping)
	docMapping.AddFieldMappingsAt("package", keywordMapping)
	mapping.AddDocumentMapping("helpdoc", docMapping)
	mapping.TypeField = "_type"
	i, err := bleve.New(indexDir, mapping)
	if err != nil {
		return err
	}
	d.Index = i
	// 初始化ID列表
	d.CurID = 0
	// 初始化新的batch
	d.batch = d.Index.NewBatch()
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
		// 如果是最后一批
		if end {
			d.batch.Reset()
			d.batch = nil
		} else {
			// 否则仅重置batch
			d.batch.Reset()
		}
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
			ID:     hit.ID,
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

// 下面的代码都应该重构，因为它们返回的不是我们想要的结果
// PaginateAllDocuments 分页查询所有文档
// TODO:这里坏了，没有办法用，本来应该是精确匹配NewMatchQuery
func (d *BleveSearchEngine) PaginateDocuments(pageSize, pageNum int, group, from, title string) (uint64, []*HelpTextItem, error) {
	var items []*HelpTextItem
	// 只有Keyword才支持NewTermQuery
	conjunctionQuery := bleve.NewConjunctionQuery()
	if group != "" {
		groupQuery := bleve.NewTermQuery(group)
		groupQuery.SetField("group")
		conjunctionQuery.AddQuery(groupQuery)
	}
	if from != "" {
		fromQuery := bleve.NewTermQuery(from)
		fromQuery.SetField("from")
		conjunctionQuery.AddQuery(fromQuery)
	}
	if title != "" {
		titleQuery := bleve.NewTermQuery(title)
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
	newTermQuery := query.NewTermQuery(title)
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
