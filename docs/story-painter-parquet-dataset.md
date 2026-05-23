# StoryPainter 嵌入版 Parquet 数据源方案

## 背景

StoryPainter 原先面向完整文本或完整 JSON 工作，嵌入到控制台后第一版也沿用了类似模型：后端导出 Parquet，前端下载后立即调用 `parquetReadObjects`，再把所有行 `map` 成 `StoryPainterLogItem[]`。这个方案能跑通功能，但破坏了 Parquet 的价值：压缩和列式存储在下载后立刻被摊平成全量通用对象，内存占用随日志行数和消息长度线性膨胀。

本版调整的重点不是减少 Parquet 文件下载。Parquet 压缩效果好，完整下载通常不是主要瓶颈。真正要避免的是“下载后立即全量解码、全量格式化、全量挂到响应式状态”。

## 新方案

- 前端新增 `StoryPainterParquetDataset`，持有 Parquet `Blob` 和 metadata，提供 `readRows(start, end, columns)`、`iterRows({ chunkSize, columns })`、`readAll(columns)`。
- 初始加载只读取 metadata 和轻量列，构建角色列表和可见行索引；消息正文不进入 `items` 全量响应式数组。
- 预览使用可见行索引和小窗口缓存，按需读取具体行内容；虚拟列表只渲染可见区域。
- 文本类导出和复制按 chunk 读取并格式化，不再依赖全量 `previewItems`；复制最终仍需要生成完整文本字符串，这是剪贴板 API 的边界，但不需要先保留完整日志对象数组。
- 原始 Parquet 导出直接保存已下载的压缩 blob。
- 后端 `GetLogParquetBytes` 改为游标分页直写 `GenericWriter`，设置 row group 行数，去掉额外 `GenericBuffer` 全量排序缓存。

## 与上一版方案的区别

上一版：

- `fetch blob -> parquetReadObjects -> map -> StoryPainterLogItem[]`
- `items` 和 `previewItems` 都可能保存完整日志对象。
- 预览、筛选、导出共享同一份全量对象数组。
- 后端 Parquet API 先把所有行写入 `GenericBuffer`，再复制到 writer。

本版：

- `fetch blob -> metadata -> dataset`
- 初始只保留索引/角色所需轻量字段。
- 预览窗口和导出按需读取 Parquet 行/列。
- `previewItems` 不再是 Parquet 模式下的主数据源，可见行由 `visibleIndexes` 表示。
- 后端按数据库游标顺序直接写 Parquet row group，避免额外全量 row buffer。

## 编辑与角色变更

编辑器模式仍然是完整文本编辑器，因此只有用户点击载入完整文本时才读取全部行数据。载入前，页面保留 Parquet dataset、轻量索引行和角色表。

角色改名和删除不会改写 Parquet blob。本版在前端维护一层会话级覆盖：

- 改名保存“原始 nickname/IMUserId -> 新 nickname”的映射。
- 删除保存原始说话人 key。
- 按需读取、预览刷新、分块导出都会先套用这层覆盖，再进入 StoryPainter 的格式化逻辑。

## UI 对齐原则

UI 行为以原 story-painter 为准，不再重新设计染色器工作流：

- 预览、论坛代码、论坛代码(内容多行)、回声工坊保持互斥切换语义。
- 导出和复制类主操作使用主色按钮层级。
- 左侧选项恢复原有说明文案。
- 角色面板只做控制台侧栏内的响应式适配，不改变角色/颜色/隐藏等语义。
- 论坛代码复制输出按原组件 `textContent` 语义生成纯文本，不输出预览用 HTML。
- 回声工坊模式保留原 StoryPainter 对 `commandInfo` 生成 `<dice>` / `<hitpoint>` 辅助行的行为。

## 限制

- 编辑器模式天然需要完整文本；进入编辑器时会按需读取完整行数据。
- doc/docx 生成库本身需要完整文档结构，最终文档对象仍会占用内存；本版只避免在生成前额外维护一份完整通用日志数组。
- 当前仍完整下载 Parquet blob，没有引入 HTTP Range。后续只有在文件下载本身成为瓶颈时才需要扩展 Range/缓存 URL。
