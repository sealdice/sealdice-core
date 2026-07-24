package docengine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/fy0/bluge"
	"github.com/fy0/bluge/index"
	"github.com/fy0/bluge/search"
	"github.com/oklog/ulid/v2"

	"sealdice-core/logger"
)

type BlugeSearchEngine struct {
	// mu serializes writer access and the numeric ID cache.
	mu        sync.Mutex
	Writer    *bluge.Writer
	batch     *index.Batch
	batchSize int
	CurID     uint64

	freshlyCreated bool
	idList         []string
	idToNumber     map[string]int
	numericIDDirty bool
}

const (
	DefaultCacheDir = "./cache/_help_cache"
	DefaultIndexDir = DefaultCacheDir + "/_index"
	indexSchemaFile = "schema_version"
	indexSchema     = "2"
	groupExactField = "_group_exact"
)

var indexDir = DefaultIndexDir
var reSpace = regexp.MustCompile(`\s+`)

const (
	deleteSearchBatchSize = 1000
	writeBatchSize        = 50
)

func (d *BlugeSearchEngine) getNextID() string {
	return ulid.Make().String()
}

func NewBlugeSearchEngine() (*BlugeSearchEngine, error) {
	engine := &BlugeSearchEngine{}
	if err := engine.Init(); err != nil {
		return nil, err
	}
	return engine, nil
}

func (d *BlugeSearchEngine) GetSuffixText() string {
	return "(本次搜索由全文搜索完成)"
}

func (d *BlugeSearchEngine) GetPrefixText() string {
	return "[全文搜索]"
}

func (d *BlugeSearchEngine) GetShowBestOffset() int {
	return 1
}

func inspectIndexDir(path string) (hasBlugeSnapshot, hasBleveMarker, schemaCurrent bool, err error) {
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, false, nil
	}
	if err != nil {
		return false, false, false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".snp" {
			hasBlugeSnapshot = true
		}
		if entry.Name() == "index_meta.json" || entry.Name() == "store" {
			hasBleveMarker = true
		}
	}
	schemaData, schemaErr := os.ReadFile(filepath.Join(path, indexSchemaFile))
	if schemaErr == nil {
		schemaCurrent = string(schemaData) == indexSchema
	} else if !errors.Is(schemaErr, os.ErrNotExist) {
		return false, false, false, schemaErr
	}
	return hasBlugeSnapshot, hasBleveMarker, schemaCurrent, nil
}

func openBlugeWriter(path string) (*bluge.Writer, bool, error) {
	hasSnapshot, hasBleveMarker, schemaCurrent, err := inspectIndexDir(path)
	if err != nil {
		return nil, false, err
	}
	requiresRebuild := hasBleveMarker || (hasSnapshot && !schemaCurrent)
	freshlyCreated := !hasSnapshot || requiresRebuild
	if requiresRebuild {
		if removeErr := os.RemoveAll(path); removeErr != nil {
			return nil, false, fmt.Errorf("remove incompatible search index: %w", removeErr)
		}
	}

	writer, err := bluge.OpenWriter(bluge.DefaultConfig(path))
	if err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(filepath.Join(path, indexSchemaFile), []byte(indexSchema), 0o600); err != nil {
		_ = writer.Close()
		return nil, false, fmt.Errorf("write search index schema: %w", err)
	}
	return writer, freshlyCreated, nil
}

func (d *BlugeSearchEngine) Init() error {
	writer, freshlyCreated, err := openBlugeWriter(indexDir)
	if err != nil {
		return err
	}

	d.Writer = writer
	d.CurID = 0
	d.freshlyCreated = freshlyCreated
	d.batch = bluge.NewBatch()
	d.batchSize = 0
	d.markNumericIDDirtyLocked()
	return nil
}

func (d *BlugeSearchEngine) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.Writer != nil {
		_ = d.Writer.Close()
		d.Writer = nil
	}
}

func (d *BlugeSearchEngine) GetTotalID() uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.CurID
}

func (d *BlugeSearchEngine) IndexFreshlyCreated() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.freshlyCreated
}

func (d *BlugeSearchEngine) markNumericIDDirtyLocked() {
	d.numericIDDirty = true
}

func (d *BlugeSearchEngine) ensureNumericIDMappingLocked() error {
	if d.Writer == nil {
		d.idList = nil
		d.idToNumber = nil
		d.CurID = 0
		d.numericIDDirty = false
		return nil
	}
	if !d.numericIDDirty && d.idList != nil && d.idToNumber != nil {
		return nil
	}
	return d.rebuildNumericIDMappingLocked()
}

func (d *BlugeSearchEngine) rebuildNumericIDMappingLocked() error {
	ids, err := d.listAllDocumentIDsLocked()
	if err != nil {
		return err
	}

	sort.Strings(ids)
	d.idList = ids
	d.idToNumber = make(map[string]int, len(ids))
	for i, id := range ids {
		d.idToNumber[id] = i + 1
	}
	d.CurID = uint64(len(ids))
	d.numericIDDirty = false
	return nil
}

func (d *BlugeSearchEngine) getNumericIDByULID(id string) string {
	if num, ok := d.idToNumber[id]; ok {
		return strconv.Itoa(num)
	}
	return id
}

func (d *BlugeSearchEngine) newReaderLocked() (*bluge.Reader, error) {
	if d.Writer == nil {
		return nil, errors.New("搜索引擎已关闭")
	}
	return d.Writer.Reader()
}

func storedFields(match *search.DocumentMatch) (string, map[string]interface{}, error) {
	var id string
	fields := make(map[string]interface{}, 5)
	err := match.VisitStoredFields(func(field string, value []byte) bool {
		if field == "_id" {
			id = string(value)
		} else {
			fields[field] = string(value)
		}
		return true
	})
	return id, fields, err
}

func helpItemFromFields(fields map[string]interface{}) *HelpTextItem {
	return &HelpTextItem{
		Group:       fmt.Sprintf("%v", fields["group"]),
		From:        fmt.Sprintf("%v", fields["from"]),
		Title:       fmt.Sprintf("%v", fields["title"]),
		Content:     fmt.Sprintf("%v", fields["content"]),
		PackageName: fmt.Sprintf("%v", fields["package"]),
	}
}

func (d *BlugeSearchEngine) deleteByField(field, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.Writer == nil {
		return nil
	}
	deletedAny := false
	for {
		reader, err := d.newReaderLocked()
		if err != nil {
			return err
		}
		query := bluge.NewTermQuery(value).SetField(field)
		matches, err := reader.Search(context.Background(), bluge.NewTopNSearch(deleteSearchBatchSize, query))
		if err != nil {
			_ = reader.Close()
			return err
		}

		ids := make([]string, 0, deleteSearchBatchSize)
		for {
			match, nextErr := matches.Next()
			if nextErr != nil {
				_ = reader.Close()
				return nextErr
			}
			if match == nil {
				break
			}
			id, _, visitErr := storedFields(match)
			if visitErr != nil {
				_ = reader.Close()
				return visitErr
			}
			if id != "" {
				ids = append(ids, id)
			}
		}
		if closeErr := reader.Close(); closeErr != nil {
			return closeErr
		}
		if len(ids) == 0 {
			break
		}

		batch := bluge.NewBatch()
		for _, id := range ids {
			batch.Delete(bluge.Identifier(id))
		}
		if err := d.Writer.Batch(batch); err != nil {
			return err
		}
		deletedAny = true
		if len(ids) < deleteSearchBatchSize {
			break
		}
	}

	if deletedAny {
		d.markNumericIDDirtyLocked()
	}
	return nil
}

func (d *BlugeSearchEngine) DeleteByFrom(path string) error {
	return d.deleteByField("from", path)
}

func (d *BlugeSearchEngine) DeleteByGroup(group string) error {
	return d.deleteByField(groupExactField, normalizeExactValue(group))
}

func normalizeExactValue(value string) string {
	return strings.ToLower(value)
}

func helpDocument(id string, item HelpTextItem) *bluge.Document {
	return bluge.NewDocument(id).
		AddField(bluge.NewStoredOnlyField("group", []byte(item.Group))).
		AddField(bluge.NewKeywordField(groupExactField, normalizeExactValue(item.Group))).
		AddField(bluge.NewKeywordField("from", item.From).StoreValue()).
		AddField(bluge.NewTextField("title", item.Title).StoreValue().SearchTermPositions()).
		AddField(bluge.NewTextField("content", item.Content).StoreValue().SearchTermPositions()).
		AddField(bluge.NewKeywordField("package", item.PackageName).StoreValue())
}

func (d *BlugeSearchEngine) AddItem(item HelpTextItem) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.batch == nil {
		return "", errors.New("已通过end参数执行AddItemApply，不允许新增文档。请检查代码逻辑")
	}
	id := d.getNextID()
	doc := helpDocument(id, item)
	d.batch.Update(doc.ID(), doc)
	d.batchSize++
	if d.batchSize >= writeBatchSize {
		if err := d.addItemApplyLocked(false); err != nil {
			return "", err
		}
	}
	return id, nil
}

func (d *BlugeSearchEngine) AddItemApply(end bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.addItemApplyLocked(end)
}

func (d *BlugeSearchEngine) addItemApplyLocked(end bool) error {
	if d.batch == nil {
		return nil
	}
	if d.batchSize == 0 {
		if end {
			d.batch = nil
		}
		return nil
	}
	if d.Writer == nil {
		return errors.New("搜索引擎已关闭")
	}
	if err := d.Writer.Batch(d.batch); err != nil {
		return err
	}
	d.batch.Reset()
	d.batchSize = 0
	if end {
		d.batch = nil
	}
	d.markNumericIDDirtyLocked()
	return nil
}

func (d *BlugeSearchEngine) Search(helpPackages []string, text string, titleOnly bool, pageSize, pageNum int, group string) (*GeneralSearchResult, int, int, int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	titleOrContent := bluge.NewBooleanQuery().SetMinShould(1)
	titleOrContent.AddShould(bluge.NewMatchPhraseQuery(text).SetField("title"))
	if !titleOnly {
		for _, term := range reSpace.Split(text, -1) {
			if term != "" {
				titleOrContent.AddShould(bluge.NewMatchPhraseQuery(term).SetField("content"))
			}
		}
	}

	query := bluge.NewBooleanQuery().AddMust(titleOrContent)
	for _, packageName := range helpPackages {
		query.AddMust(bluge.NewTermQuery(packageName).SetField("package"))
	}
	if group != "" {
		query.AddMust(bluge.NewTermQuery(normalizeExactValue(group)).SetField(groupExactField))
	}

	if pageSize < 0 {
		pageSize = 0
	}
	from := (pageNum - 1) * pageSize
	if from < 0 {
		from = 0
	}
	reader, err := d.newReaderLocked()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer func() { _ = reader.Close() }()

	request := bluge.NewTopNSearch(pageSize, query).SetFrom(from).WithStandardAggregations()
	matches, err := reader.Search(context.Background(), request)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	if err := d.ensureNumericIDMappingLocked(); err != nil {
		logger.M().Warnf("[帮助文档] 重建数字ID映射失败，将退回内部ID展示: %v", err)
	}
	resultList := make(MatchCollection, 0, pageSize)
	for {
		match, nextErr := matches.Next()
		if nextErr != nil {
			return nil, 0, 0, 0, nextErr
		}
		if match == nil {
			break
		}
		id, fields, visitErr := storedFields(match)
		if visitErr != nil {
			return nil, 0, 0, 0, visitErr
		}
		resultList = append(resultList, &MatchResult{
			ID:     d.getNumericIDByULID(id),
			Fields: fields,
			Score:  match.Score,
		})
	}

	total := int(matches.Aggregations().Count())
	pageEnd := from + len(resultList)
	return &GeneralSearchResult{Hits: resultList, Total: uint64(total)}, total, from, pageEnd, nil
}

func (d *BlugeSearchEngine) listAllDocumentIDsLocked() ([]string, error) {
	reader, err := d.newReaderLocked()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	matches, err := reader.Search(context.Background(), bluge.NewAllMatches(bluge.NewMatchAllQuery()))
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for {
		match, nextErr := matches.Next()
		if nextErr != nil {
			return nil, nextErr
		}
		if match == nil {
			break
		}
		id, _, visitErr := storedFields(match)
		if visitErr != nil {
			return nil, visitErr
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (d *BlugeSearchEngine) ListAllDocumentIDs() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureNumericIDMappingLocked(); err != nil {
		return nil, err
	}
	out := make([]string, len(d.idList))
	copy(out, d.idList)
	return out, nil
}

func (d *BlugeSearchEngine) PaginateDocuments(pageSize, pageNum int, group, from, title string) (uint64, []*HelpTextItem, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	queries := make([]bluge.Query, 0, 3)
	if group != "" {
		queries = append(queries, bluge.NewWildcardQuery(fmt.Sprintf("*%s*", normalizeExactValue(group))).SetField(groupExactField))
	}
	if from != "" {
		queries = append(queries, bluge.NewWildcardQuery(fmt.Sprintf("*%s*", from)).SetField("from"))
	}
	if title != "" {
		queries = append(queries, bluge.NewWildcardQuery(fmt.Sprintf("*%s*", title)).SetField("title"))
	}

	var query bluge.Query = bluge.NewMatchAllQuery()
	if len(queries) > 0 {
		query = bluge.NewBooleanQuery().AddMust(queries...)
	}
	if pageSize < 0 {
		pageSize = 0
	}
	fromIndex := (pageNum - 1) * pageSize
	if fromIndex < 0 {
		fromIndex = 0
	}
	reader, err := d.newReaderLocked()
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = reader.Close() }()

	request := bluge.NewTopNSearch(pageSize, query).SetFrom(fromIndex).WithStandardAggregations()
	matches, err := reader.Search(context.Background(), request)
	if err != nil {
		return 0, nil, err
	}
	items := make([]*HelpTextItem, 0, pageSize)
	for {
		match, nextErr := matches.Next()
		if nextErr != nil {
			return 0, nil, nextErr
		}
		if match == nil {
			break
		}
		_, fields, visitErr := storedFields(match)
		if visitErr != nil {
			return 0, nil, visitErr
		}
		items = append(items, helpItemFromFields(fields))
	}
	return matches.Aggregations().Count(), items, nil
}

func (d *BlugeSearchEngine) GetItemByID(id string) (*HelpTextItem, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.getItemByIDLocked(id)
}

func (d *BlugeSearchEngine) getItemByIDLocked(id string) (*HelpTextItem, error) {
	reader, err := d.newReaderLocked()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	query := bluge.NewTermQuery(id).SetField("_id")
	matches, err := reader.Search(context.Background(), bluge.NewTopNSearch(1, query))
	if err != nil {
		return nil, err
	}
	match, err := matches.Next()
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, errors.New("未找到匹配的文档")
	}
	_, fields, err := storedFields(match)
	if err != nil {
		return nil, err
	}
	return helpItemFromFields(fields), nil
}

func (d *BlugeSearchEngine) GetHelpTextItemByTermTitle(title string) (*HelpTextItem, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	reader, err := d.newReaderLocked()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	query := bluge.NewMatchQuery(title).SetField("title")
	matches, err := reader.Search(context.Background(), bluge.NewTopNSearch(1, query))
	if err != nil {
		return nil, err
	}
	match, err := matches.Next()
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, errors.New("查询失败，未查询到数据")
	}
	_, fields, err := storedFields(match)
	if err != nil {
		return nil, err
	}
	return helpItemFromFields(fields), nil
}

var _ SearchEngine = (*BlugeSearchEngine)(nil)
