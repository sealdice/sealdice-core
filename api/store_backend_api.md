# SealDice 商店接口文档

## 接口基础信息

所有接口都基于基础 URL 构建，通常为 `<baseUrl>` + `/dice/app/store`，如官方 API 使用 `http://sealdice.com/dice/api/store`。

## 接口列表

### 获取商店信息

**接口地址**: `GET /info`

**请求方式**: GET

**接口说明**: 获取商店的基本信息，包括支持的协议版本、公告等。

**响应数据结构**:
```json
{
  "name": "string",               // 商店名称
  "protocolVersions": ["string"], // 支持的协议版本列表，目前仅有 "1.0"
  "announcement": "string",       // 商店公告
  "sign": "string"                // （可选）签名信息，使用相应私钥签名的商店 url
}
```

### 获取推荐扩展

**接口地址**: `GET /recommend`

**请求方式**: GET

**接口说明**: 获取商店推荐的扩展列表。

**响应数据结构**:
```json
{
  "result": true,
  "data": [
    {
      "id": "string",                         // 扩展唯一 ID，格式 <namespace>@<key>@<version>，例如 seal@example@1.0.0
      "namespace": "string",                  // 命名空间
      "key": "string",                        // 扩展键名
      "version": "string",                    // 版本号
      "type": "plugin|deck|reply|helpdoc",    // 扩展类型
      "ext": "string",                        // 扩展名后缀（.js|.json|.toml 等）
      "name": "string",                       // 扩展名称
      "authors": ["string"],                  // 作者列表
      "desc": "string",                       // 描述
      "license": "string",                    // 许可证
      "releaseTime": 0,                       // 发布时间（unix 时间戳）
      "updateTime": 0,                        // 更新时间（unix 时间戳）
      "tags": ["string"],                     // 标签列表
      "rate": 0,                              // 评分（0-5）
      "extra": {"key": "value"},              // 额外信息
      "downloadNum": 0,                       // 下载次数
      "downloadUrl": "string",                // 下载地址，目标扩展的直接下载地址
      "hash": {"sha256": "string"},           // 文件哈希值
      "homePage": "string",                   // 主页地址
      "sealVersion": "string",                // 支持的最低 sealdice 版本
      "dependencies": {"key": "value"}        // 依赖信息
    }
  ],
  "err": "string" // 错误信息（仅在 result 为 false 时存在）
}
```

### 分页获取扩展列表

**接口地址**: `GET /page`

**请求方式**: GET

**请求参数**:
| 参数名      | 类型   | 必填 | 说明 |
| ----------- | ------ | ---- | ---- |
| type        | string | 是   | 扩展类型（plugin/deck/reply/helpdoc） |
| pageNum     | int    | 是   | 页码（从 1 开始） |
| pageSize    | int    | 是   | 每页数量 |
| author      | string | 否   | 作者 |
| name        | string | 否   | 扩展名称 |
| sortBy      | string | 否   | 排序（updateTime/downloadNum） |
| order       | string | 否   | 排序方式（asc/desc） |

例如：`/page?type=plugin&pageNum=1&pageSize=20`

**响应数据结构**:
```json
{
  "result": true,
  "data": {
    "data": [
      // 扩展数组，结构同 /recommend 接口
    ],
    "pageNum": 0,    // 当前页码
    "pageSize": 0,   // 每页数量
    "next": true     // 是否有下一页
  },
  "err": "string" // 错误信息（仅在 result 为 false 时存在）
}
```

## 错误处理

所有接口在出错时会返回相应的 HTTP 状态码和错误信息：
- HTTP 200: 请求成功
- 其他状态码：请求失败，错误信息在响应体中

对于返回 JSON 格式的接口，使用 `result` 字段表示请求是否成功：
- `result: true`: 请求成功，数据在 `data` 字段中
- `result: false`: 请求失败，错误信息在 `err` 字段中