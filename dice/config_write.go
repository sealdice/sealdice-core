package dice

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"strings"
)

var ErrInvalidConfig = errors.New("配置内容无效")

// SaveTextCategory 供 V1、V2 共用：校验并持久化成功后才发布运行时文案。
func (d *Dice) SaveTextCategory(category string, data TextTemplateWithWeight) error {
	category = strings.TrimSpace(category)
	if category == "" {
		return fmt.Errorf("%w: 分类不能为空", ErrInvalidConfig)
	}
	normalized := make(TextTemplateWithWeight, len(data))
	for key, items := range data {
		values := make(TextTemplateItemList, 0, len(items))
		for _, item := range items {
			if len(item) != 2 {
				return fmt.Errorf("%w: %s 必须包含文本和权重", ErrInvalidConfig, key)
			}
			text, ok := item[0].(string)
			if !ok {
				return fmt.Errorf("%w: %s 文本必须为字符串", ErrInvalidConfig, key)
			}
			var weight float64
			switch n := item[1].(type) {
			case int:
				weight = float64(n)
			case int64:
				weight = float64(n)
			case float32:
				weight = float64(n)
			case float64:
				weight = n
			default:
				return fmt.Errorf("%w: %s 权重必须为数字", ErrInvalidConfig, key)
			}
			if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 || weight >= float64(math.MaxInt) {
				return fmt.Errorf("%w: %s 权重超出范围", ErrInvalidConfig, key)
			}
			values = append(values, []any{strings.TrimSpace(text), int(weight)})
		}
		normalized[key] = values
	}
	next := maps.Clone(d.TextMapRaw)
	if next == nil {
		next = TextTemplateWithWeightDict{}
	}
	next[category] = normalized
	buf, err := marshalTextTemplate(next)
	if err != nil {
		return err
	}
	if err = saveTextTemplateFile(filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml"), buf); err != nil {
		return err
	}
	d.TextMapRaw = next
	SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
	d.GenerateTextMap()
	for key, item := range d.TextMapRaw[category] {
		TextMapCompatibleCheck(d, category, key, item)
	}
	return nil
}

// UpdateAdvancedConfig 统一高级设置的变更标记与既有管理器持久化规则。
func (d *Dice) UpdateAdvancedConfig(config AdvancedConfig) {
	d.AdvancedConfig = config
	d.MarkModified()
	d.Parent.Save()
}
