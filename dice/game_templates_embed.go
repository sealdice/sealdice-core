package dice

import (
	"embed"
	"fmt"
	"path"
)

//go:embed templates/*.yaml
var gameTemplateFS embed.FS

func loadBuiltinTemplate(name string) (*GameSystemTemplate, error) {
	data, err := gameTemplateFS.ReadFile(path.Join("templates", name))
	if err != nil {
		return nil, fmt.Errorf("failed to load builtin template %s: %w", name, err)
	}
	return loadGameSystemTemplateFromData(data, path.Ext(name))
}
