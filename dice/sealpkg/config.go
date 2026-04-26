package sealpkg

import (
	"errors"
	"fmt"
)

// ValidateConfigValue 验证单个配置值
func ValidateConfigValue(key string, value interface{}, schema ConfigSchema) error {
	// 类型检查
	switch schema.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("配置项 " + key + " 应为字符串类型")
		}

	case "integer":
		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				return errors.New("配置项 " + key + " 应为整数类型")
			}
			if schema.Min != nil && v < *schema.Min {
				return fmt.Errorf("配置项 %s 值 %v 小于最小值 %v", key, v, *schema.Min)
			}
			if schema.Max != nil && v > *schema.Max {
				return fmt.Errorf("配置项 %s 值 %v 大于最大值 %v", key, v, *schema.Max)
			}
		case int, int64, int32:
			// OK
		default:
			return errors.New("配置项 " + key + " 应为整数类型")
		}

	case "number":
		v, ok := value.(float64)
		if !ok {
			return errors.New("配置项 " + key + " 应为数字类型")
		}
		if schema.Min != nil && v < *schema.Min {
			return fmt.Errorf("配置项 %s 值 %v 小于最小值 %v", key, v, *schema.Min)
		}
		if schema.Max != nil && v > *schema.Max {
			return fmt.Errorf("配置项 %s 值 %v 大于最大值 %v", key, v, *schema.Max)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("配置项 " + key + " 应为布尔类型")
		}

	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			return errors.New("配置项 " + key + " 应为数组类型")
		}
		// 验证数组元素
		if schema.Items != nil {
			for i, item := range arr {
				if err := ValidateConfigValue(fmt.Sprintf("%s[%d]", key, i), item, *schema.Items); err != nil {
					return err
				}
			}
		}

	case "object":
		obj, ok := value.(map[string]interface{})
		if !ok {
			return errors.New("配置项 " + key + " 应为对象类型")
		}
		// 验证对象属性
		if schema.Properties != nil {
			for propKey, propSchema := range schema.Properties {
				if propValue, exists := obj[propKey]; exists {
					if err := ValidateConfigValue(key+"."+propKey, propValue, propSchema); err != nil {
						return err
					}
				}
			}
		}
	}

	// 枚举检查
	if len(schema.Enum) > 0 {
		found := false
		for _, enumVal := range schema.Enum {
			if value == enumVal {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("配置项 %s 的值 %v 不在允许的枚举范围内", key, value)
		}
	}

	return nil
}

// ValidateConfig 验证整个配置对象
func ValidateConfig(config map[string]interface{}, schemas map[string]ConfigSchema) error {
	for key, value := range config {
		schema, exists := schemas[key]
		if !exists {
			continue // 忽略未知配置项
		}
		if err := ValidateConfigValue(key, value, schema); err != nil {
			return err
		}
	}
	return nil
}

// InitDefaultConfig 根据 schema 初始化默认配置
func InitDefaultConfig(schemas map[string]ConfigSchema) map[string]interface{} {
	config := make(map[string]interface{})
	for key, schema := range schemas {
		if schema.Default != nil {
			config[key] = schema.Default
		}
	}
	return config
}

// MergeConfig 合并配置（新配置覆盖旧配置）
func MergeConfig(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 复制基础配置
	for k, v := range base {
		result[k] = v
	}

	// 覆盖
	for k, v := range override {
		result[k] = v
	}

	return result
}

// GetConfigWithDefaults 获取配置，缺失的键使用默认值填充
func GetConfigWithDefaults(config map[string]interface{}, schemas map[string]ConfigSchema) map[string]interface{} {
	result := make(map[string]interface{})

	// 先设置默认值
	for key, schema := range schemas {
		if schema.Default != nil {
			result[key] = schema.Default
		}
	}

	// 再用实际配置覆盖
	for k, v := range config {
		result[k] = v
	}

	return result
}
