# UI 可见页面视觉审查补充报告

审查日期：2026-08-10  
审查范围：`ui/` 中当前导航可达的 25 个用户页面及其直接使用组件  
审查目标：补充排版、布局、视觉层级、桌面端适配和状态呈现证据；不替代既有静态可用性报告

## 1. 结论摘要

本轮在正确后端数据下，以真实浏览器完成了 25 个可见页面、双主题、`1440×1000` 与 `1280×800` 两档桌面视口的视觉审查，共归档 301 张 PNG。未发现阻断级问题；确认 4 项严重、11 项一般、2 项建议问题。

最优先的问题不是单一页面“好不好看”，而是三类跨页面系统性风险：语义色令牌同时承担填充色和文字色，已产生可测量的低对比文字；同级页面没有稳定的标题、工具栏和内容起点；数据密集页面缺少受限工作区，实时日志和多值文案会把整页拉到 4341px、7887px。

当前壳层在两档桌面宽度和双主题下整体稳定，没有观察到全局横向溢出或顶栏互相遮挡。Naive UI 控件的基础状态、深色主题覆盖、备份/基础设置/指令测试等页面的主模式切换也基本清楚。建议保留这些基础，只修正信息密度、层级和长内容策略，不做全站重绘。

## 2. 依据与审查口径

### 2.1 参考依据

- 仓库根目录《政企产品用户界面设计指南》（征求意见稿），重点参考 6.1 色彩、6.2 字体、6.3 分割线、6.4 图标、6.5 布局，以及 7.1、7.2 的组件和导航规则。
- `design-government-enterprise-ui` 技能中关于桌面端政企/B2B 产品的信息层级、数据密度、任务流、状态反馈和无障碍基线。
- 项目现有 Naive UI 组件结构与主题令牌；不以另一套视觉体系替换 Naive UI。
- 现代桌面 Web 口径：以内容驱动的响应规则替代机械套用旧断点；以 WCAG 2.2 AA 作为基础目标，核心正文可继续追求 AAA。

该 PDF 明确标注为“征求意见稿”，不是当前强制性标准。本报告把它作为产品设计参考，并对旧式固定断点、平均字号和绝对化 AAA 要求作现代化解释；未经完整测量，不宣称项目已经或尚未整体符合 WCAG。

### 2.2 运行环境

- 前端：`http://127.0.0.1:5175`
- 后端：`http://127.0.0.1:3211`
- 浏览器：Chrome for Testing 149，由 `agent-browser 0.31.1` 驱动
- 字体：浏览器进程显式使用 `FONTCONFIG_FILE=/etc/fonts/fonts.conf`；`Microsoft YaHei` 已匹配到 `/usr/share/fonts/winfonts/winfonts/msyh.ttc`
- 数据状态：1 个失败账号、2 个群、6 个牌堆、1 个 JS 插件、1 个资源项、32 个备份文件；封禁和敏感词列表在当前后端数据下为空

系统并非“无认证启动”，而是连接了正确后端并使用其正常认证流程。审查只执行页面导航、页签和筛选切换；未点击保存、删除、清理、恢复等会改变后端状态的操作。

### 2.3 截图矩阵

- 25 个页面均有浅色/深色、`1440×1000`/`1280×800` 默认态，共 100 张。
- 每个页面另有一张浅色 `1440px` 内容区完整高度图，共 25 张。
- 44 组一级模式补充双主题、双宽度截图，共 176 张；其中 JS“插件列表”与默认态相同，仍保留为显式模式证据。
- 合计 301 张：`light/1440x1000` 94 张，其他三个目录各 69 张。
- 标准视口截图尺寸全部匹配目录名；完整高度图宽度均为 1440px，最高为“娱乐”文案页 7887px；未发现小于 10KB 的异常空白 PNG。

截图中的底部居中圆形下箭头是开发模式 Vue Devtools 的面板开关，不属于产品界面，不计入发现。

## 3. 严重问题

### S1. 语义色未区分前景色与填充色，浅色和深色主题均有低对比文字

- 证据：[首页浅色](ui-review-assets/2026-08-10/light/1440x1000/home--default.png)、[COC 文案浅色](ui-review-assets/2026-08-10/light/1440x1000/custom-text-coc--default.png)、[COC 文案深色](ui-review-assets/2026-08-10/dark/1280x800/custom-text-coc--default.png)、[关于页深色](ui-review-assets/2026-08-10/dark/1280x800/about--default.png)。源码见 `ui/src/features/theme/themePalette.ts:18`、`ui/src/components/custom-text/CustomTextToolbar.vue:14`、`ui/src/components/about/AboutHero.vue:224`。
- 规则：原稿 6.1.2.1；语义颜色要维持功能一致性，同时满足文字可辨识度。现代化基线按 WCAG 2.2 AA，普通文字对比度至少 `4.5:1`。
- 影响：按当前令牌值计算，`info #0891b2` 在浅色页面、浅色次按钮、深色次按钮上约为 `3.35:1`、`2.77:1`、`3.98:1`；深色“关于”链接 `#1d4ed8` 对 `#182133` 约为 `2.40:1`。说明、按钮和链接容易被弱视用户漏读。
- 最小修改：将 `solid/action` 与 `text/link` 语义令牌拆分；浅色信息前景使用更深色，暗色链接使用更亮色，并覆盖 Naive UI secondary button、Tag、Text 的实际前景令牌。
- 验收：对五类语义色在页面、卡片、Tag、secondary button、链接表面逐项测量，普通文字均不低于 `4.5:1`。上述数值是令牌/截图采样结果，不等同于完整无障碍认证。

### S2. 帮助文档“词条”表格内容侵入相邻“分类”列

- 证据：[浅色 1280](ui-review-assets/2026-08-10/light/1280x800/mod-helpdoc--item.png)、[深色 1280](ui-review-assets/2026-08-10/dark/1280x800/mod-helpdoc--item.png)。源码见 `ui/src/components/helpdoc/HelpdocItemPane.vue:16`、`ui/src/pages/mod/helpdoc.vue:218`、`ui/src/pages/mod/helpdoc.vue:313`。
- 规则：原稿 6.5.3、7.1.11；表格列边界应稳定，内容不能越过单元格破坏列关系。Naive UI DataTable 应通过明确列宽、ellipsis 或受控横向滚动处理长内容。
- 影响：1280px 下“内容”预览覆盖或贴入“分类”列，用户无法可靠判断单元格归属，属于直接的数据读取错误风险。
- 最小修改：让 `.help-content-preview` 使用 `display:block; width:100%; max-width:100%`，并在 DataTable 列定义中设置可预测宽度和 ellipsis；完整内容保留 Tooltip/展开查看。
- 验收：在 1280/1440、双主题下用长中文、英文连续串和多行内容测试；各列不重叠，横向滚动只在表格内部发生，完整内容仍可访问。

### S3. 首页实时日志没有高度边界，最多 500 行会持续拉长整页

- 证据：[首页完整高度图](ui-review-assets/2026-08-10/light/1440x1000/home--default-full.png) 已达到 `1440×4341`。源码见 `ui/src/pages/index.vue:139`、`ui/src/features/base/logStreamStore.ts:28`。
- 规则：原稿 6.5.3、7.1.11；数据密集任务区应保持关键上下文和表头可见，长列表使用受限高度、分页或虚拟滚动。
- 影响：查看旧日志后，实时状态、排序和刷新控制均离开视野；达到 500 行时页面可能增长到数万像素，定位最新消息和返回顶部的成本很高。
- 最小修改：日志区域改为视口相关 `max-height` 的独立工作区，启用固定表头和 virtual scroll；控制区保持在表格上方可见，并保留正序/倒序定位语义。
- 验收：加载 500 行长短混合日志，页面总高度不随行数增长；两种排序均能定位最新日志，滚动时表头和必要控制持续可见。

### S4. “娱乐”多值文案沿用普通双列文本域，形成近 8000px 的失衡编辑页

- 证据：[完整高度图](ui-review-assets/2026-08-10/light/1440x1000/custom-text-entertainment--default-full.png) 为 `1440×7887`；[深色未修改筛选](ui-review-assets/2026-08-10/dark/1440x1000/custom-text-entertainment--unmodified.png) 可见一列连续文本域、另一列大面积空白。源码见 `ui/src/components/custom-text/CustomTextEditor.vue:29`、`ui/src/components/custom-text/CustomTextEntryCard.vue:62`。
- 规则：原稿 6.5.3 的对齐与接近分组；数据密集工具应可扫描，不能让可变高度内容破坏栅格节律。
- 影响：用户需要滚动多个屏幕才能越过一个回复集合；双列网格由最高项决定行高，另一列留下无意义空白，后续分组和顶部筛选难以到达。
- 最小修改：按 `items.length` 分流，多值条目跨满整行并使用紧凑列表；默认展示前若干项并明确展开，文本域采用 `minRows:1`、`maxRows:4`；顶部筛选/统计可在内容区内吸顶。
- 验收：使用当前“娱乐”数据验证首屏能看到集合数量和后续分组入口；展开后焦点顺序连续，页面高度显著低于 7887px，顶部筛选不因滚动丢失。

## 4. 一般问题

### G1. 同级页面的标题、面包屑和内容起点没有统一层级

- 证据：首页以 `h4` 开始，账号页重复“账号设置”，自定义文案没有内容主标题，高级设置使用 `h2`，群组页以卡片标题充当页标题，资源页使用 38px 营销式标题。代表截图：[高级设置](ui-review-assets/2026-08-10/light/1440x1000/misc-advanced-setting--default.png)、[群组设置](ui-review-assets/2026-08-10/light/1440x1000/misc-group--default.png)、[资源管理](ui-review-assets/2026-08-10/light/1440x1000/tool-resource--default.png)。源码见 `ui/src/components/app-shell/AppBreadcrumb.vue:2`、`ui/src/pages/index.vue:17`、`ui/src/pages/connect.vue:3`、`ui/src/pages/tool/resource.vue:286`。
- 影响：跨页面工作时会遇到重复标题、无标题、卡片标题和营销级标题四种内容起点；资源页的大圆角、渐变和展示型统计也与安静的运维工具语境不一致。
- 最小修改：建立共享 `PageHeader` 规格，稳定主标题、可选说明、页级动作和下方间距；面包屑只表达位置。资源页保留资源状态，但压缩为同一工作页标题和统计条，不要求“关于”页完全去品牌化。
- 验收：25 页各有且仅有一个明确主标题；标题字号、起始线和上下间距一致，面包屑不重复承担标题职责，关于页可作为品牌型例外记录。

### G2. 牌堆和 JS 插件列表使用过多展开卡片，数据密度和层级偏低

- 证据：[牌堆默认页](ui-review-assets/2026-08-10/light/1280x800/mod-deck--default.png) 的完整高度为 2149px；[JS 插件列表](ui-review-assets/2026-08-10/dark/1280x800/mod-js--default.png) 存在外层容器加内层卡片。源码见 `ui/src/pages/mod/deck.vue:13`、`ui/src/pages/mod/deck.vue:120`、`ui/src/components/common/FoldableCard.vue:176`、`ui/src/components/js/JsListView.vue:104`、`ui/src/components/js/JsListView.vue:610`。
- 影响：少量对象就占用多个屏幕，用户难以横向比较状态、版本和动作；嵌套圆角/阴影增加了视觉噪声。
- 最小修改：默认采用紧凑表格或列表行，详情按需展开；插件页移除一层容器卡片，描述限制行数，动作并入稳定的行尾区域。
- 验收：6 个牌堆和多个插件在 1280px 首屏能展示明显更多对象；展开详情不推动无关行，页面不出现卡片套卡片。

### G3. 基础设置的标签与开关距离过远，群组筛选标签又因固定宽度换行

- 证据：[基础设置平台页](ui-review-assets/2026-08-10/light/1280x800/misc-base-setting--platform.png)、[基础设置行为页](ui-review-assets/2026-08-10/dark/1280x800/misc-base-setting--behavior-message.png)、[群组设置](ui-review-assets/2026-08-10/light/1280x800/misc-group--default.png)。源码见 `ui/src/components/settings-panel/SettingRow.vue:52`、`ui/src/features/baseSetting/viewModel.ts:154`、`ui/src/pages/misc/group.vue:29`、`ui/src/pages/misc/group.vue:304`。
- 影响：同一表单体系一处把标签和控件拉得很开，另一处把“按最后使用排序”挤成两行，字段对应关系和纵向节律都不稳定。
- 最小修改：设置行采用有最大宽度的两列 grid；短开关靠近标签/说明，长输入占剩余空间。群组筛选提高 label width 或缩短文案，避免正常桌面宽度下换行。
- 验收：1280/1440 双主题下，标签和控件关系清楚；最长标签单行显示，开关不会贴到内容区最右端。

### G4. 敏感词响应设置缺少稳定列网格

- 证据：[浅色设置页](ui-review-assets/2026-08-10/light/1280x800/mod-censor--default.png)、[深色设置页](ui-review-assets/2026-08-10/dark/1280x800/mod-censor--default.png)。源码见 `ui/src/components/censor/CensorConfigView.vue:91`、`:148`、`:162`。
- 影响：等级、阈值、动作和分值按内容自然流动，行间难以横向比较；字段增加或文案变长后会继续错位。
- 最小修改：将响应规则行定义为明确 grid 列：等级、阈值、动作、分值；小宽度只允许按语义成组换行，不由文本长度随机决定。
- 验收：使用最长文案、不同数字位数测试，同行控件列对齐；1280px 不遮挡、不出现孤立字段。

### G5. 剧情卡记录动作过多且优先级平坦

- 证据：[剧情卡列表](ui-review-assets/2026-08-10/light/1280x800/mod-story--default.png)。源码见 `ui/src/pages/mod/story.vue:61`、`ui/src/components/common/FoldableCard.vue`。
- 影响：查看、提取、备份、删除等动作并列，主任务不突出，危险动作在高频区持续占位，增加误触和扫描成本。
- 最小修改：保留“查看/提取”等一到两个主动作，其余收进更多菜单；危险动作置于菜单末尾并保持二次确认。
- 验收：单行默认只显示核心动作；键盘和鼠标均能访问更多菜单，删除不与高频动作相邻。

### G6. 列表型工具栏没有统一模式：封禁动作纵向散落，敏感词筛选无可见标签

- 证据：[封禁列表](ui-review-assets/2026-08-10/light/1280x800/misc-ban--default.png)、[敏感词列表](ui-review-assets/2026-08-10/light/1440x1000/mod-censor--word.png)。源码见 `ui/src/components/ban/BanListPanel.vue:3`、`:15`、`:263`，`ui/src/components/censor/CensorWordsView.vue:2`、`:9`。
- 影响：封禁页命令与筛选关系松散，敏感词输入框又无法在空值时说明筛选对象；相似列表的操作肌理不一致。
- 最小修改：建立“筛选区 + 结果/批量动作区”两行工具栏；输入框提供持久可见标签“筛选敏感词”和明确 placeholder，危险动作与普通筛选分区。
- 验收：1280px 下命令保持横向分组且不拥挤；清空输入值后仍能识别字段用途，键盘焦点顺序与视觉顺序一致。

### G7. 公共骰列表把父容器透明度与子控件 disabled 状态叠加

- 证据：[深色 1280](ui-review-assets/2026-08-10/dark/1280x800/misc-dice-public--default.png)。源码见 `ui/src/pages/misc/dice-public.vue:111`、`:212`。
- 影响：禁用控件已经由 Naive UI 降低对比度，父层再设 `opacity: .72` 后形成双重变灰，文字和边界在深色主题下尤其难读。
- 最小修改：移除父容器整体 opacity，用状态背景、图标或提示说明“内容不可编辑”；让控件 disabled 样式单独表达不可操作。
- 验收：深色/浅色下禁用状态仍清楚，但说明文字保持可读；实测普通说明文字达到目标对比度。

### G8. 自定义文案筛选无匹配结果时没有空态

- 证据：[COC 修改过](ui-review-assets/2026-08-10/light/1280x800/custom-text-coc--modified.png)、[DND 修改过](ui-review-assets/2026-08-10/dark/1440x1000/custom-text-dnd--modified.png)、[其它旧版](ui-review-assets/2026-08-10/dark/1440x1000/custom-text-other--deprecated.png)。源码见 `ui/src/components/custom-text/CustomTextEditor.vue:20`、`ui/src/features/customText/useCustomTextEditor.ts:61`。
- 影响：筛选栏下直接出现大面积空白，用户无法判断筛选是否生效、数据是否尚未加载。
- 最小修改：在分类存在但过滤结果为零时显示 `n-empty`，说明当前条件无结果并提供“清除筛选”。
- 验收：搜索、修改过、未修改、旧版文本和指定分组的空结果均有对应持久空态。

### G9. “关于”页按视口而非实际内容宽度断点，1280px 下版本号被截断

- 证据：[深色 1280](ui-review-assets/2026-08-10/dark/1280x800/about--default.png) 中“最新版本”显示为 `1.6.0+20260...`。源码见 `ui/src/components/about/AboutHero.vue:148`、`:173`、`:232`。
- 影响：216px 侧栏压缩了实际容器，但五等分布局仍根据全视口判断；关键版本信息失真且没有完整值入口。
- 最小修改：使用容器查询或 `auto-fit/minmax` 根据内容宽度换行；必须截断时提供 Tooltip。关于页可保留较强品牌表达，但关键运行信息必须完整。
- 验收：1280/1440、长版本号和长运行环境组合下，完整值直接可见或可通过 Tooltip 获取，布局不横向溢出。

### G10. 备份文件列表缺少面向增长的局部滚动或分页

- 证据：[备份文件页](ui-review-assets/2026-08-10/light/1280x800/misc-backup--files.png)，当前后端已有 32 条记录。源码见 `ui/src/components/backup/BackupFileList.vue:36`、`:43`。
- 影响：当前数据量尚可使用，但继续增长会把标题、筛选和恢复上下文推离视野；长文件名也会降低扫描效率。
- 最小修改：增加分页，或使用有最大高度的表格与固定表头；文件名列设置 ellipsis 和完整值 Tooltip。
- 验收：以 100 条记录、长文件名和不同大小字段测试，页级工具栏不随数据增长远离视口，排序/分页状态可理解。

### G11. JS 数据页在未选择插件时呈现无解释空白

- 证据：[JS 数据页](ui-review-assets/2026-08-10/light/1440x1000/mod-js--data.png)。源码见 `ui/src/components/js/JsDataView.vue:15`。
- 影响：空白无法说明是“未选择插件”、插件无数据还是加载失败，状态反馈不完整。
- 最小修改：未选择时显示 `n-empty`“请选择插件查看数据”；已选择但无数据时使用不同文案，加载失败保留错误反馈。
- 验收：未选择、无数据、有数据、加载失败四种状态可仅凭页面文案区分。

## 5. 建议问题

### A1. 侧栏展开祖先和当前叶节点使用相同黄色

- 证据：[COC 文案浅色](ui-review-assets/2026-08-10/light/1440x1000/custom-text-coc--default.png)、[COC 文案深色](ui-review-assets/2026-08-10/dark/1280x800/custom-text-coc--default.png)。源码见 `ui/src/features/theme/themePalette.ts:176`。
- 影响：当前叶节点要结合缩进和面包屑才能确认；菜单增长后定位效率会下降。
- 建议：展开父级保持中性文字，只让唯一当前叶节点使用黄色、背景或侧边标记。验收时不借助面包屑即可定位当前项，同时能看出父级已展开。

### A2. 默认宽度页与 wide 页的内容左边界不连续

- 证据：默认布局在 1440px 下以 `1180px` 居中，wide 页面使用全宽；从[基础设置](ui-review-assets/2026-08-10/light/1440x1000/misc-base-setting--default.png)切换到[资源管理](ui-review-assets/2026-08-10/light/1440x1000/tool-resource--default.png)时内容起点发生跳动。源码见 `ui/src/components/app-shell/AppShell.vue:226`、`ui/src/router/navigation.ts`。
- 影响：跨模块切换时标题和主操作线轻微跳动，削弱全站栅格一致性。
- 建议：定义统一页级 gutter；default 与 wide 只改变最大内容宽度，不改变左侧基准线。数据表可在标题以下扩展，不要连同页头一起漂移。

## 6. 建议实施顺序

### P0：直接读取风险

1. 修复帮助文档词条表格越界。
2. 拆分语义色前景/填充令牌并完成双主题对比度测量。

### P1：建立跨页面骨架

1. 统一 `PageHeader`、页级 gutter、列表工具栏和表单行 grid。
2. 将首页日志改为受限高度工作区。
3. 重构多值文案、牌堆和插件列表的长内容/高密度呈现。

### P2：补齐状态与增长策略

1. 补充文案筛选、JS 数据页空态。
2. 为备份列表加入分页或内部滚动。
3. 调整关于页容器断点、公共骰禁用态、侧栏层级色。

实施时应优先复用 Naive UI 的 `NDataTable`、`NEmpty`、`NTooltip`、`NDropdown`、`NTabs` 和主题覆盖能力；避免另建一套相同功能组件，也避免用更多卡片包裹来解决分组问题。

## 7. 已确认的稳定点

- `AppShell` 在 1280/1440、浅色/深色下未出现整体横向溢出，侧栏、顶栏和内容区没有互相遮挡。
- 深色主题覆盖较完整，不是简单反色；输入、页签、表格和常用按钮的状态总体一致。
- 基础设置的七个模式、备份的设置/文件模式、指令测试的私聊/群聊模式具有清楚的主导航。
- 回复、插件包管理在当前数据状态下没有发现硬性布局错误；工具测试和个人资料页在当前数据下也基本稳定。
- 关于页属于品牌/产品信息页，允许比运维页更丰富的视觉表达；本报告只要求其关键信息和响应布局可用。

## 8. 覆盖索引

下表给出每个菜单页面的状态、截图数量和代表性默认态。完整归档位于 `docs/ui-review-assets/2026-08-10/`，文件名格式为 `route-slug--state.png`。

| 页面 | 路由 | 已覆盖状态 | 数量 | 浅色 1440 | 深色 1280 |
| --- | --- | --- | ---: | --- | --- |
| 主页 | `/` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/home--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/home--default.png) |
| 账号设置 | `/connect` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/connect--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/connect--default.png) |
| COC 文案 | `/custom-text/COC` | 全部、未修改、修改过、指定分组、旧版 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/custom-text-coc--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/custom-text-coc--default.png) |
| DND 文案 | `/custom-text/DND` | 全部、未修改、修改过、指定分组、旧版 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/custom-text-dnd--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/custom-text-dnd--default.png) |
| 其它文案 | `/custom-text/其它` | 全部、未修改、修改过、指定分组、旧版 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/custom-text-other--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/custom-text-other--default.png) |
| 娱乐文案 | `/custom-text/娱乐` | 全部、未修改、修改过、指定分组、旧版 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/custom-text-entertainment--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/custom-text-entertainment--default.png) |
| 日志文案 | `/custom-text/日志` | 全部、未修改、修改过、指定分组、旧版 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/custom-text-log--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/custom-text-log--default.png) |
| 核心文案 | `/custom-text/核心` | 全部、未修改、修改过、指定分组、旧版 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/custom-text-core--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/custom-text-core--default.png) |
| 自定义回复 | `/mod/reply` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-reply--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-reply--default.png) |
| 牌堆 | `/mod/deck` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-deck--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-deck--default.png) |
| 插件包管理 | `/mod/package` | 已安装、商店、管理 | 13 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-package--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-package--default.png) |
| 剧情卡 | `/mod/story` | 列表、清理、备份 | 13 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-story--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-story--default.png) |
| JS 插件 | `/mod/js` | 默认/列表、列表显式态、控制台、配置、数据 | 21 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-js--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-js--default.png) |
| 帮助文档 | `/mod/helpdoc` | 文件、词条 | 9 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-helpdoc--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-helpdoc--default.png) |
| 敏感词 | `/mod/censor` | 设置、词库、日志 | 13 | [查看](ui-review-assets/2026-08-10/light/1440x1000/mod-censor--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/mod-censor--default.png) |
| 基础设置 | `/misc/base-setting` | 7 个设置分组 | 29 | [查看](ui-review-assets/2026-08-10/light/1440x1000/misc-base-setting--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/misc-base-setting--default.png) |
| 群组设置 | `/misc/group` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/misc-group--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/misc-group--default.png) |
| 黑白名单 | `/misc/ban` | 列表、配置 | 9 | [查看](ui-review-assets/2026-08-10/light/1440x1000/misc-ban--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/misc-ban--default.png) |
| 公共骰列表 | `/misc/dice-public` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/misc-dice-public--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/misc-dice-public--default.png) |
| 备份 | `/misc/backup` | 设置、文件 | 9 | [查看](ui-review-assets/2026-08-10/light/1440x1000/misc-backup--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/misc-backup--default.png) |
| 高级设置 | `/misc/advanced-setting` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/misc-advanced-setting--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/misc-advanced-setting--default.png) |
| 指令测试 | `/tool/test` | 私聊、群聊 | 9 | [查看](ui-review-assets/2026-08-10/light/1440x1000/tool-test--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/tool-test--default.png) |
| 资源管理 | `/tool/resource` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/tool-resource--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/tool-resource--default.png) |
| 个人资料 | `/tool/profile` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/tool-profile--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/tool-profile--default.png) |
| 关于 | `/about` | 默认、完整高度 | 5 | [查看](ui-review-assets/2026-08-10/light/1440x1000/about--default.png) | [查看](ui-review-assets/2026-08-10/dark/1280x800/about--default.png) |

## 9. 未覆盖与残余风险

- 本轮仅覆盖桌面端；移动端、侧栏折叠、200% 缩放和系统高对比模式未纳入。
- 未触发保存、删除、清理、恢复、上传等会修改后端数据的弹窗、抽屉和结果态。
- 当前群组、封禁、敏感词、插件和资源数据量较低；超长名称、大批量记录、异常请求和权限差异仍需专项压力数据验证。
- 已完成静态截图和代表性交互检查，但没有执行完整键盘遍历、读屏语义审计或全页面自动化对比度扫描。
- 截图来自开发模式，Vue Devtools 开关不应出现在正式构建；正式验收应再用 production build 复拍关键页面。

