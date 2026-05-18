package paginate

type IRender interface {
	// SetTotal 总数
	SetTotal(total int64)
	// SetSimple 是否为简单模式分页
	SetSimple(simple bool)
	// SetPerPage 每页的数量
	SetPerPage(perPage int64)
	// SetCurrentPage 当前页
	SetCurrentPage(currentPage int64)
	// SetLastPage 最后一页
	SetLastPage(lastPage int64)
	// SetData 数据集
	SetData(data any)
	// SetHasMore 是否可以进行下一页
	SetHasMore(hasMore bool)

	Render() any
}

type Render struct {
	//总数
	Total int64 `json:"total" xml:"total"`
	//数据集
	Data any `json:"data" xml:"data"`
	//是否为简单模式分页
	Simple bool `json:"simple" xml:"simple"`
	//每页的数量
	PerPage int64 `json:"per_page" xml:"perPage"`
	//当前页
	CurrentPage int64 `json:"current_page" xml:"currentPage"`
	//最后一页
	LastPage int64 `json:"last_page" xml:"lastPage"`
	//是否有下一页
	HasMore bool `json:"has_more" xml:"hasMore"`
}

func (r *Render) SetTotal(total int64) {
	r.Total = total
}

func (r *Render) SetSimple(simple bool) {
	r.Simple = simple
}

func (r *Render) SetPerPage(perPage int64) {
	r.PerPage = perPage
}

func (r *Render) SetCurrentPage(currentPage int64) {
	r.CurrentPage = currentPage
}

func (r *Render) SetLastPage(lastPage int64) {
	r.LastPage = lastPage
}

func (r *Render) SetData(data any) {
	r.Data = data
}

func (r *Render) SetHasMore(hasMore bool) {
	r.HasMore = hasMore
}
func (r *Render) Render() any {
	return r
}
