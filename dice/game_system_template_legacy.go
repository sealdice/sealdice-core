package dice

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// legacyNameTemplateItem mirrors the v1 sn template entry.
type legacyNameTemplateItem struct {
	Template string `json:"template" yaml:"template"`
	HelpText string `json:"helpText" yaml:"helpText"`
}

// legacyAttrConfig mirrors the v1 attrConfig block.
type legacyAttrConfig struct {
	Top          []string          `json:"top"          yaml:"top,flow"`
	SortBy       string            `json:"sortBy"       yaml:"sortBy"`
	Ignores      []string          `json:"ignores"      yaml:"ignores"`
	ShowAs       map[string]string `json:"showAs"       yaml:"showAs"`
	ShowAsKey    map[string]string `json:"showAsKey"    yaml:"showAsKey"`
	ItemsPerLine int               `json:"itemsPerLine" yaml:"itemsPerLine"`
}

// legacySetConfig mirrors the v1 setConfig block.
type legacySetConfig struct {
	RelatedExt    []string `json:"relatedExt"    yaml:"relatedExt"`
	DiceSides     int      `json:"diceSides"     yaml:"diceSides"`
	DiceSidesExpr string   `json:"diceSidesExpr" yaml:"diceSidesExpr"`
	Keys          []string `json:"keys"          yaml:"keys"`
	EnableTip     string   `json:"enableTip"     yaml:"enableTip"`
}

// legacyTemplate captures the v1 template layout.
type legacyTemplate struct {
	Name             string                            `json:"name"             yaml:"name"`
	FullName         string                            `json:"fullName"         yaml:"fullName"`
	Authors          []string                          `json:"authors"          yaml:"authors"`
	Version          string                            `json:"version"          yaml:"version"`
	UpdatedTime      string                            `json:"updatedTime"      yaml:"updatedTime"`
	TemplateVer      string                            `json:"templateVer"      yaml:"templateVer"`
	NameTemplate     map[string]legacyNameTemplateItem `json:"nameTemplate"     yaml:"nameTemplate"`
	AttrConfig       legacyAttrConfig                  `json:"attrConfig"       yaml:"attrConfig"`
	SetConfig        legacySetConfig                   `json:"setConfig"        yaml:"setConfig"`
	Defaults         map[string]int                    `json:"defaults"         yaml:"defaults"`
	DefaultsComputed map[string]string                 `json:"defaultsComputed" yaml:"defaultsComputed"`
	DetailOverwrite  map[string]string                 `json:"detailOverwrite"  yaml:"detailOverwrite"`
	Alias            map[string][]string               `json:"alias"            yaml:"alias"`
	TextMap          map[string]any                    `json:"textMap"          yaml:"textMap"`
	TextMapHelpInfo  map[string]any                    `json:"textMapHelpInfo"  yaml:"TextMapHelpInfo"`
	PreloadCode      string                            `json:"preloadCode"      yaml:"preloadCode"`
}

func parseLegacyTemplate(data []byte, format string) (*legacyTemplate, error) {
	var tmpl legacyTemplate
	var err error
	switch strings.ToLower(format) {
	case "json":
		err = json.Unmarshal(data, &tmpl)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &tmpl)
	default:
		err = fmt.Errorf("unsupported legacy template format: %s", format)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse legacy template: %w", err)
	}
	return &tmpl, nil
}

func convertLegacyTemplate(data []byte, format string) (*GameSystemTemplate, error) {
	legacy, err := parseLegacyTemplate(data, format)
	if err != nil {
		return nil, err
	}
	dst := &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Name:        legacy.Name,
			FullName:    legacy.FullName,
			Authors:     cloneStrings(legacy.Authors),
			Version:     legacy.Version,
			UpdatedTime: legacy.UpdatedTime,
			TemplateVer: legacy.TemplateVer,
			InitScript:  legacy.PreloadCode,
			Attrs: Attrs{
				Defaults:         cloneIntMap(legacy.Defaults),
				DefaultsComputed: cloneStringMap(legacy.DefaultsComputed),
				DetailOverwrite:  cloneDetailOverwrite(legacy.DetailOverwrite),
			},
			Alias: cloneAlias(legacy.Alias),
			Commands: Commands{
				Set: SetConfig{
					EnableTip:  legacy.SetConfig.EnableTip,
					Keys:       cloneStrings(legacy.SetConfig.Keys),
					RelatedExt: cloneStrings(legacy.SetConfig.RelatedExt),
				},
				Sn: make(SnConfig, len(legacy.NameTemplate)),
				St: StConfig{
					Show: StShowConfig{
						Top:          cloneStrings(legacy.AttrConfig.Top),
						SortBy:       legacy.AttrConfig.SortBy,
						Ignores:      cloneStrings(legacy.AttrConfig.Ignores),
						ShowValueAs:  cloneStringMap(legacy.AttrConfig.ShowAs),
						ShowKeyAs:    cloneStringMap(legacy.AttrConfig.ShowAsKey),
						ItemsPerLine: legacy.AttrConfig.ItemsPerLine,
					},
				},
			},
		},
	}
	if dst.TemplateVer == "" {
		dst.TemplateVer = "2.0"
	}

	assignLegacyDiceSideExpr(&dst.Commands.Set, legacy.SetConfig)
	for key, tmpl := range legacy.NameTemplate {
		dst.Commands.Sn[key] = SnTemplate(tmpl)
	}

	if len(dst.Commands.Sn) == 0 {
		dst.Commands.Sn = nil
	}

	textMap, err := convertLegacyTextMap(legacy.TextMap)
	if err != nil {
		return nil, err
	}
	if textMap != nil {
		dst.TextMap = textMap
	}

	helpInfo, err := convertLegacyTextMapHelp(legacy.TextMapHelpInfo)
	if err != nil {
		return nil, err
	}
	if helpInfo != nil {
		dst.TextMapHelpInfo = helpInfo
	}

	dst.Init()
	return dst, nil
}

func assignLegacyDiceSideExpr(dst *SetConfig, legacy legacySetConfig) {
	if dst == nil {
		return
	}
	if expr := strings.TrimSpace(legacy.DiceSidesExpr); expr != "" {
		dst.DiceSideExpr = expr
		return
	}
	if legacy.DiceSides != 0 {
		dst.DiceSideExpr = strconv.Itoa(legacy.DiceSides)
	}
}

func convertLegacyTextMap(raw map[string]any) (*TextTemplateWithWeightDict, error) {
	if len(raw) == 0 {
		return nil, nil //nolint:nilnil
	}
	buf, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var typed TextTemplateWithWeightDict
	if err := yaml.Unmarshal(buf, &typed); err != nil {
		return nil, err
	}
	return &typed, nil
}

func convertLegacyTextMapHelp(raw map[string]any) (*TextTemplateWithHelpDict, error) {
	if len(raw) == 0 {
		return nil, nil //nolint:nilnil
	}
	buf, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var typed TextTemplateWithHelpDict
	if err := yaml.Unmarshal(buf, &typed); err != nil {
		return nil, err
	}
	return &typed, nil
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneAlias(src map[string][]string) Alias {
	if len(src) == 0 {
		return nil
	}
	dst := make(Alias, len(src))
	for k, v := range src {
		dst[k] = cloneStrings(v)
	}
	return dst
}

func cloneDetailOverwrite(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
