package pprof

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	stdpprof "net/http/pprof"
	"net/url"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

type Service struct{}

type ProfileQuery struct {
	Seconds int `query:"seconds"`
	Debug   int `query:"debug"`
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/cmdline", s.GetCmdline, func(o *huma.Operation) {
		o.Description = "获取命令行参数 pprof 视图"
	})
	huma.Get(grp, "/profile", s.GetProfile, func(o *huma.Operation) {
		o.Description = "获取 CPU profile"
	})
	huma.Get(grp, "/trace", s.GetTrace, func(o *huma.Operation) {
		o.Description = "获取 trace 采样"
	})
	huma.Get(grp, "/allocs", s.GetAllocs, func(o *huma.Operation) {
		o.Description = "获取 allocs profile"
	})
	huma.Get(grp, "/block", s.GetBlock, func(o *huma.Operation) {
		o.Description = "获取 block profile"
	})
	huma.Get(grp, "/goroutine", s.GetGoroutine, func(o *huma.Operation) {
		o.Description = "获取 goroutine profile"
	})
	huma.Get(grp, "/heap", s.GetHeap, func(o *huma.Operation) {
		o.Description = "获取 heap profile"
	})
	huma.Get(grp, "/mutex", s.GetMutex, func(o *huma.Operation) {
		o.Description = "获取 mutex profile"
	})
	huma.Get(grp, "/threadcreate", s.GetThreadcreate, func(o *huma.Operation) {
		o.Description = "获取 threadcreate profile"
	})
}

func (s *Service) GetCmdline(_ context.Context, _ *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(http.HandlerFunc(stdpprof.Cmdline), "/cmdline", nil), nil
}

func (s *Service) GetProfile(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(http.HandlerFunc(stdpprof.Profile), "/profile", buildProfileQuery(query, true)), nil
}

func (s *Service) GetTrace(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(http.HandlerFunc(stdpprof.Trace), "/trace", buildProfileQuery(query, true)), nil
}

func (s *Service) GetAllocs(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(stdpprof.Handler("allocs"), "/allocs", buildProfileQuery(query, false)), nil
}

func (s *Service) GetBlock(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(stdpprof.Handler("block"), "/block", buildProfileQuery(query, false)), nil
}

func (s *Service) GetGoroutine(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(stdpprof.Handler("goroutine"), "/goroutine", buildProfileQuery(query, false)), nil
}

func (s *Service) GetHeap(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(stdpprof.Handler("heap"), "/heap", buildProfileQuery(query, false)), nil
}

func (s *Service) GetMutex(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(stdpprof.Handler("mutex"), "/mutex", buildProfileQuery(query, false)), nil
}

func (s *Service) GetThreadcreate(_ context.Context, query *ProfileQuery) (*huma.StreamResponse, error) {
	return newPprofStreamResponse(stdpprof.Handler("threadcreate"), "/threadcreate", buildProfileQuery(query, false)), nil
}

func buildProfileQuery(query *ProfileQuery, allowSeconds bool) url.Values {
	values := url.Values{}
	if query == nil {
		return values
	}
	if allowSeconds && query.Seconds > 0 {
		values.Set("seconds", strconv.Itoa(query.Seconds))
	}
	if query.Debug >= 0 {
		values.Set("debug", strconv.Itoa(query.Debug))
	}
	return values
}

func newPprofStreamResponse(handler http.Handler, path string, query url.Values) *huma.StreamResponse {
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			requestURL := path
			if encoded := query.Encode(); encoded != "" {
				requestURL += "?" + encoded
			}

			req, err := http.NewRequest(http.MethodGet, requestURL, nil)
			if err != nil {
				ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
				_, _ = io.WriteString(ctx.BodyWriter(), err.Error())
				return
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			for key, values := range recorder.Header() {
				if len(values) > 0 {
					ctx.SetHeader(key, values[0])
				}
			}
			_, _ = io.Copy(ctx.BodyWriter(), recorder.Body)
			if flusher, ok := ctx.BodyWriter().(http.Flusher); ok {
				flusher.Flush()
			}
		},
	}
}
