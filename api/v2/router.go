package v2

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/api/v2/backup"
	"sealdice-core/api/v2/ban"
	"sealdice-core/api/v2/base"
	"sealdice-core/api/v2/basesetting"
	"sealdice-core/api/v2/censor"
	"sealdice-core/api/v2/config"
	"sealdice-core/api/v2/customreply"
	"sealdice-core/api/v2/customtext"
	"sealdice-core/api/v2/deck"
	"sealdice-core/api/v2/extension"
	"sealdice-core/api/v2/group"
	"sealdice-core/api/v2/helpdoc"
	"sealdice-core/api/v2/imconnection"
	"sealdice-core/api/v2/js"
	"sealdice-core/api/v2/middleware"
	"sealdice-core/api/v2/pprof"
	"sealdice-core/api/v2/realtime"
	"sealdice-core/api/v2/resource"
	"sealdice-core/api/v2/story"
	"sealdice-core/api/v2/tooltest"
	"sealdice-core/dice"
)

// InitV2Router registers HTTP handlers without starting background workers.
// The caller owns the returned realtime server's lifetime.
func InitV2Router(api huma.API, dm *dice.DiceManager) *realtime.Server {
	auth := middleware.AuthMiddleware(api, dm.GetDice())
	write := middleware.WriteProtectedMiddleware(api, dm.GetDice())
	groupFor := func(name string, guards ...func(huma.Context, func(huma.Context))) *huma.Group {
		grp := huma.NewGroup(api, "/sd-api/v2/"+name)
		grp.UseSimpleModifier(huma.OperationTags(name))
		grp.UseMiddleware(guards...)
		return grp
	}

	// Login/bootstrap is public; network diagnostics only require authentication.
	baseService := base.NewBaseService(dm)
	baseService.RegisterRoutes(groupFor("base"))
	baseService.RegisterProtectedRoutes(groupFor("base", auth))

	realtimeServer := realtime.RegisterRoutes(groupFor("realtime", auth), dm)
	toolTestService := tooltest.NewService(dm)
	toolTestService.SetPublisher(realtimeServer)
	realtimeServer.SetToolMessageProvider(toolTestService)

	// Modules classify their operations explicitly. HTTP method alone does not
	// determine write protection (e.g. POST connection tests and store queries).
	for _, module := range []struct {
		name    string
		service interface {
			RegisterRoutes(grp *huma.Group)
			RegisterProtectedRoutes(grp *huma.Group)
		}
	}{
		{"base-setting", basesetting.NewService(dm)},
		{"group", group.NewGroupService(dm)},
		{"backup", backup.NewBackupService(dm)},
		{"ban", ban.NewBanService(dm)},
		{"custom-text", customtext.NewService(dm)},
		{"config", config.NewService(dm)},
		{"custom-reply", customreply.NewService(dm)},
		{"deck", deck.NewService(dm)},
		{"story", story.NewService(dm)},
		{"js", js.NewService(dm)},
		{"helpdoc", helpdoc.NewService(dm)},
		{"censor", censor.NewService(dm)},
		{"imconnection", imconnection.NewService(dm)},
		{"tool-test", toolTestService},
		{"resource", resource.NewService(dm)},
		{"extension", extension.NewService(dm)},
	} {
		module.service.RegisterRoutes(groupFor(module.name, auth))
		module.service.RegisterProtectedRoutes(groupFor(module.name, write))
	}

	pprof.NewService().RegisterRoutes(groupFor("pprof", auth))
	return realtimeServer
}
