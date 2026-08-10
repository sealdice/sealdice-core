# 应用壳、页面宽度与备份 Tab 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task with checkpoints.

**Goal:** 让路由切换复用同一个应用壳，统一阅读型页面的居中宽度，并将备份页改为两个一级 Tab。

**Architecture:** `App.vue` 保持一个稳定的 `AppShell`，根据路由元信息传递 `contentMode` 和 `containerMode`；全局认证弹窗放在路由内容之外，默认登录只在根组件启动。`default` 模式实现 1180px 居中列，`wide/workspace` 保持全宽语义。备份页面保留现有查询和 mutation 编排，只将两个展示组件放入线型 Tab，并移除外层 Card。

**Tech Stack:** Vue 3 Composition API、TypeScript、Vue Router、Pinia auth store、Naive UI、Vue Query。

---

### Task 1: 稳定应用壳与布局语义

**Files:**
- Modify: `ui/src/App.vue`
- Modify: `ui/src/components/app-shell/AppShell.vue`
- Modify: `ui/src/components/app-shell/appShellLayout.ts`
- Modify: `ui/src/router/navigation.ts`
- Modify: `ui/src/router/types.ts` only if the route mode mapping requires a type adjustment

- [ ] **Step 1: Replace dynamic shell layout components with one stable shell branch.**

  In `App.vue`, keep the `PlainLayout` branch for `route.meta.layout === 'plain'`. For every other route render one imported `AppShell` and pass:

  ```ts
  const shellContentMode = computed(() => route.meta.layout === 'wide' || route.meta.layout === 'workspace' ? 'wide' : 'default');
  const shellContainerMode = computed(() => route.meta.layout === 'workspace' ? 'workspace' : 'default');
  ```

  Keep the existing keyed page transition inside this stable shell. Render `AppUnlockDialog` beside `RouterView`, inside the existing Naive UI providers.

- [ ] **Step 2: Move default sign-in bootstrap to the application root.**

  Call `void authSession.tryDefaultSignin()` once from `App.vue` setup. Remove the `useAuthSession` import and the corresponding call from `AppShell.vue`; retain shell-local unsaved-change, search, and sidebar behavior.

- [ ] **Step 3: Implement the default content column.**

  Update `AppShell.vue` so `.sd-main-container` has `width: min(100%, 1180px)` and `margin-inline: auto`; preserve the existing mobile padding. Leave `.sd-main-container--wide` at full available width. Keep `.sd-page-shell--workspace` flex/min-height rules unchanged.

- [ ] **Step 4: Align route metadata with the content intent.**

  In `ui/src/router/navigation.ts`, set these to `layout: 'default'`: `/mod/deck`, `/mod/package`, `/mod/story`, `/mod/js`, `/mod/helpdoc`, `/mod/censor`, `/misc/base-setting`, `/misc/backup`, `/misc/group`, `/misc/ban`, and `/misc/advanced-setting`. Keep `/mod/reply` and `/tool/test` as `workspace`; keep `/misc/dice-public` as `wide`.

- [ ] **Step 5: Review the route mapping and shell mount hierarchy.**

  Run:

  ```bash
  rg -n 'AppUnlockDialog|tryDefaultSignin|layouts\[|<AppShell|layout:' ui/src/App.vue ui/src/components/app-shell/AppShell.vue ui/src/router/navigation.ts
  ```

  Expected: one `AppUnlockDialog` and one `tryDefaultSignin` under `App.vue`; no authentication bootstrap under `AppShell.vue`; normal route changes use one shell component.

### Task 2: Convert backup page cards into line tabs

**Files:**
- Modify: `ui/src/pages/misc/backup.vue`
- Modify: `ui/src/components/backup/BackupConfigPanel.vue`
- Modify: `ui/src/components/backup/BackupFileList.vue`

- [ ] **Step 1: Add local active tab state and line tabs in the page.**

  Add `const activeTab = shallowRef<'settings' | 'files'>('settings')` in `backup.vue`. Replace `.backup-page__grid` with:

  ```vue
  <n-tabs v-model:value="activeTab" type="line" animated>
    <n-tab-pane name="settings" tab="备份设置">...</n-tab-pane>
    <n-tab-pane name="files" tab="备份文件">...</n-tab-pane>
  </n-tabs>
  ```

  Keep the existing error alerts, queries, mutations, dialogs, and event bindings unchanged.

- [ ] **Step 2: Remove card wrappers while preserving panel contracts.**

  Change the root of `BackupConfigPanel.vue` and `BackupFileList.vue` from `n-card` to `section` with stable classes. Move the existing card header content into a section header class and preserve all props, emits, buttons, loading states, and table behavior.

- [ ] **Step 3: Replace grid-only styles with centered tab content spacing.**

  Remove `.backup-page__grid` grid columns. Add styles for the Tab content and section headers so titles, actions, form, and table retain consistent vertical rhythm. Keep the existing mobile header stacking rules and do not add nested cards.

- [ ] **Step 4: Check the backup data-flow boundaries.**

  Run:

  ```bash
  rg -n 'BackupConfigPanel|BackupFileList|activeTab|backup-page__grid|n-tabs|n-card' ui/src/pages/misc/backup.vue ui/src/components/backup
  ```

  Expected: the page owns Tab state; child components retain only presentation and command events; no two-column backup grid remains.

### Task 3: Verification and handoff

**Files:**
- No new test files, per the user's explicit request to run tests themselves.

- [ ] **Step 1: Format only touched Vue/TypeScript files.**

  Run Prettier on the touched files and use `gofmt` only if a Go file is unexpectedly touched.

- [ ] **Step 2: Run frontend type checking.**

  Run `cd ui && pnpm run type-check`.

- [ ] **Step 3: Run static repository checks.**

  Run `git diff --check`, `git ls-files -u`, and `rg -n '^(<<<<<<<|=======|>>>>>>>)' ui/src`.

- [ ] **Step 4: Review the final diff.**

  Confirm only the application-shell/layout/backup files and the design/plan documents are part of this task; preserve unrelated dirty worktree changes.
