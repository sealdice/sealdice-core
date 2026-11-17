package dynamicform

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
)

//go:embed forms.json
var formsJSON []byte

// Protocol definitions ship with the binary; they are not runtime configuration.
var forms = func() map[string][]*FormConfigItem {
	var definitions map[string][]*FormConfigItem
	if err := json.Unmarshal(formsJSON, &definitions); err != nil {
		panic(fmt.Errorf("decode embedded connection forms: %w", err))
	}
	return definitions
}()

// GetFormConfig returns the read-only definition for a protocol.
func GetFormConfig(key string) []*FormConfigItem {
	return forms[key]
}

// GetAll copies the index; its definitions remain read-only.
func GetAll() map[string][]*FormConfigItem {
	return maps.Clone(forms)
}

// BuildParamsByConfig validates field-name keyed input and applies defaults.
// Only declared fields reach the adapter.
func BuildParamsByConfig(items []*FormConfigItem, cfg map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{}, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		value := cfg[item.FieldName]
		if value == nil || value == "" {
			if item.Default != "" {
				value = item.Default
			} else if item.IsRequired == RequiredTrue {
				return nil, fmt.Errorf("%s: missing params", item.FieldName)
			} else {
				continue
			}
		}
		parsed, err := parseValue(item, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", item.FieldName, err)
		}
		params[item.FieldName] = parsed
	}
	return params, nil
}

func parseValue(item *FormConfigItem, value interface{}) (interface{}, error) {
	// HTML inputs may supply strings while JSON supplies native scalar values.
	// Never truncate fractional numbers or stringify compound values.
	var data string
	switch v := value.(type) {
	case string:
		data = v
	case int:
		data = strconv.Itoa(v)
	case int64:
		data = strconv.FormatInt(v, 10)
	case uint64:
		data = strconv.FormatUint(v, 10)
	case float64:
		data = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		data = strconv.FormatBool(v)
	default:
		return nil, errors.New("expected a scalar value")
	}

	switch item.InputType {
	case InputTypeText:
		if item.CheckType == CheckTypeNum {
			if _, err := strconv.Atoi(data); err != nil {
				return nil, errors.New("need pure numbers")
			}
		}
		return data, nil
	case InputTypeNum:
		n, err := strconv.Atoi(data)
		if err != nil {
			return nil, errors.New("need an integer")
		}
		if item.Minimum != nil && n < *item.Minimum || item.Maximum != nil && n > *item.Maximum {
			return nil, errors.New("integer out of range")
		}
		return n, nil
	case InputTypeBool:
		switch data {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return nil, errors.New("need boolean")
		}
	case InputTypeSelect:
		for _, option := range item.SubOption {
			if option != nil && option.Value == data {
				return data, nil
			}
		}
		return nil, errors.New("invalid option")
	default:
		return nil, fmt.Errorf("unsupported input type: %d", item.InputType)
	}
}
