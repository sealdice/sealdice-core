# dynamic_form

简洁的 JSON 驱动动态表单库，面向“添加连接”等场景。前端只需按定义渲染与提交，后端统一校验并转换为业务参数。

- 下发：GET /sd-api/v2/imconnection/forms/{key}
- 提交：POST /sd-api/v2/imconnection/create/{key}，请求体统一 items

## 特性

- 支持文本、数字、时间、范围时间、单选（radio）、复选（checkbox）、下拉（dropdown）
- 支持是否必填、placeholder、默认值 default 与范围默认值 default_range
- 选项采用 {label, value} 格式，支持静态与动态（API）来源
- 简单统一的提交格式，后端负责类型转换与校验

## 目录

- [types.go](file:///d:/sealdice-core/api/v2/imconnection/dynamic_form/types.go)：数据结构与枚举
- [loader.go](file:///d:/sealdice-core/api/v2/imconnection/dynamic_form/loader.go)：加载、校验、转换与动态选项
- [forms.json](file:///d:/sealdice-core/api/v2/imconnection/dynamic_form/forms.json)：示例表单定义
- [loader_test.go](file:///d:/sealdice-core/api/v2/imconnection/dynamic_form/loader_test.go)：单元测试

## 快速开始

```go
import dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"

_ = dynamicform.LoadFromFile("api/v2/imconnection/dynamic_form/forms.json")
keys := dynamicform.Keys()
items := dynamicform.GetFormConfig("discord")

params, err := dynamicform.BuildParamsBySubmit(items, submitItems)
```

提交体统一为：

```json
{
  "items": [
    { "id": 1, "data": "xxxx" },
    { "id": 2, "data": "3100" }
  ]
}
```

## 数据模型

- 表单项
  - id：数字
  - name：展示名
  - field_name：转换后参数名
  - input_type：枚举（Text、Num、Date、DateRange、Sin、Mul、Select、Bool）
  - is_required：是否必填（1/0）
  - placeholder：占位展示
  - default：默认字符串值（适用于 Text/Num/Date/Sin/Mul/Select/Bool）
  - default_range：范围默认值（适用于 DateRange，结构为 {start,end}）
  - sub_option：静态选项 [{label,value}]
  - options_url：动态选项来源（交由应用提供）

## 选项来源

- 静态：在 JSON 中直接提供 sub_option
- 动态：在表单项设置 options_url，并在后端注册选项提供者

```go
dynamicform.RegisterOptionsProvider(func(item *dynamicform.FormConfigItem) ([]*dynamicform.Option, error) {
    if item.OptionsURL == "mock://colors" {
        return []*dynamicform.Option{
            {Label: "Red", Value: "red"},
            {Label: "Blue", Value: "blue"},
        }, nil
    }
    return nil, nil
})
```

当存在选项来源时：
- Select/Radio 值必须为某个 option.value
- Checkbox 值必须是选项集合（提交为 JSON 数组，支持字符串或数字数组）

## 时间与范围

- 时间：提交为数字字符串（如 Unix 秒），后端转换为 int64
- 范围时间：提交为 JSON 对象 {"start": 1, "end": 2}；当 data 为空且 default_range 存在时，将使用默认范围

## Huma 接入建议

- 路径采用枚举 key：GET /sd-api/v2/imconnection/forms/{key}，POST /sd-api/v2/imconnection/create/{key}
- POST 仅描述统一 items 格式，避免将具体业务字段硬编码到 OpenAPI
- 校验失败直接返回 400 与简洁错误消息，如 "missing params"、"convert error"、"invalid option"

## 示例 JSON 片段

```json
{
  "satori": [
    { "id": 1, "name": "platform", "input_type": 0, "is_required": 1, "field_name": "platform", "default": "", "placeholder": "QQ/Telegram" },
    { "id": 3, "name": "port", "input_type": 1, "is_required": 1, "field_name": "port", "default": "3100", "placeholder": "3100" }
  ]
}
```

## 测试

运行：

```bash
go test ./api/v2/imconnection/dynamic_form -v
```

覆盖点：
- 加载/下发、必填与默认、数值转换
- 单选/复选/下拉的静态与动态选项
- 时间与范围时间校验与转换
