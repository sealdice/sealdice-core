package v2

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v2"

	"sealdice-core/dice"
	"sealdice-core/logger"
)

// BuildOpenAPI registers the v2 router on a throwaway Fiber instance and returns
// the generated OpenAPI document. The minimal DiceManager is intentionally not
// loaded from disk, so spec generation stays side-effect free.
func BuildOpenAPI() *huma.OpenAPI {
	app := fiber.New()
	api := humafiber.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	InitV2Router(api, app, newOpenAPIDiceManager())
	return api.OpenAPI()
}

func WriteOpenAPI(path string) error {
	spec := BuildOpenAPI()
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func newOpenAPIDiceManager() *dice.DiceManager {
	d := &dice.Dice{
		ImSession: &dice.IMSession{
			EndPoints:    []*dice.EndPointInfo{},
			ServiceAtNew: &dice.SyncMap[string, *dice.GroupInfo]{},
			PendingQuits: &dice.SyncMap[string, *dice.PendingQuitInfo]{},
		},
		Logger: logger.M(),
	}
	d.ImSession.Parent = d
	dm := &dice.DiceManager{
		Dice:         []*dice.Dice{d},
		ServeAddress: "127.0.0.1:3211",
	}
	d.Parent = dm
	return dm
}
