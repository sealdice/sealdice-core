package docengine

type MatchResult struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
	Score  float64                `json:"score"`
}

type Fields struct {
}

type MatchCollection []*MatchResult

// GeneralSearchResult Copied from bleve
type GeneralSearchResult struct {
	Hits  MatchCollection
	Total uint64
}

type HelpTextItem struct {
	Group       string
	From        string
	Title       string
	Content     string
	PackageName string
	// 这俩玩意有用？
	KeyWords   string
	RelatedExt []string
}

// SearchEngine TODO: 进一步优化结构，封装成通用的搜索
type SearchEngine interface {
	GetSuffixText() string
	GetPrefixText() string
	GetShowBestOffset() int
	// Init 初始化搜索引擎
	Init() error
	// Close 关闭搜索引擎
	Close()
	// AddItem 添加文档条目，返回添加文档的ID
	AddItem(item HelpTextItem) (string, error)
	// AddItemApply 提交文档条目
	AddItemApply(end bool) error
	// Search 搜索文档条目
	Search(helpPackages []string, text string, titleOnly bool, pageSize, pageNum int, group string) (*GeneralSearchResult, int, int, int, error)
	// GetHelpTextItemByTermTitle 精确查询title，用于嵌套获取数据的情形
	GetHelpTextItemByTermTitle(title string) (*HelpTextItem, error)
	// GetItemByID 通过ID获取Item数据的方案
	GetItemByID(id string) (*HelpTextItem, error)
	// PaginateDocuments 分页获取数据
	PaginateDocuments(pageSize, pageNum int, group, from, title string) (uint64, []*HelpTextItem, error)
	// GetTotalID 获取当前ID总数，注意，ID必须是顺序排列的
	GetTotalID() uint64
}
