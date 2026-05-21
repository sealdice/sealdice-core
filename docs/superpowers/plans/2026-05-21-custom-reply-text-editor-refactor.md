# 自定义回复与自定义文案编辑器拆分重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `ui/src/pages/mod/reply.vue` 和 `ui/src/pages/custom-text/[category].vue` 拆为薄路由页、可测试 feature 模块和职责单一的 Vue 组件，保持现有交互与 API 行为不变。

**Architecture:** 路由页只负责传递 route 参数并挂载 feature 容器；数据标准化、导入解析、筛选排序、脏状态判断进入 `features/customReply` 与 `features/customText`。Vue 组件使用 `<script setup lang="ts">`、props down/events up；跨区块状态由 feature composable 统一托管。

**Tech Stack:** Vue 3.5 + Composition API + TypeScript + Vue Query + Naive UI + ProNaiveUI + `node scripts/run-tests.mjs` + `vue-tsc --build`

---

### 文件结构映射

| 文件 | 职责 |
|------|------|
| `ui/src/pages/mod/reply.vue` | 路由壳，仅渲染 `CustomReplyEditor` |
| `ui/src/components/custom-reply/CustomReplyEditor.vue` | 自定义回复 feature 容器，组合 sidebar、详情、规则、弹窗 |
| `ui/src/components/custom-reply/ReplyFileSidebar.vue` | 文件列表、文件筛选、上传、分页和文件级操作 |
| `ui/src/components/custom-reply/ReplyMetaSection.vue` | 当前回复文件基础信息表单 |
| `ui/src/components/custom-reply/ReplyCommonConditionsSection.vue` | 公共触发条件列表和分页 |
| `ui/src/components/custom-reply/ReplyRulesSection.vue` | 规则列表、规则分页和新增/删除 |
| `ui/src/components/custom-reply/ReplyImportModal.vue` | 旧格式导入弹窗 |
| `ui/src/components/custom-reply/ReplyLicenseModal.vue` | 自定义回复启用协议弹窗 |
| `ui/src/features/customReply/model.ts` | 回复文件草稿类型、标准化、clone、API payload 转换 |
| `ui/src/features/customReply/model.test.ts` | 回复模型标准化和 payload 测试 |
| `ui/src/features/customReply/importParser.ts` | 旧版回复文本导入解析 |
| `ui/src/features/customReply/importParser.test.ts` | 导入解析边界测试 |
| `ui/src/features/customReply/useCustomReplyEditor.ts` | 查询、草稿、分页、保存、上传、删除、未保存守卫 |
| `ui/src/pages/custom-text/[category].vue` | 路由壳，仅解析 category 并渲染 `CustomTextEditor` |
| `ui/src/components/custom-text/CustomTextEditor.vue` | 自定义文案 feature 容器 |
| `ui/src/components/custom-text/CustomTextHelp.vue` | 帮助内容展示 |
| `ui/src/components/custom-text/CustomTextToolbar.vue` | 搜索、刷新预览、导入导出入口 |
| `ui/src/components/custom-text/CustomTextFilterBar.vue` | 文案筛选模式和分组筛选 |
| `ui/src/components/custom-text/CustomTextEntryCard.vue` | 单个文案 key 的多条文本编辑、变量、预览提示、删除/重置 |
| `ui/src/components/custom-text/CustomTextImportModal.vue` | 自定义文案导入导出弹窗 |
| `ui/src/features/customText/viewModel.ts` | 文案筛选排序、分组、导入导出 payload、条目 key 生成 |
| `ui/src/features/customText/viewModel.test.ts` | 文案 view model 行为测试 |
| `ui/src/features/customText/useCustomTextEditor.ts` | 查询、草稿同步、保存、预览刷新、未保存守卫 |
| `ui/components.d.ts` | 自动组件声明，运行类型检查后更新 |

### Component Map

| Component | Single responsibility | Props | Emits |
|-----------|-----------------------|-------|-------|
| `CustomReplyEditor` | 组合自定义回复完整工作台 | 无 | 无 |
| `ReplyFileSidebar` | 文件筛选和文件列表操作 | `files`, `total`, `query`, `selectedFilename`, `loading`, `page`, `pageSize` | `select`, `update:query`, `update:page`, `create`, `delete`, `download`, `upload` |
| `ReplyMetaSection` | 编辑当前文件元信息 | `draft`, `replyEnabled`, `saving` | `toggle-reply-enabled`, `toggle-file-enabled`, `save`, `update:draft` |
| `ReplyCommonConditionsSection` | 编辑公共触发条件 | `conditions`, `page`, `pageSize`, `total` | `add`, `delete`, `update:conditions`, `update:page` |
| `ReplyRulesSection` | 编辑规则列表 | `rules`, `page`, `pageSize`, `total` | `add`, `delete`, `update:rules`, `update:page` |
| `ReplyImportModal` | 接收旧格式文本并触发导入 | `show`, `loading` | `update:show`, `import` |
| `CustomTextEditor` | 组合自定义文案完整工作台 | `category` | 无 |
| `CustomTextToolbar` | 搜索和工具动作 | `keyword`, `previewLoading` | `update:keyword`, `refresh-preview`, `open-import` |
| `CustomTextFilterBar` | 筛选模式和分组选择 | `mode`, `groups`, `group` | `update:mode`, `update:group` |
| `CustomTextEntryCard` | 编辑一个文案 key 下的多条文本 | `category`, `keyName`, `items`, `help`, `preview` | `add-item`, `remove-item`, `change`, `delete-key`, `reset-key` |
| `CustomTextImportModal` | 文案导入导出 | `show`, `content`, `onlyCurrent`, `compact`, `saving` | `update:show`, `update:content`, `update:onlyCurrent`, `update:compact`, `copy`, `clear`, `import` |

### Task 1: 自定义回复模型抽取

**Files:**
- Create: `ui/src/features/customReply/model.ts`
- Create: `ui/src/features/customReply/model.test.ts`
- Modify: `ui/src/pages/mod/reply.vue`

- [ ] **Step 1: 写模型测试**

```ts
import assert from 'node:assert/strict';
import {
  cloneReplyFileDraft,
  cloneReplyTask,
  normalizeCondition,
  normalizeReplyFileDetail,
  normalizeReplyTask,
  toApiReplyConfig,
  type ReplyFileDraft,
} from './model';

const task = normalizeReplyTask({
  enable: false,
  conditions: [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '12' }],
  results: [{ resultType: 'replyToSender', delay: '3', message: [['ok', '2']] }],
});

assert.equal(task.enable, false);
assert.deepEqual(task.conditions, [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '12' }]);
assert.deepEqual(task.results[0]?.message, [['ok', 2]]);

const numericCondition = normalizeCondition({ condType: 'textLenLimit', matchType: 'matchGreater', value: 5 });
assert.equal(numericCondition.value, 5);

const draft = normalizeReplyFileDetail({
  enable: true,
  interval: 2,
  name: 'demo',
  author: ['a'],
  version: '1',
  createTimestamp: 1,
  updateTimestamp: 2,
  desc: 'desc',
  storeID: 'store',
  conditions: [{ condType: 'textMatch', matchType: 'matchExact', value: 'ping' }],
  items: [],
  filename: 'demo.yaml',
  itemCount: 1,
} as never);

assert.equal(draft.filename, 'demo.yaml');
assert.equal(draft.items.length, 0);

const clonedTask = cloneReplyTask(task);
clonedTask.results[0]!.message[0]![0] = 'changed';
assert.equal(task.results[0]!.message[0]![0], 'ok');

const clonedDraft = cloneReplyFileDraft({
  ...draft,
  items: [task],
} satisfies ReplyFileDraft);
clonedDraft.author.push('b');
assert.deepEqual(draft.author, ['a']);

const payload = toApiReplyConfig({
  ...draft,
  items: [task],
  conditions: [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '7' }],
});
assert.equal(payload.conditions[0]!.value, 7);
assert.equal(payload.items[0]!.results[0]!.delay, 3);
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd ui && pnpm test -- customReply/model.test.ts`

Expected: FAIL，错误包含 `Cannot find module './model'`。

- [ ] **Step 3: 创建 `model.ts` 并迁移类型与函数**

从 `ui/src/pages/mod/reply.vue` 移入并导出这些成员：

```ts
export type ReplyCondition = {
  condType: string;
  matchType: string;
  matchOp?: string;
  value: string | number;
};

export type ReplyMessage = [string, number];

export type ReplyResult = {
  resultType: string;
  delay: number;
  message: ReplyMessage[];
};

export type ReplyTask = {
  enable: boolean;
  conditions: ReplyCondition[];
  results: ReplyResult[];
};

export type ReplyFileDraft = {
  enable: boolean;
  interval: number;
  name: string;
  author: string[];
  version: string;
  createTimestamp: number;
  updateTimestamp: number;
  desc: string;
  storeID: string;
  conditions: ReplyCondition[];
  items: ReplyTask[];
  filename: string;
  itemCount: number;
};
```

继续迁移并导出：

```ts
export function cloneReplyTask(item: ReplyTask): ReplyTask
export function cloneReplyFileDraft(item: ReplyFileDraft): ReplyFileDraft
export function normalizeReplyFileDetail(detail: ReplyFileDetail): ReplyFileDraft
export function normalizeReplyTask(item: unknown): ReplyTask
export function normalizeConditions(items: unknown[] | null | undefined): ReplyCondition[]
export function normalizeCondition(item: unknown): ReplyCondition
export function normalizeResults(items: unknown[] | null | undefined): ReplyResult[]
export function normalizeMessages(items: unknown[] | null | undefined): ReplyMessage[]
export function toApiReplyConfig(draft: ReplyFileDraft): {
  enable: boolean;
  interval: number;
  items: Array<{
    enable: boolean;
    conditions: ReplyCondition[];
    results: Array<{ resultType: string; delay: number; message: ReplyMessage[] }>;
  }>;
  name: string;
  author: string[];
  version: string;
  createTimestamp: number;
  updateTimestamp: number;
  desc: string;
  storeID: string;
  filename: string;
  conditions: ReplyCondition[];
}
```

- [ ] **Step 4: 更新页面 imports**

在 `reply.vue` 中删除本地类型和函数定义，改为：

```ts
import {
  cloneReplyFileDraft,
  cloneReplyTask,
  normalizeCondition,
  normalizeReplyFileDetail,
  normalizeReplyTask,
  toApiReplyConfig,
  type ReplyCondition,
  type ReplyFileDraft,
  type ReplyTask,
} from '@/features/customReply/model';
```

- [ ] **Step 5: 运行测试并提交**

Run: `cd ui && pnpm test -- customReply/model.test.ts`

Expected: PASS。

Commit:

```bash
git add ui/src/features/customReply/model.ts ui/src/features/customReply/model.test.ts ui/src/pages/mod/reply.vue
git commit --author="PaienNate <1101839859@qq.com>" -m "refactor(custom-reply): 抽取回复草稿模型"
```

### Task 2: 自定义回复导入解析抽取

**Files:**
- Create: `ui/src/features/customReply/importParser.ts`
- Create: `ui/src/features/customReply/importParser.test.ts`
- Modify: `ui/src/pages/mod/reply.vue`

- [ ] **Step 1: 写导入解析测试**

```ts
import assert from 'node:assert/strict';
import { parseReplyImportLine, parseReplyImportText } from './importParser';

assert.deepEqual(parseReplyImportLine('ping/pong'), {
  conditions: ['ping'],
  replies: ['pong'],
  rest: '',
});

assert.deepEqual(parseReplyImportLine('a|b/c|d'), {
  conditions: ['a', 'b'],
  replies: ['c', 'd'],
  rest: '',
});

assert.deepEqual(parseReplyImportLine('a/|b/c'), {
  conditions: ['a/|b'],
  replies: ['c'],
  rest: '',
});

assert.deepEqual(parseReplyImportLine('a/line\\nnext\\nrest'), {
  conditions: ['a'],
  replies: ['line\nnext'],
  rest: 'rest',
});

const tasks = parseReplyImportText('a/b\nc/d');
assert.equal(tasks.length, 2);
assert.equal(tasks[0]!.conditions[0]!.value, 'a');
assert.deepEqual(tasks[1]!.results[0]!.message, [['d', 1]]);
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd ui && pnpm test -- customReply/importParser.test.ts`

Expected: FAIL，错误包含 `Cannot find module './importParser'`。

- [ ] **Step 3: 创建 `importParser.ts`**

从页面迁移 `parseString` 的行为，并对外暴露：

```ts
import type { ReplyTask } from './model';

export type ReplyImportLine = {
  conditions: string[];
  replies: string[];
  rest: string;
};

export function parseReplyImportLine(input: string): ReplyImportLine

export function parseReplyImportText(input: string): ReplyTask[] {
  const tasks: ReplyTask[] = [];
  let text = input;
  while (text) {
    const { conditions, replies, rest } = parseReplyImportLine(text);
    if (conditions.length && replies.length) {
      tasks.push({
        enable: true,
        conditions: [{
          condType: 'textMatch',
          matchType: 'matchMulti',
          value: conditions.join('|'),
        }],
        results: [{
          resultType: 'replyToSender',
          delay: 0,
          message: replies.map(reply => [reply, 1]),
        }],
      });
    }
    text = rest;
  }
  return tasks;
}
```

- [ ] **Step 4: 替换页面导入逻辑**

在 `reply.vue` 中引入：

```ts
import { parseReplyImportText } from '@/features/customReply/importParser';
```

将 `doImport()` 中手写 while 循环替换为：

```ts
const importedTasks = parseReplyImportText(configForImport.value);
currentFileDraft.value.items.push(...importedTasks);
```

- [ ] **Step 5: 运行测试并提交**

Run: `cd ui && pnpm test -- customReply/importParser.test.ts customReply/model.test.ts`

Expected: PASS。

Commit:

```bash
git add ui/src/features/customReply/importParser.ts ui/src/features/customReply/importParser.test.ts ui/src/pages/mod/reply.vue
git commit --author="PaienNate <1101839859@qq.com>" -m "refactor(custom-reply): 抽取回复导入解析"
```

### Task 3: 自定义回复编辑器组件拆分

**Files:**
- Create: `ui/src/components/custom-reply/CustomReplyEditor.vue`
- Create: `ui/src/components/custom-reply/ReplyFileSidebar.vue`
- Create: `ui/src/components/custom-reply/ReplyMetaSection.vue`
- Create: `ui/src/components/custom-reply/ReplyCommonConditionsSection.vue`
- Create: `ui/src/components/custom-reply/ReplyRulesSection.vue`
- Create: `ui/src/components/custom-reply/ReplyImportModal.vue`
- Create: `ui/src/components/custom-reply/ReplyLicenseModal.vue`
- Create: `ui/src/features/customReply/useCustomReplyEditor.ts`
- Modify: `ui/src/pages/mod/reply.vue`

- [ ] **Step 1: 建立 composable 公共返回契约**

在 `useCustomReplyEditor.ts` 中导出：

```ts
export function useCustomReplyEditor() {
  return {
    selectedFilename,
    fileQuery,
    fileItems,
    fileTotal,
    pageBusy,
    replyEnabled,
    currentFileDraft,
    commonConditionsPage,
    commonConditionsPageSize,
    commonConditionsTotal,
    pagedCommonConditions,
    rulesPage,
    rulesPageSize,
    rulesTotal,
    rulePageItems,
    modified,
    saveMutation,
    replyConfigMutation,
    selectFile,
    handleReplySwitchUpdate,
    acceptLicense,
    refuseLicense,
    addCommonCondition,
    deleteCommonCondition,
    toggleCurrentFileEnable,
    getFileEnableStatus,
    addReplyItem,
    deleteReplyItem,
    saveCurrent,
    createNewFile,
    deleteCurrentFile,
    downloadCurrentFile,
    uploadFile,
    doImport,
    formatUpdateTime,
  };
}
```

The variables and functions are moved from `reply.vue` without changing business logic.

- [ ] **Step 2: 拆 `ReplyFileSidebar.vue`**

Props:

```ts
const props = defineProps<{
  files: FileInfo[];
  total: number;
  selectedFilename: string;
  query: { page: number; pageSize: number; keyword: string; sortBy: string; sortOrder: string };
  loading: boolean;
  getFileEnableStatus: (filename: string, fallback: boolean) => boolean;
  formatUpdateTime: (ts: number) => string;
}>();
```

Emits:

```ts
const emit = defineEmits<{
  select: [filename: string];
  create: [];
  delete: [];
  download: [];
  upload: [options: UploadCustomRequestOptions];
  'update:query': [query: typeof props.query];
}>();
```

Move file list, search form, upload button, new file button, delete/download buttons and pagination markup from the current sidebar.

- [ ] **Step 3: 拆 `ReplyMetaSection.vue`**

Props:

```ts
const draft = defineModel<ReplyFileDraft>('draft', { required: true });
const props = defineProps<{
  replyEnabled: boolean;
  saving: boolean;
}>();
```

Emits:

```ts
const emit = defineEmits<{
  toggleReplyEnabled: [value: boolean];
  toggleFileEnabled: [];
  save: [];
}>();
```

Move the header form, reply enable switch, current file enable switch, save button and file metadata controls.

- [ ] **Step 4: 拆条件和规则区块**

`ReplyCommonConditionsSection.vue` uses:

```ts
const conditions = defineModel<ReplyCondition[]>('conditions', { required: true });
const props = defineProps<{ page: number; pageSize: number; total: number }>();
const emit = defineEmits<{ add: []; delete: [index: number]; 'update:page': [page: number] }>();
```

`ReplyRulesSection.vue` uses:

```ts
const rules = defineModel<ReplyTask[]>('rules', { required: true });
const props = defineProps<{ page: number; pageSize: number; total: number }>();
const emit = defineEmits<{ add: []; delete: [index: number]; 'update:page': [page: number] }>();
```

Move `ConditionBuilder` and `NestedRuleEditor` usage into these components.

- [ ] **Step 5: 拆弹窗组件**

`ReplyImportModal.vue`:

```ts
const show = defineModel<boolean>('show', { required: true });
const content = defineModel<string>('content', { required: true });
const emit = defineEmits<{ import: [] }>();
```

`ReplyLicenseModal.vue`:

```ts
const show = defineModel<boolean>('show', { required: true });
const emit = defineEmits<{ accept: []; refuse: [] }>();
```

- [ ] **Step 6: 路由页变薄**

`ui/src/pages/mod/reply.vue` becomes:

```vue
<script setup lang="ts">
import CustomReplyEditor from '@/components/custom-reply/CustomReplyEditor.vue';
</script>

<template>
  <CustomReplyEditor />
</template>
```

- [ ] **Step 7: 运行验证并提交**

Run: `cd ui && pnpm run type-check`

Expected: exit 0。

Run: `cd ui && pnpm test -- customReply`

Expected: PASS。

Commit:

```bash
git add ui/src/pages/mod/reply.vue ui/src/components/custom-reply ui/src/features/customReply ui/components.d.ts
git commit --author="PaienNate <1101839859@qq.com>" -m "refactor(custom-reply): 拆分回复编辑器组件"
```

### Task 4: 自定义文案 view model 抽取

**Files:**
- Create: `ui/src/features/customText/viewModel.ts`
- Create: `ui/src/features/customText/viewModel.test.ts`
- Modify: `ui/src/pages/custom-text/[category].vue`

- [ ] **Step 1: 写筛选和导入导出测试**

```ts
import assert from 'node:assert/strict';
import {
  buildCustomTextExportContent,
  createTextItemKeyStore,
  getCustomTextGroups,
  parseCustomTextImportContent,
  sortCustomTextCategory,
} from './viewModel';
import type { TextTemplateHelpDict, TextTemplateItem, TextTemplateWithWeightDict } from './types';

const texts: TextTemplateWithWeightDict = {
  core: {
    diceName: [['SealDice', 1]],
    deprecated: [['old', 1]],
    modified: [['new', 1]],
  },
};

const helpInfo: TextTemplateHelpDict = {
  core: {
    diceName: { subType: '基础 名称', vars: ['$t玩家'] },
    deprecated: { subType: '旧版 条目', notBuiltin: true },
    modified: { subType: '基础 修改', modified: true, topOrder: 10 },
  },
};

assert.deepEqual(getCustomTextGroups(helpInfo.core), ['基础', '旧版']);
assert.equal(sortCustomTextCategory({ texts, helpInfo, category: 'core', filterMode: 'modified' })[0]![1].length, 1);
assert.equal(sortCustomTextCategory({ texts, helpInfo, category: 'core', filterMode: 'deprecated' })[0]![1][0]![0], 'deprecated');

const compact = buildCustomTextExportContent({ texts, category: 'core', onlyCurrent: true, compact: true });
assert.equal(JSON.parse(compact).items.core.diceName[0][0], 'SealDice');

const pretty = buildCustomTextExportContent({ texts, category: 'core', onlyCurrent: false, compact: false });
assert.ok(pretty.includes('\n  '));

const parsed = parseCustomTextImportContent(JSON.stringify({ title: 'x', items: texts }));
assert.deepEqual(parsed.core.diceName, [['SealDice', 1]]);

const keyStore = createTextItemKeyStore();
const item: TextTemplateItem = ['text', 1];
assert.equal(keyStore.keyOf('diceName', item), keyStore.keyOf('diceName', item));
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd ui && pnpm test -- customText/viewModel.test.ts`

Expected: FAIL，错误包含 missing exported functions。

- [ ] **Step 3: 创建 `viewModel.ts`**

导出这些函数和类型：

```ts
export type CustomTextFilterMode = 'all' | 'unmodified' | 'modified' | 'group' | 'deprecated';

export type SortCustomTextCategoryInput = {
  texts: TextTemplateWithWeightDict;
  helpInfo: TextTemplateHelpDict;
  category: string;
  filterMode: CustomTextFilterMode;
  filterName?: string;
  filterGroup?: string;
};

export function getCustomTextGroups(helpGroup: TextTemplateHelpGroup): string[]
export function sortCustomTextCategory(input: SortCustomTextCategoryInput): Array<[string, Array<[string, TextTemplateItem[]]>]>
export function buildCustomTextExportContent(input: { texts: TextTemplateWithWeightDict; category: string; onlyCurrent: boolean; compact: boolean }): string
export function parseCustomTextImportContent(content: string): TextTemplateWithWeightDict
export function createTextItemKeyStore(): { keyOf: (keyName: string, item: TextTemplateItem) => string }
```

- [ ] **Step 4: 更新页面调用**

替换页面里的 `textItemKeys`、`doSort`、`importRefresh` 和导入 JSON 解析逻辑为 view model 调用。

- [ ] **Step 5: 运行测试并提交**

Run: `cd ui && pnpm test -- customText/viewModel.test.ts`

Expected: PASS。

Commit:

```bash
git add ui/src/features/customText/viewModel.ts ui/src/features/customText/viewModel.test.ts ui/src/pages/custom-text/[category].vue
git commit --author="PaienNate <1101839859@qq.com>" -m "refactor(custom-text): 抽取文案视图模型"
```

### Task 5: 自定义文案编辑器组件拆分

**Files:**
- Create: `ui/src/components/custom-text/CustomTextEditor.vue`
- Create: `ui/src/components/custom-text/CustomTextHelp.vue`
- Create: `ui/src/components/custom-text/CustomTextToolbar.vue`
- Create: `ui/src/components/custom-text/CustomTextFilterBar.vue`
- Create: `ui/src/components/custom-text/CustomTextEntryCard.vue`
- Create: `ui/src/components/custom-text/CustomTextImportModal.vue`
- Create: `ui/src/features/customText/useCustomTextEditor.ts`
- Modify: `ui/src/pages/custom-text/[category].vue`

- [ ] **Step 1: 建立 composable 返回契约**

`useCustomTextEditor.ts` exports:

```ts
export function useCustomTextEditor(category: MaybeRefOrGetter<string>) {
  return {
    texts,
    configForImport,
    importOnlyCurrent,
    importImpact,
    dialogImportVisible,
    filterMode,
    filterGroups,
    currentFilterGroup,
    currentFilterName,
    helpInfo,
    previewInfo,
    hasCategory,
    modified,
    sortedCategory,
    customTextQuery,
    saveMutation,
    previewRefreshMutation,
    textItemKeyOf,
    copied,
    importRefresh,
    doImport,
    addItem,
    doChanged,
    removeItem,
    save,
    refreshPreview,
    getPreview,
    getPreviewCheckErr,
    getPreviewInfo,
    askDeleteValue,
    askResetValue,
    handleFilterModeChange,
  };
}
```

Move query, mutations, draft sync, import/export actions and unsaved guard from the route page into this composable.

- [ ] **Step 2: 拆帮助和工具栏组件**

`CustomTextHelp.vue` contains the current `TipBox` help block.

`CustomTextToolbar.vue` contract:

```ts
const keyword = defineModel<string>('keyword', { required: true });
const props = defineProps<{ previewLoading: boolean }>();
const emit = defineEmits<{ refreshPreview: []; openImport: [] }>();
```

- [ ] **Step 3: 拆筛选组件**

`CustomTextFilterBar.vue` contract:

```ts
const mode = defineModel<CustomTextFilterMode>('mode', { required: true });
const group = defineModel<string>('group', { required: true });
const props = defineProps<{ groups: string[] }>();
```

The component owns `filterModes` labels:

```ts
const filterModes = [
  { value: 'all', desc: '全部' },
  { value: 'unmodified', desc: '默认文案' },
  { value: 'modified', desc: '修改过' },
  { value: 'group', desc: '指定分组' },
  { value: 'deprecated', desc: '旧版文本' },
] satisfies Array<{ value: CustomTextFilterMode; desc: string }>;
```

- [ ] **Step 4: 拆文案条目组件**

`CustomTextEntryCard.vue` contract:

```ts
const items = defineModel<TextTemplateItem[]>('items', { required: true });
const props = defineProps<{
  category: string;
  keyName: string;
  help?: Value;
  preview?: Record<string, TextItemCompatibleInfo>;
  textItemKeyOf: (keyName: string, item: TextTemplateItem) => string;
}>();
const emit = defineEmits<{
  addItem: [keyName: string];
  removeItem: [items: TextTemplateItem[], index: number];
  change: [category: string, keyName: string];
  deleteKey: [category: string, keyName: string];
  resetKey: [category: string, keyName: string];
}>();
```

Move the current `n-form-item` block for each `keyName` into this component.

- [ ] **Step 5: 拆导入导出组件**

`CustomTextImportModal.vue` contract:

```ts
const show = defineModel<boolean>('show', { required: true });
const content = defineModel<string>('content', { required: true });
const onlyCurrent = defineModel<boolean>('onlyCurrent', { required: true });
const compact = defineModel<boolean>('compact', { required: true });
const props = defineProps<{ saving: boolean }>();
const emit = defineEmits<{ copy: []; clear: []; import: [] }>();
```

- [ ] **Step 6: 路由页变薄**

`ui/src/pages/custom-text/[category].vue` becomes:

```vue
<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import CustomTextEditor from '@/components/custom-text/CustomTextEditor.vue';

const props = defineProps<{ category?: string }>();
const route = useRoute();
const category = computed(() => {
  const routeParams = route.params as Record<string, string | string[] | undefined>;
  const routeCategory = routeParams.category;
  const fallback = Array.isArray(routeCategory) ? routeCategory[0] : routeCategory;
  return props.category ?? String(fallback ?? '');
});
</script>

<template>
  <CustomTextEditor :category="category" />
</template>
```

- [ ] **Step 7: 运行验证并提交**

Run: `cd ui && pnpm run type-check`

Expected: exit 0。

Run: `cd ui && pnpm test -- customText`

Expected: PASS。

Commit:

```bash
git add ui/src/pages/custom-text ui/src/components/custom-text ui/src/features/customText ui/components.d.ts
git commit --author="PaienNate <1101839859@qq.com>" -m "refactor(custom-text): 拆分文案编辑器组件"
```

### Task 6: 最终清理与整体验证

**Files:**
- Modify: files changed by Tasks 1-5

- [ ] **Step 1: 检查路由页行数**

Run:

```bash
wc -l ui/src/pages/mod/reply.vue 'ui/src/pages/custom-text/[category].vue'
```

Expected:

```text
  10 ui/src/pages/mod/reply.vue
  22 ui/src/pages/custom-text/[category].vue
```

Line counts can differ by imports and formatting, but both route files must stay below 80 lines.

- [ ] **Step 2: 全量类型检查**

Run: `cd ui && pnpm run type-check`

Expected: exit 0。

- [ ] **Step 3: 全量前端测试**

Run: `cd ui && pnpm test`

Expected: all tests PASS。

- [ ] **Step 4: Go 回归测试**

Run: `go test ./...`

Expected: exit 0。

- [ ] **Step 5: 空白和冲突标记检查**

Run: `git diff --check`

Expected: exit 0。

- [ ] **Step 6: 提交最终清理**

If `components.d.ts` or small import/style cleanup remains:

```bash
git add ui/components.d.ts ui/src/pages/mod/reply.vue ui/src/pages/custom-text ui/src/components/custom-reply ui/src/components/custom-text ui/src/features/customReply ui/src/features/customText
git commit --author="PaienNate <1101839859@qq.com>" -m "chore(ui): 清理编辑器重构遗留声明"
```

Skip this commit if `git status --short` is empty after Task 5.

### 验收标准

- `reply.vue` and `custom-text/[category].vue` are route-level composition surfaces, not feature implementations.
- All moved domain logic has direct unit tests.
- No API request path, query key, mutation payload or route path changes.
- Unsaved changes guard still protects both editors.
- File upload/download/import/save flows remain wired to existing API clients.
- Search/filter state stays deterministic after route changes and reset actions.
- Responsive behavior is at least no worse than the current implementation; browser manual responsive验收 is a later task, not part of this plan.
