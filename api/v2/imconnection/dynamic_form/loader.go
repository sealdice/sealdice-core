package dynamicform

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
)

var forms map[string][]*FormConfigItem

var optionsProvider func(item *FormConfigItem) ([]*Option, error)

func RegisterOptionsProvider(p func(item *FormConfigItem) ([]*Option, error)) {
	optionsProvider = p
}

// LoadFromFile 从 JSON 文件加载所有表单定义
func LoadFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	m := make(map[string][]*FormConfigItem)
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	forms = m
	return nil
}

// GetFormConfig 获取指定 key 的表单项列表
func GetFormConfig(key string) []*FormConfigItem {
	return forms[key]
}

// Keys 返回当前已加载的所有平台 key（按字典序）
func Keys() []string {
	ks := make([]string, 0, len(forms))
	for k := range forms {
		ks = append(ks, k)
	}
	if len(ks) > 1 {
		// 简单稳定排序
		for i := 0; i < len(ks)-1; i++ {
			for j := i + 1; j < len(ks); j++ {
				if ks[i] > ks[j] {
					ks[i], ks[j] = ks[j], ks[i]
				}
			}
		}
	}
	return ks
}

// GetAll 返回完整的 key→items 映射（只读副本）
func GetAll() map[string][]*FormConfigItem {
	out := make(map[string][]*FormConfigItem, len(forms))
	for k, v := range forms {
		out[k] = v
	}
	return out
}

func effectiveOptions(item *FormConfigItem) []*Option {
	if item == nil {
		return nil
	}
	if len(item.SubOption) > 0 {
		return item.SubOption
	}
	if item.OptionsURL != "" && optionsProvider != nil {
		opts, err := optionsProvider(item)
		if err == nil && len(opts) > 0 {
			return opts
		}
	}
	return nil
}

func hasOptionValue(opts []*Option, v string) bool {
	for _, op := range opts {
		if op != nil && op.Value == v {
			return true
		}
	}
	return false
}

func parseStringArrayOrIntArray(data string) ([]string, error) {
	arrStr := make([]string, 0)
	if err := json.Unmarshal([]byte(data), &arrStr); err == nil {
		return arrStr, nil
	}
	arrInt := make([]int, 0)
	if err := json.Unmarshal([]byte(data), &arrInt); err == nil {
		out := make([]string, 0, len(arrInt))
		for _, n := range arrInt {
			out = append(out, strconv.Itoa(n))
		}
		return out, nil
	}
	return nil, errors.New("params error")
}

// CheckSubmitForms 校验提交数据是否满足表单项定义与校验规则
func CheckSubmitForms(formConfigItems []*FormConfigItem, submitFormIDDataMap map[uint64]string) error {
	for _, form := range formConfigItems {
		data, ok := submitFormIDDataMap[form.ID]
		dataLen := len(data)
		if !ok {
			if form.IsRequired == RequiredTrue {
				if form.InputType == InputTypeDateRange && form.DefaultRange != nil {
					continue
				}
				if len(form.Default) > 0 {
					continue
				}
				return errors.New("missing params")
			}
			continue
		}
		if form.IsRequired == RequiredTrue && dataLen < 1 {
			if form.InputType == InputTypeDateRange && form.DefaultRange != nil {
				continue
			}
			if len(form.Default) > 0 {
				continue
			}
			return errors.New("missing params")
		}
		if dataLen < 1 {
			continue
		}
		switch form.InputType {
		case InputTypeText, InputTypeNum:
			if err := checkSubmitType(data, form.CheckType); err != nil {
				return err
			}
		case InputTypeDate:
			if err := checkSubmitType(data, CheckTypeNum); err != nil {
				return err
			}
		case InputTypeDateRange:
			r := &RangeValue{}
			if err := json.Unmarshal([]byte(data), r); err != nil || r.Start == 0 && r.End == 0 {
				return errors.New("params error")
			}
		case InputTypeSin:
			opts := effectiveOptions(form)
			if len(opts) == 0 {
				if err := checkSubmitType(data, CheckTypeNum); err != nil {
					return err
				}
			} else {
				if !hasOptionValue(opts, data) {
					return errors.New("invalid option")
				}
			}
		case InputTypeMul:
			opts := effectiveOptions(form)
			if len(opts) == 0 {
				ids := make([]int, 0)
				err := json.Unmarshal([]byte(data), &ids)
				if err != nil {
					return errors.New("params error")
				}
			} else {
				values, err := parseStringArrayOrIntArray(data)
				if err != nil {
					return err
				}
				for _, v := range values {
					if !hasOptionValue(opts, v) {
						return errors.New("invalid option")
					}
				}
			}
		case InputTypeSelect:
			opts := effectiveOptions(form)
			if len(opts) > 0 && !hasOptionValue(opts, data) {
				return errors.New("invalid option")
			}
		case InputTypeBool:
			if data != "true" && data != "false" && data != "1" && data != "0" {
				return errors.New("need boolean")
			}
		}
	}
	return nil
}

// BuildParamsBySubmit 将提交的 items 转换为标准参数 map
func BuildParamsBySubmit(forms []*FormConfigItem, submit SubmitFormItems) (map[string]interface{}, error) {
	idx := map[uint64]*FormConfigItem{}
	for _, it := range forms {
		if it != nil {
			idx[it.ID] = it
		}
	}
	dataMap := map[uint64]string{}
	for _, s := range submit {
		dataMap[s.ID] = s.Data
	}
	if err := CheckSubmitForms(forms, dataMap); err != nil {
		return nil, err
	}
	params := map[string]interface{}{}
	for id, it := range idx {
		v := dataMap[id]
		if len(v) == 0 {
			switch it.InputType {
			case InputTypeDateRange:
				if it.DefaultRange != nil {
					params[it.FieldName] = it.DefaultRange
					continue
				}
			default:
				if len(it.Default) > 0 {
					v = it.Default
				}
			}
			if len(v) == 0 && it.IsRequired != RequiredTrue {
				continue
			}
		}
		switch it.InputType {
		case InputTypeText:
			params[it.FieldName] = v
		case InputTypeNum:
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, errors.New("convert error")
			}
			params[it.FieldName] = n
		case InputTypeDate:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, errors.New("convert error")
			}
			params[it.FieldName] = n
		case InputTypeDateRange:
			r := &RangeValue{}
			if err := json.Unmarshal([]byte(v), r); err != nil {
				return nil, errors.New("convert error")
			}
			params[it.FieldName] = r
		case InputTypeSin:
			opts := effectiveOptions(it)
			if len(opts) == 0 {
				n, err := strconv.Atoi(v)
				if err != nil {
					return nil, errors.New("convert error")
				}
				params[it.FieldName] = n
			} else {
				if !hasOptionValue(opts, v) {
					return nil, errors.New("convert error")
				}
				params[it.FieldName] = v
			}
		case InputTypeMul:
			opts := effectiveOptions(it)
			if len(opts) == 0 {
				arr := make([]int, 0)
				err := json.Unmarshal([]byte(v), &arr)
				if err != nil {
					return nil, errors.New("convert error")
				}
				params[it.FieldName] = arr
			} else {
				values, err := parseStringArrayOrIntArray(v)
				if err != nil {
					return nil, errors.New("convert error")
				}
				for _, val := range values {
					if !hasOptionValue(opts, val) {
						return nil, errors.New("convert error")
					}
				}
				params[it.FieldName] = values
			}
		case InputTypeSelect:
			// 直接传字符串值
			params[it.FieldName] = v
		case InputTypeBool:
			vv := strings.ToLower(v)
			if vv == "true" || v == "1" {
				params[it.FieldName] = true
			} else if vv == "false" || v == "0" {
				params[it.FieldName] = false
			} else {
				return nil, errors.New("convert error")
			}
		}
	}
	return params, nil
}

// checkSubmitType 按校验类型检查 data
func checkSubmitType(data string, checkType int) error {
	switch checkType {
	case CheckTypeNull:
		return nil
	case CheckTypeNum:
		_, err := strconv.Atoi(data)
		if err != nil {
			return errors.New("need pure numbers")
		}
	default:
		return nil
	}
	return nil
}
