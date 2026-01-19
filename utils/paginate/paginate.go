package paginate

import (
	"errors"
	"math"
)

var DefaultRender = new(Render)

type IAdapter interface {
	// Length 数据长度
	Length() (int64, error)
	// Slice 切割数据(分页)
	Slice(offset, length int64, data any) error
}

type IPaginate interface {
	Clone() IPaginate
	// SetRender 设置渲染
	SetRender(render IRender) IPaginate
	// SetData 该data是查询数据库的数据源
	SetData(data any) IPaginate
	// SetCurrentPage 设置当前页数
	SetCurrentPage(currentPage int64)
	// GetCurrentPage 获取当前页页码
	GetCurrentPage() int64
	// GetTotal 获取数据总条数
	GetTotal() (int64, error)
	// GetListRows 获取每页数量
	GetListRows() int64
	//GetLastPage 获取最后一页页码
	GetLastPage() (int64, error)
	//HasPages 数据是否足够分页
	HasPages() bool
	// Get 获取数据
	Get(data any) error
	// Render 获取Paginate结构体数据
	Render(data any) any
}
type Paginate struct {
	//数据集适配器
	adapter IAdapter
	//渲染
	render IRender
	// 模式
	simple bool
	//设置输出源
	Data any
	//当前页
	currentPage int64
	//最后一页
	lastPage int64
	//数据总数
	total int64
	//每页数量
	listRows int64
	//是否有下一页
	hasMore bool
}

func SimplePaginate(adapter IAdapter, listRows, currentPage int64) IPaginate {
	return Make(adapter, listRows, currentPage, 0, true)
}

func TotalPaginate(adapter IAdapter, listRows, currentPage, total int64) IPaginate {
	return Make(adapter, listRows, currentPage, total, false)
}

func Make(adapter IAdapter, listRows int64, currentPage int64, total int64, simple bool) IPaginate {
	p := &Paginate{}
	p.Clone()
	p.simple = simple
	p.adapter = adapter
	p.listRows = listRows
	if p.simple {
		p.SetCurrentPage(currentPage)
		count, err := p.adapter.Length()
		if err != nil {
			panic(err)
		}
		p.total = count
		p.hasMore = count > p.currentPage
	} else {
		p.total = total
		p.lastPage = int64(math.Ceil(float64(p.total) / float64(p.listRows)))
		p.SetCurrentPage(currentPage)
		p.hasMore = p.currentPage < p.lastPage
	}
	return p
}
func (p *Paginate) Clone() IPaginate {
	p.simple = false
	p.adapter = nil
	p.Data = nil
	p.currentPage = 0
	p.lastPage = 0
	p.total = 0
	p.listRows = 0
	p.hasMore = false
	if p.render == nil {
		p.render = DefaultRender
	}
	return p
}

// SetData 设置data
func (p *Paginate) SetData(data any) IPaginate {
	p.Data = data
	return p
}

func (p *Paginate) SetRender(render IRender) IPaginate {
	p.render = render
	return p
}

// SetCurrentPage 设置当前页数
func (p *Paginate) SetCurrentPage(currentPage int64) {
	if !p.simple && currentPage > p.lastPage {
		if p.lastPage > 0 {
			p.currentPage = p.lastPage
			return
		} else {
			p.currentPage = 1
			return
		}
	}
	p.currentPage = currentPage
}

// GetTotal 获取数据总条数
func (p *Paginate) GetTotal() (int64, error) {
	return p.total, nil
}

// GetListRows 获取每页数量
func (p *Paginate) GetListRows() int64 {
	return p.listRows
}

// GetCurrentPage 获取当前页页码
func (p *Paginate) GetCurrentPage() int64 {
	return p.currentPage
}

// GetLastPage 获取最后一页页码
func (p *Paginate) GetLastPage() (int64, error) {
	if p.simple {
		return 0, errors.New("not support last")
	}
	return p.lastPage, nil
}

// HasPages 数据是否足够分页
func (p *Paginate) HasPages() bool {
	return !(p.currentPage == 1 && !p.hasMore)
}

// Get 获取指定长度的数据
func (p *Paginate) Get(data any) error {
	var offset int64
	data = p.initData(data)
	page := p.GetCurrentPage()
	if page > 1 {
		offset = (page - 1) * p.listRows
	}
	return p.adapter.Slice(offset, p.listRows, data)
}
func (p *Paginate) initData(dest any) any {
	if dest != nil {
		p.SetData(dest)
	}
	if p.Data == nil {
		panic("Dest uninitialized")
	}
	return p.Data
}

// Render 渲染数据
func (p *Paginate) Render(data any) any {
	total, _ := p.GetTotal()
	p.render.SetTotal(total)
	p.render.SetSimple(p.simple)
	p.render.SetPerPage(p.GetListRows())
	p.render.SetCurrentPage(p.GetCurrentPage())
	p.render.SetHasMore(p.hasMore)
	lastPage, _ := p.GetLastPage()
	p.render.SetLastPage(lastPage)
	if data == nil && p.Data == nil {
		return p.render.Render()
	}
	if data != nil && p.Data == nil {
		if err := p.Get(data); err != nil {
			panic(err)
		}
		p.render.SetData(data)
		return p.render.Render()
	}
	if data == nil && p.Data != nil {
		p.render.SetData(p.Data)
		return p.render.Render()
	}
	if data != nil && p.Data != nil {
		p.render.SetData(data)
		return p.render.Render()
	}
	return p.render.Render()
}
