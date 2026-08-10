# UI 可见页面静态可用性审查

审查日期：2026-08-10

审查范围：`ui/` 下当前可见用户页面与共享壳层。

- 顶层页：`/`、`/connect`、`/custom-text/:category`、`/about`
- 扩展功能：`/mod/reply`、`/mod/deck`、`/mod/package`、`/mod/story`、`/mod/js`、`/mod/helpdoc`、`/mod/censor`
- 综合设置：`/misc/base-setting`、`/misc/group`、`/misc/ban`、`/misc/dice-public`、`/misc/backup`、`/misc/advanced-setting`
- 辅助工具：`/tool/test`、`/tool/resource`、`/tool/profile`
- 共享框架：`AppShell`、侧栏、面包屑、全局搜索

证据来源：静态代码审查。未执行浏览器运行时、接口故障注入、键盘/读屏/缩放实测。

## 发现

### [阻断] 跑团日志页“全选”实际执行反选，批量删除目标不可确认

- 证据：`ui/src/pages/mod/story.vue:28`、`ui/src/pages/mod/story.vue:35`、`ui/src/pages/mod/story.vue:550`
- 规则：违反高风险动作必须明确对象与后果、避免不可逆误操作的要求。
- 影响：部分已选时点击“全选”会把选择集反转；删除确认又不回显数量或名称，误删风险直接升高。
- 整改：改为显式“全选/全不选”；删除确认至少展示选中数量与日志名摘要。
- 验证：先选中 1 条再点“全选”，结果应是全部选中而不是反选；删除确认应能仅靠弹层确认目标。

### [阻断] 添加账号向导前三步只支持鼠标，键盘无法完成核心创建流程

- 证据：`ui/src/components/connect/ConnectCreateWizard.vue:13`、`ui/src/components/connect/ConnectCreateWizard.vue:35`、`ui/src/components/connect/ConnectCreateWizard.vue:57`
- 规则：核心任务必须具备完整键盘路径和语义控件。
- 影响：键盘用户无法在平台、连接方式、协议三步中完成选择，账号创建主流程在入口阶段即被阻断。
- 整改：把三列可选项改成真正的 `button`/`n-button`/可聚焦列表项，补齐 `Enter/Space` 触发与选中/禁用语义。
- 验证：只用 `Tab/Shift+Tab/Enter/Space` 应能完整走完“添加账号”流程。

### [严重] 全局搜索结果项嵌套二级按钮，语义非法且键盘路径不稳定

- 证据：`ui/src/components/app-shell/AppSearchMenu.vue:57`、`ui/src/components/app-shell/AppSearchMenu.vue:76`
- 规则：违反语义控件、清晰焦点顺序、不能只支持鼠标的要求。
- 影响：结果行本身是 `<button>`，内部又嵌套多个 `n-button`。这会破坏辅助技术的名称/角色解析，也会让复制链接、另开窗口、删除历史等操作的键盘行为不可预测。
- 整改：把结果行改成非按钮容器加独立操作按钮，或把主操作与次操作拆成并列可聚焦元素，并补 `role="option"` / `aria-selected`。
- 验证：仅用键盘应能稳定完成“选中结果”“复制链接”“删除历史”三条路径；可访问性树不应再出现交互控件嵌套。

### [严重] 群组管理退群确认不回显真实对象

- 证据：`ui/src/pages/misc/group.vue:136`、`ui/src/pages/misc/group.vue:139`、`ui/src/pages/misc/group.vue:393`、`ui/src/pages/misc/group.vue:407`
- 规则：高风险动作确认层必须明确对象与后果。
- 影响：单群退群不显示 `groupId/groupName/diceId`，批量退群只显示数量；操作者无法仅看弹层核对目标。
- 整改：单群场景回显群号、群名、执行账号；批量场景回显目标列表或至少摘要。
- 验证：不看背景页，仅看弹层即可确认退群对象。

### [严重] 群组管理缺少提交态、防重复提交和就地错误恢复

- 证据：`ui/src/pages/misc/group.vue:131`、`ui/src/pages/misc/group.vue:152`、`ui/src/pages/misc/group.vue:327`、`ui/src/pages/misc/group.vue:372`、`ui/src/pages/misc/group.vue:412`、`ui/src/pages/misc/group.vue:463`
- 规则：异步操作应有可见提交态、失败态与恢复路径，不能只靠 Toast。
- 影响：搜索、保存、批量通知、退群都直接等待请求；按钮在请求中仍可能再次点击，失败后也没有保留在当前弹层/页面内的错误说明。
- 整改：为各操作建立独立 `pending/error`；确认按钮绑定 `loading/disabled`；在当前上下文内展示失败原因与重试入口。
- 验证：模拟超时/500 并双击提交，界面应只发起一次请求并保留原输入。

### [严重] 公骰设置的即时开关会连带提交整页草稿

- 证据：`ui/src/pages/misc/dice-public.vue:6`、`ui/src/pages/misc/dice-public.vue:15`、`ui/src/pages/misc/dice-public.vue:151`、`ui/src/pages/misc/dice-public.vue:160`
- 规则：状态模型必须一致、可预测。
- 影响：页面同时存在“保存”按钮和未保存保护，但启用/关闭开关会直接提交整份草稿。用户只想改开关时，可能把昵称、终端勾选等未准备好的内容一并保存。
- 整改：要么整页都改成显式保存，要么把开关拆成独立 mutation，只提交 `publicDiceEnable`。
- 验证：修改其他字段但不保存，再切换启用开关，不应无提示连带提交其他草稿。

### [严重] 高级设置一次保存两份资源，失败时无法区分部分成功

- 证据：`ui/src/pages/misc/advanced-setting.vue:222`、`ui/src/pages/misc/advanced-setting.vue:228`、`ui/src/pages/misc/advanced-setting.vue:241`
- 规则：部分成功必须可理解，失败不能只有笼统结论。
- 影响：高级配置保存成功但回复调试日志保存失败时，页面只提示“保存失败”；用户不知道哪些值已落库、哪些没有。
- 整改：拆成两个可见结果，或在一次保存里明确展示“已保存/未保存”的分项结果，并回读服务端状态对齐 UI。
- 验证：让第二个接口失败、第一成功，页面应能准确呈现部分成功。

### [严重] 备份设置缺少字段级格式校验，失败反馈还会重复出现

- 证据：`ui/src/components/backup/BackupConfigPanel.vue:23`、`ui/src/components/backup/BackupConfigPanel.vue:48`、`ui/src/components/backup/BackupConfigPanel.vue:98`、`ui/src/components/backup/BackupConfigPanel.vue:123`、`ui/src/pages/misc/backup.vue:160`、`ui/src/pages/misc/backup.vue:295`
- 规则：格式敏感字段应配套规则与字段级错误提示。
- 影响：`autoBackupTime`、`backupCleanKeepDur`、`backupCleanCron` 都可能因为格式错误保存失败，但页面只能靠 Toast；而页面层和 mutation 层都会报错，单次失败会重复提示。
- 整改：为 cron、时长和触发组合增加字段级校验；统一错误出口，避免同一次失败双重提示。
- 验证：输入非法 cron 后提交，错误应挂在字段下，且只出现一次失败提示。

### [严重] 跑团日志原始日志分页把页大小当成页码处理

- 证据：`ui/src/pages/mod/story.vue:175`、`ui/src/pages/mod/story.vue:182`、`ui/src/pages/mod/story.vue:628`
- 规则：分页行为必须连续、可理解。
- 影响：`@update:page-size` 绑定到了只更新 `pageNum` 的处理器；用户改每页条数时会跳到错误页码。
- 整改：拆分页码与页大小处理器；切页大小时更新 `pageSize` 并重置 `pageNum = 1`。
- 验证：把每页条数从 50 改到 200，请求参数应变为 `pageSize=200&pageNum=1`。

### [严重] 跑团日志清理参数没有程序化标签

- 证据：`ui/src/pages/mod/story.vue:208`
- 规则：危险表单必须有语义标签和名称关联。
- 影响：`n-input-number` 与 `n-switch` 只靠视觉邻近说明，对键盘/读屏用户无法可靠区分“未更新月数”和“执行 VACUUM”。
- 整改：改成 `n-form` + `n-form-item`，或补齐可访问名称与描述关联。
- 验证：在可访问性树中，这两个控件都应有稳定可读名称。

### [严重] JS 扩展总开关在后端失败时仍会切换本地状态

- 证据：`ui/src/pages/mod/js.vue:274`、`ui/src/pages/mod/js.vue:293`、`ui/src/pages/mod/js.vue:313`
- 规则：结果必须可理解、失败可恢复，状态以真实后端为准。
- 影响：`reload/shutdown` 失败后仍会把本地 `jsEnable` 改到目标值，造成“看起来已启用/已关闭，实际没有”的假状态。
- 整改：失败时回滚 UI，或强制回读服务端状态后再刷新开关。
- 验证：模拟失败后，开关应恢复原值并展示真实状态。

### [严重] 包管理的安装入口可绕过预览/确认

- 证据：`ui/src/components/package/PackageManagerView.vue:114`、`ui/src/components/package/PackageManagerView.vue:124`、`ui/src/components/package/PackageManagerView.vue:143`、`ui/src/components/package/PackageManagerView.vue:407`
- 规则：高风险导入/安装动作应先说明对象和影响，再确认。
- 影响：上传安装、URL 安装、商店安装都可能直接执行；用户可跳过包 ID、版本、文件树与覆盖动作确认。
- 整改：统一所有安装入口先走预览，再从预览界面发正式安装请求。
- 验证：三种入口都应在发安装请求前展示同一套预览确认。

### [严重] 帮助文档批量删除不披露实际删除范围

- 证据：`ui/src/components/helpdoc/HelpdocFilePane.vue:60`、`ui/src/components/helpdoc/HelpdocFilePane.vue:68`、`ui/src/pages/mod/helpdoc.vue:259`
- 规则：批量危险操作必须明确目标集合。
- 影响：文件树是级联勾选，但确认框只写“确认删除选择的文件吗？”，用户无法知道自己到底删几项、是否包含子节点。
- 整改：确认框展示选中文件数量和文件名列表，并标注级联子项数量。
- 验证：勾选父节点后，确认框应能准确反映将删除的完整范围。

### [严重] 账号设置的启用/删除确认不回显账号对象与目标状态

- 证据：`ui/src/pages/connect.vue:314`、`ui/src/pages/connect.vue:328`、`ui/src/components/connect/ConnectTableColumns.tsx:79`
- 规则：删除、禁用、启用等高风险动作应在确认层复述对象与后果。
- 影响：表格里可以看到昵称、协议、状态，但确认弹层只说“删除此项帐号，确定吗？”和“确认修改此账号的在线状态吗？”。多账号相似时容易误操作。
- 整改：确认框至少展示昵称/用户 ID/协议，并明确目标动作是“启用”还是“禁用”。
- 验证：仅看弹层即可区分操作对象和目标状态。

### [严重] 编辑账号配置加载失败时直接关闭弹窗并清空上下文

- 证据：`ui/src/pages/connect.vue:268`、`ui/src/pages/connect.vue:277`、`ui/src/components/connect/ConnectEditDialog.vue:11`
- 规则：弹层内关键错误应保留当前上下文并提供恢复路径，不能只给一次 Toast。
- 影响：编辑链路失败后，页面直接清空目标账号和表单上下文并关弹窗；用户被踢回列表，无法原位重试。
- 整改：保留弹窗并展示“读取失败 + 重试/关闭”；不要在错误分支立刻清空 `editingEndpoint`。
- 验证：模拟配置读取失败，弹窗仍应保留目标账号信息并可原位重试。

### [严重] 自定义文案导入把解析失败和保存失败混成同一类错误，且可能部分成功

- 证据：`ui/src/features/customText/useCustomTextEditor.ts:158`
- 规则：部分成功必须可理解，失败反馈必须真实。
- 影响：导入按分类逐个保存；任一分类保存失败都会被 `catch` 吃掉并显示“格式不正确”。这会把服务端/网络故障误报成输入格式错误，同时隐藏“前几个分类已经保存成功”的事实。
- 整改：把“解析失败”和“保存失败”拆开处理；若存在逐分类写入，需明确告知已成功/失败的分类并保留重试入口。
- 验证：模拟第二个分类保存失败，界面应显示部分成功而不是“格式不正确”。

### [严重] 资源管理允许多文件上传，却完全隐藏逐文件队列和结果

- 证据：`ui/src/components/resource/ResourceListPanel.vue:22`、`ui/src/components/resource/ResourceListPanel.vue:27`、`ui/src/pages/tool/resource.vue:145`、`ui/src/pages/tool/resource.vue:167`
- 规则：上传流程应提供逐文件进度、成功、失败与重试，不应只靠短暂消息提示。
- 影响：用户一次选 3 个以上文件时，界面无法说明哪张成功、哪张失败，也没有逐项重试路径，容易重复上传或漏传。
- 整改：显示上传队列并保留逐文件状态；若暂时做不到，先移除 `multiple`，退回单文件明确流程。
- 验证：一次选择多个文件并制造混合结果，界面应能逐项区分成败。

### [严重] 指令测试成员侧栏把“整行选中”和“编辑身份”嵌套在同一区域，语义冲突

- 证据：`ui/src/components/tool-test/ToolTestMemberRail.vue:33`、`ui/src/components/tool-test/ToolTestMemberRail.vue:46`、`ui/src/components/tool-test/ToolTestMemberRail.vue:53`
- 规则：不同任务应拆成独立语义控件，不能嵌套交互控件。
- 影响：切换当前用户和编辑身份是两个不同任务，但现在会出现焦点歧义和误触发，键盘与辅助技术路径都不稳定。
- 整改：把整行选中和编辑入口拆成两个并列控件；若整行可点，行内容内部不要再嵌套按钮。
- 验证：仅用 `Tab/Enter/Space` 依次操作“选中当前用户”和“编辑身份”时，两者应互不串扰。

### [一般] 牌堆 diff 弹层在更新失败时被直接关闭

- 证据：`ui/src/pages/mod/deck.vue:255`、`ui/src/pages/mod/deck.vue:516`
- 规则：失败时应保留当前上下文，便于恢复。
- 影响：更新失败后丢失旧/新内容对比，只剩短暂错误提示，用户需要重新执行检查更新才能重新判断。
- 整改：失败时保留弹层和 diff 内容，并在弹层内展示错误与重试入口。
- 验证：模拟更新失败后，diff 仍应保持打开。

### [一般] 包配置抽屉允许非法 JSON 继续提交

- 证据：`ui/src/components/package/PackageDetailDrawer.vue:50`、`ui/src/components/package/PackageDetailDrawer.vue:65`、`ui/src/components/package/PackageDetailDrawer.vue:142`
- 规则：复杂结构输入应有字段级校验。
- 影响：JSON 解析失败时没有字段级错误，用户只能等服务端失败后得到笼统提示。
- 整改：为 JSON 字段增加本地校验；存在错误时禁用保存。
- 验证：输入非法 JSON 后，应直接在字段下看到错误并无法提交。

### [一般] 折叠卡片按钮未暴露展开/收起状态

- 证据：`ui/src/components/shared/FoldableCard.vue:19`、`ui/src/components/shared/FoldableCard.vue:87`、`ui/src/components/settings-panel/SettingCategoryBox.vue:10`、`ui/src/components/settings-panel/SettingCategoryBox.vue:22`
- 规则：状态型控件应提供名称、角色和值。
- 影响：群组页和基本设置页的折叠开关缺少 `aria-expanded` / `aria-controls`，读屏用户无法清楚得知当前状态。
- 整改：为按钮补 `aria-label`、`aria-expanded`、`aria-controls`，面板补稳定 `id`。
- 验证：切换前后可访问性树中的展开状态应同步变化。

### [一般] 黑白名单删除确认未复述目标对象

- 证据：`ui/src/pages/misc/ban.vue:208`、`ui/src/components/ban/BanListPanel.vue:48`
- 规则：破坏性确认应重述一次对象。
- 影响：弹层里只有“是否删除此记录？”，相似记录下需要依赖记忆回想目标。
- 整改：确认内容中带上 `item.ID`、`item.name` 和等级摘要。
- 验证：只看弹层即可区分相邻相似记录。

### [一般] 首页网络质量刷新只绑定在可点击 `div` 上

- 证据：`ui/src/pages/index.vue:33`
- 规则：可操作元素应使用语义按钮并支持键盘。
- 影响：当前“点击重新进行检测”依赖一个带 `@click` 的普通容器；键盘用户和辅助技术用户无法把它稳定识别为可触发动作。
- 整改：改为 `n-button` 或给容器补 `role="button"`、`tabindex="0"` 和键盘事件。
- 验证：仅用键盘应能触发网络检测刷新。

### [一般] 指令测试聊天窗口收到新消息就强制滚到底，历史上下文不稳定

- 证据：`ui/src/components/tool-test/ToolTestChatWindow.vue:13`、`ui/src/components/tool-test/ToolTestChatWindow.vue:23`
- 规则：数据密集型工作台应保留滚动上下文，不应在刷新时抢走用户位置。
- 影响：用户查看历史消息时，只要新消息进入就被拉回底部，影响比对上下文和排查连续输出。
- 整改：只在用户已接近底部时自动滚动；否则提示“有新消息”，由用户决定是否跳转。
- 验证：上滚到历史消息后再注入新消息，滚动位置应保持不变。

### [一般] 自定义文案卡片把新增/删除/重置做成纯图标点击区，缺少按钮语义和名称

- 证据：`ui/src/components/custom-text/CustomTextEntryCard.vue:28`、`ui/src/components/custom-text/CustomTextEntryCard.vue:44`、`ui/src/components/custom-text/CustomTextEntryCard.vue:69`
- 规则：关键动作应提供稳定点击区与可访问名称，危险动作不能只靠图标和颜色。
- 影响：条目维护动作的可发现性和键盘可达性都偏弱，误触后也不容易理解刚刚执行了什么。
- 整改：改成带名称的 `n-button`，或至少补 `aria-label` 的图标按钮；危险动作再加 `popconfirm`。
- 验证：只用键盘应能完成“新增条目、删除条目、重置键、删除键”四个动作，并能读出名称。

### [建议] 顶栏新闻入口目前是可见占位功能，不提供真实内容或未读状态

- 证据：`ui/src/components/app-shell/AppBreadcrumb.vue:54`、`ui/src/components/app-shell/AppBreadcrumb.vue:97`、`ui/src/components/app-shell/AppBreadcrumb.vue:136`
- 规则：任务型后台的顶栏入口应提供真实状态，不应制造空功能占位。
- 影响：新闻入口始终以本地默认值运行，未读状态不会变化，弹层内容固定为“暂无内容”，会消耗顶部注意力却不产生业务价值。
- 整改：若功能暂未接入，建议隐藏；若保留，则补齐数据加载、已读状态与失败态。
- 验证：断网/有新内容/已读三种状态都应有可观察差异。

## 待核验

- 未执行浏览器运行时验证：键盘完整路径、焦点返回、屏幕阅读器名称/角色/值、`200%` 文本缩放、`400%` 页面缩放。
- 未执行服务端行为验证：安装/删除/退群/启用禁用/备份是否具备幂等、防重复提交、审计与部分成功对账。
- 静态审查中未发现可证实高风险问题的页面包括：`/mod/reply`、`/mod/censor`、`/tool/profile`、`/about`。这些页面仍需运行时补测键盘、读屏和异常路径。

## 简要结论

- 最高优先级问题集中在三类：高风险动作确认不完整、失败后状态不可信、格式敏感表单缺少字段级校验。
- 本次共享壳层里，`AppSearchMenu` 的交互语义问题最值得先修；页面层里优先处理 `story`、`connect`、`group`、`dice-public`、`advanced-setting`、`backup`、`js`、`package`、`helpdoc`、`custom-text`、`tool/resource`、`tool/test`。
- 如果要安排整改顺序，建议先修阻断与严重问题中的“删除/退群/安装/启停”动作，再修分页、字段校验和可访问性语义。
