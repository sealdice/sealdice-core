package paginate

var (
	DefaultRequestMaxSize int64 = 50
	DefaultRequestSize    int64 = 20
)

type IRequest interface {
	// GetPage 获取当前页数
	GetPage() int64
	// GetSize 获取拉去多少条数据
	GetSize() int64
	// MaxSize 安全大小，为了防止拉所有数据导致拖垮数据
	MaxSize() int64
}

func TotalRequest(adapt IAdapter, request IRequest) IPaginate {
	total, _ := adapt.Length()
	size := request.GetSize()
	if maxSize := request.MaxSize(); maxSize > 0 {
		if size > maxSize {
			size = maxSize
		}
	}
	return TotalPaginate(adapt, size, request.GetPage(), total)
}

func SimpleRequest(adapt IAdapter, request IRequest) IPaginate {
	size := request.GetSize()
	if maxSize := request.MaxSize(); maxSize > 0 {
		if size > maxSize {
			size = maxSize
		}
	}
	return SimplePaginate(adapt, size, request.GetPage())
}

type Request struct {
	Page int64 `json:"page" form:"page" param:"page" uri:"page" query:"page" xml:"Page"`
	Size int64 `json:"size" form:"size" param:"size" uri:"size" query:"size" xml:"Size"`
}

func (r *Request) GetPage() int64 {
	return r.Page
}

func (r *Request) GetSize() int64 {
	if r.Size == 0 {
		return DefaultRequestSize
	}
	return r.Size
}
func (r *Request) MaxSize() int64 {
	return DefaultRequestMaxSize
}
