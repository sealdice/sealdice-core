package base

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	nanoid "github.com/matoous/go-nanoid/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

var startTime = time.Now()

// BaseService 基础服务，封装dice依赖
type BaseService struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

// NewBaseService 创建新的基础服务实例
// 特殊增加dm降低封装层 - 或许应该传入dm获取dice才是正道。
func NewBaseService(dm *dice.DiceManager) *BaseService {
	return &BaseService{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *BaseService) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/health", s.Health, func(o *huma.Operation) {
		o.Description = "检查服务是否正常"
		o.Summary = "检查服务是否正常"
	})
	huma.Get(grp, "/network-health", s.NetworkHealth, func(o *huma.Operation) {
		o.Description = "检查网络质量"
		o.Summary = "检查网络质量"
	})
	huma.Get(grp, "/overview", s.Overview, func(o *huma.Operation) {
		o.Description = "获取基础运行概览"
		o.Summary = "获取基础运行概览"
	})
	huma.Post(grp, "/login", s.Login, func(o *huma.Operation) {
		o.Description = "登录获取Token"
		o.Summary = "登录获取Token"
	})
	huma.Get(grp, "/login/salt", s.LoginSalt, func(o *huma.Operation) {
		o.Description = "获取登录盐"
		o.Summary = "获取登录盐"
	})
	huma.Get(grp, "/security-check", s.SecurityCheck, func(o *huma.Operation) {
		o.Description = "检查安全状态"
		o.Summary = "检查安全状态"
	})
}

// Health 健康检查处理函数
func (s *BaseService) Health(_ context.Context, _ *request.Empty) (*response.ItemResponse[HealthData], error) {
	if s.dice == nil {
		return nil, huma.Error500InternalServerError("Dice instance is nil,contact administrator")
	}
	initialized := s.dice.Parent != nil
	testMode := false
	if initialized {
		testMode = s.dice.Parent.JustForTest
	}
	data := HealthData{
		Status:      "ok",
		TestMode:    testMode,
		Initialized: initialized,
	}
	return response.NewItemResponse[HealthData](data), nil
}

func (s *BaseService) Overview(_ context.Context, _ *request.Empty) (*response.ItemResponse[OverviewData], error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var versionLatest string
	var versionLatestNote string
	var versionLatestCode int64
	if s.dm.AppVersionOnline != nil {
		versionLatest = s.dm.AppVersionOnline.VersionLatestDetail
		versionLatestNote = s.dm.AppVersionOnline.VersionLatestNote
		versionLatestCode = s.dm.AppVersionOnline.VersionLatestCode
	}

	extraTitle := ""
	if s.dice != nil && s.dice.ImSession != nil {
		func() {
			defer func() {
				_ = recover()
			}()
			ctx := &dice.MsgContext{Dice: s.dice, EndPoint: nil, Session: s.dice.ImSession}
			extraTitle = dice.DiceFormatTmpl(ctx, "核心:骰子名字")
		}()
	}

	data := OverviewData{
		AppName:    dice.APPNAME,
		AppChannel: dice.APP_CHANNEL,
		ExtraTitle: extraTitle,
		Version: VersionInfo{
			Value:  dice.VERSION.String(),
			Simple: dice.VERSION_MAIN + dice.VERSION_PRERELEASE,
			Code:   dice.VERSION_CODE,
			Detail: VersionDetail{
				Major:         dice.VERSION.Major(),
				Minor:         dice.VERSION.Minor(),
				Patch:         dice.VERSION.Patch(),
				Prerelease:    dice.VERSION.Prerelease(),
				BuildMetaData: dice.VERSION.Metadata(),
			},
			Latest:     versionLatest,
			LatestNote: versionLatestNote,
			LatestCode: versionLatestCode,
		},
		Runtime: RuntimeInfo{
			Uptime:        int64(time.Since(startTime).Seconds()),
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
			JustForTest:   s.dm.JustForTest,
			ContainerMode: s.dm.ContainerMode,
		},
		Memory: MemoryInfo{
			Alloc:   m.Alloc,
			Sys:     m.Sys,
			UsedSys: m.Sys - m.HeapReleased,
		},
	}
	return response.NewItemResponse[OverviewData](data), nil
}

// Login 用户登录
func (s *BaseService) Login(_ context.Context, req *LoginReq) (*response.ItemResponse[LoginResponse], error) {
	if s.dm.UIPasswordHash == "" || s.dm.UIPasswordHash == req.Body.Password {
		// 改用一个其他的生成策略简化冗余代码
		head := fmt.Sprintf("%x", time.Now().Unix())
		token, err := nanoid.New(64)
		if err != nil {
			return nil, err
		}
		token += ":" + head
		s.dice.Parent.AccessTokens.Store(token, true)
		s.dice.LastUpdatedTime = time.Now().Unix()
		s.dice.Parent.Save()
		return response.NewItemResponse[LoginResponse](LoginResponse{
			Token: token,
		}), nil
	}
	return nil, huma.Error401Unauthorized("Invalid password")
}

func (s *BaseService) LoginSalt(_ context.Context, _ *request.Empty) (*response.ItemResponse[LoginSaltResponse], error) {
	return response.NewItemResponse[LoginSaltResponse](LoginSaltResponse{
		Salt: s.dm.UIPasswordSalt,
	}), nil
}

// SecurityCheck 安全配置检查
func (s *BaseService) SecurityCheck(_ context.Context, _ *struct{}) (*response.ItemResponse[bool], error) {
	isPublicService := strings.HasPrefix(s.dm.ServeAddress, "0.0.0.0") || s.dm.ServeAddress == ":3211"
	isEmptyPassword := s.dm.UIPasswordHash == ""
	return response.NewItemResponse[bool](!isEmptyPassword || !isPublicService), nil
}

// 私以为这应该是个WebSocket接口 准备使用Melody进行改造 先不放在这里
// func (s *BaseService) GetDiceLogItems(_ context.Context, req *request.RequestWrapper[]) (*response.ItemResponse[bool], error) {
//}
