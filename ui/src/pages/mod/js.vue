<template>
  <main class="js-page">
    <PageHeader title="JS 扩展">
      <n-flex align="center" justify="space-between" wrap>
        <n-flex align="center" size="small">
          <n-text depth="3">启用 JS 扩展</n-text>
          <n-switch
            :value="jsEnable"
            :disabled="isTestMode"
            :loading="jsSwitchBusy"
            aria-label="启用 JS 扩展"
            @update:value="handleJsEnableToggle"
          />
        </n-flex>
        <n-button v-show="jsEnable" type="primary" :disabled="isTestMode" @click="handleReload">
          <template #icon>
            <n-icon><i-tabler-refresh /></n-icon>
          </template>
          重载 JS
        </n-button>
      </n-flex>
    </PageHeader>

    <ReloadNotice :show="needReload" />

    <n-affix v-if="jsConfigEdited" :top="70">
      <TipBox type="warning">
        <n-flex wrap>
          <n-text type="warning" tag="strong" class="text-base"
            >配置内容已修改，不要忘记保存！</n-text
          >
          <n-button
            type="primary"
            secondary
            :disabled="!jsConfigEdited || isTestMode"
            @click="saveJsConfig"
          >
            <template #icon>
              <n-icon><i-tabler-device-floppy /></n-icon>
            </template>
            点我保存
          </n-button>
        </n-flex>
      </TipBox>
    </n-affix>

    <n-tabs v-model:value="tab" pane-class="mb-8" justify-content="space-evenly" class="js-tabs">
      <n-tab-pane tab="控制台" name="console">
        <section class="js-console-grid">
          <section class="js-panel js-editor-panel">
            <header class="js-panel-header">
              <n-flex align="center" justify="space-between" wrap>
                <n-text class="js-panel-title">JS 扩展执行环境</n-text>
                <n-button
                  type="primary"
                  secondary
                  :disabled="!jsEnable || jsRunning || isTestMode"
                  @click="doExecute"
                >
                  <template #icon
                    ><n-icon><i-tabler-player-play /></n-icon
                  ></template>
                  执行代码
                </n-button>
              </n-flex>
            </header>

            <div class="js-panel-body js-editor-body">
              <CodeMirror
                v-if="editorReady"
                v-model="code"
                class="js-editor"
                :extensions="editorExtensions as never[]"
                :dark="editorDark"
                :wrap="true"
              />
              <n-skeleton v-else text :repeat="8" />
            </div>

            <footer class="js-panel-footer">
              <n-text type="warning" tag="p" class="js-console-tip">
                注意：延迟执行的代码，其输出不会立即出现
              </n-text>
            </footer>
          </section>

          <section class="js-panel js-output-panel">
            <header class="js-panel-header">
              <n-flex align="center" justify="space-between" wrap>
                <div class="js-panel-heading">
                  <n-text class="js-panel-title">运行日志</n-text>
                </div>
                <n-button secondary :disabled="!jsLines.length" @click="clearLogs">
                  <template #icon
                    ><n-icon><i-tabler-trash /></n-icon
                  ></template>
                  清空日志
                </n-button>
              </n-flex>
            </header>

            <div class="js-panel-body js-output-body">
              <n-log ref="logRef" class="js-output-log" :lines="jsLines" :rows="24" trim />
            </div>
          </section>
        </section>
      </n-tab-pane>

      <n-tab-pane tab="插件列表" name="list">
        <JsListView @mark-need-reload="handleMarkNeedReload" />
      </n-tab-pane>

      <n-tab-pane tab="插件设置" name="config">
        <JsConfigView ref="jsConfigViewRef" @dirty-change="handleConfigDirtyChange" />
      </n-tab-pane>

      <n-tab-pane tab="数据管理" name="data">
        <JsDataView />
      </n-tab-pane>
    </n-tabs>
  </main>
</template>

<script setup lang="tsx">
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  watch,
} from 'vue';
import { useQuery } from '@tanstack/vue-query';
import {
  NButton,
  NFlex,
  NIcon,
  NLog,
  NTabs,
  NTabPane,
  NText,
  NSwitch,
  useMessage,
  type LogInst,
} from 'naive-ui';
import {
  getSdApiV2JsRecord,
  getSdApiV2JsStatusOptions,
  postSdApiV2JsExecute,
  postSdApiV2JsReload,
  postSdApiV2JsShutdown,
} from '@/api';
import ReloadNotice from '@/components/layout/ReloadNotice.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { hasAccessToken } from '@/features/auth/state';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  useTestMode,
} from '@/features/testMode/state';
import JsListView from '@/components/js/JsListView.vue';
import JsConfigView from '@/components/js/JsConfigView.vue';
import JsDataView from '@/components/js/JsDataView.vue';
import { useAppTheme } from '@/features/theme';

const CodeMirror = defineAsyncComponent(() => import('vue-codemirror6'));

// JS 扩展页分为四个工作区：
// list 管理脚本文件，config 管理插件配置，data 管理插件 KV 存储，
// console 用于临时执行代码和查看输出。
const message = useMessage();
const { isTestMode } = useTestMode();
const tab = ref<string>('list');

const defaultText = [
  '// 学习制作可以看这里：https://github.com/sealdice/javascript/tree/main/examples',
  '// 下载插件可以看这里：https://github.com/sealdice/javascript/tree/main/scripts',
  '// 使用 TypeScript，编写更容易 https://github.com/sealdice/javascript/tree/main/examples_ts',
  '// 目前可用于：创建自定义指令，自定义 COC 房规，发送网络请求，读写本地数据',
  '',
  "console.log('这是测试控制台');",
  "console.log('可以这样来查看变量详情：');",
  'console.log(Object.keys(seal));',
  "console.log('更多内容正在制作中...')",
  "console.log('注意：测试版！API 仍然可能发生重大变化！')",
  '// 写在这里的所有变量都是临时变量，如果你希望全局变量，使用 globalThis',
  '// 但是注意，全局变量在进程关闭后失效，想保存状态请存入硬盘。',
  'globalThis._test = 123;',
  '',
  "let ext = seal.ext.find('test');",
  'if (!ext) {',
  "  ext = seal.ext.new('test', '木落', '1.0.0');",
  '  seal.ext.register(ext);',
  '}',
];

const code = ref<string>(defaultText.join('\n'));

const jsLines = ref<string[]>([]);
const jsRunning = ref(false);
const jsEnable = ref(false);
const jsSwitchBusy = ref(false);
const needReload = ref(false);
const jsConfigEdited = ref(false);
const jsConfigViewRef = ref<InstanceType<typeof JsConfigView> | null>(null);
const logRef = ref<LogInst | null>(null);
const editorBaseExtensions = shallowRef<unknown[]>([]);
const editorDarkTheme = shallowRef<unknown | null>(null);
const { resolvedTheme } = useAppTheme();
const editorReady = computed(() => editorBaseExtensions.value.length > 0);
const editorDark = computed(() => resolvedTheme.value === 'dark');
const editorExtensions = computed(() => {
  if (!editorBaseExtensions.value.length) return [];
  if (editorDark.value && editorDarkTheme.value) {
    return [...editorBaseExtensions.value, editorDarkTheme.value];
  }
  return [...editorBaseExtensions.value];
});

// JS 总开关状态来自后端；切换时通过 reload/shutdown 表达实际语义，
// 而不是只在前端改开关显示。
const statusQuery = useQuery({
  ...getSdApiV2JsStatusOptions(),
  enabled: hasAccessToken,
});

let recordTimer: ReturnType<typeof setInterval> | null = null;

watch(
  () => statusQuery.data.value?.item?.status,
  status => {
    jsEnable.value = status === true;
  },
  { immediate: true }
);

function appendLogLines(lines: string[]) {
  const cleaned = lines.filter(Boolean);
  if (!cleaned.length) return;
  jsLines.value.push(...cleaned);
}

function appendLogLine(line: string) {
  appendLogLines([line]);
}

function appendExecutionSeparator() {
  const stamp = new Date().toLocaleString('zh-CN', { hour12: false });
  appendLogLine(`======= ${stamp} =======`);
}

function clearLogs() {
  jsLines.value = [];
}

async function doExecute() {
  if (!code.value.trim()) return;
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  jsRunning.value = true;
  appendExecutionSeparator();
  try {
    const { data } = await postSdApiV2JsExecute({
      body: { value: code.value },
      throwOnError: true,
    });
    const item = data.item;
    if (item.outputs?.length) {
      appendLogLines(item.outputs);
    }
    if (item.err) {
      appendLogLine(`[Error] ${item.err}`);
    } else if (item.ret !== undefined && item.ret !== null) {
      appendLogLine(String(item.ret));
    }
  } catch (error) {
    if (isTestModeApiError(error)) {
      appendLogLine(`[Error] ${getTestModeBlockMessage(error)}`);
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    appendLogLine('[Error] 执行失败');
    message.error('执行失败');
  } finally {
    jsRunning.value = false;
  }
}

function startRecordPolling() {
  if (recordTimer) clearInterval(recordTimer);
  recordTimer = setInterval(async () => {
    try {
      const { data } = await getSdApiV2JsRecord({ throwOnError: true });
      appendLogLines(data.item.outputs ?? []);
    } catch {
      // ignore polling errors
    }
  }, 3000);
}

async function handleReload(): Promise<boolean> {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return false;
  }
  try {
    await postSdApiV2JsReload({ throwOnError: true });
    message.success('已重载');
    needReload.value = false;
    await statusQuery.refetch();
    return true;
  } catch (error) {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return false;
    }
    message.error('重载失败');
    await statusQuery.refetch();
    return false;
  }
}

async function handleShutdown(): Promise<boolean> {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return false;
  }
  try {
    await postSdApiV2JsShutdown({ throwOnError: true });
    jsLines.value = [];
    jsEnable.value = false;
    message.success('已关闭 JS 支持');
    await statusQuery.refetch();
    return true;
  } catch (error) {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return false;
    }
    message.error('关闭失败');
    await statusQuery.refetch();
    return false;
  }
}

async function handleJsEnableToggle(value: boolean) {
  if (value === jsEnable.value) return;
  jsSwitchBusy.value = true;
  try {
    if (value) {
      await handleReload();
    } else {
      await handleShutdown();
    }
  } finally {
    jsSwitchBusy.value = false;
  }
}

function handleMarkNeedReload() {
  needReload.value = true;
}

function handleConfigDirtyChange(value: boolean) {
  jsConfigEdited.value = value;
}

async function saveJsConfig() {
  if (!jsConfigViewRef.value) return;
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  await jsConfigViewRef.value.saveAll();
}

async function loadEditorExtensions() {
  // CodeMirror 体积较大，仅进入 console tab 时加载，避免拖慢 JS 管理页首屏。
  if (editorBaseExtensions.value.length) return;
  const [{ basicSetup }, { oneDark }, { javascript }] = await Promise.all([
    import('codemirror'),
    import('@codemirror/theme-one-dark'),
    import('@codemirror/lang-javascript'),
  ]);
  editorBaseExtensions.value = [basicSetup, javascript()];
  editorDarkTheme.value = oneDark;
}

watch(
  tab,
  value => {
    if (value === 'console') {
      void loadEditorExtensions();
    }
  },
  { immediate: true }
);

watch(
  () => jsLines.value.length,
  async () => {
    await nextTick();
    logRef.value?.scrollTo({ position: 'bottom', silent: true });
  }
);

onMounted(async () => {
  startRecordPolling();
});

onBeforeUnmount(() => {
  if (recordTimer) clearInterval(recordTimer);
});
</script>

<style scoped>
.js-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.js-console-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 0.92fr);
  gap: 1.25rem;
  align-items: stretch;
}

.js-panel {
  display: flex;
  min-height: 36rem;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}

.js-panel-header {
  padding: 0.9rem 1rem;
  border-bottom: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated-soft);
}

.js-panel-heading {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.1rem;
}

.js-panel-title {
  font-size: 0.97rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--sd-text-primary);
}

.js-panel-body {
  display: flex;
  min-height: 0;
  flex: 1;
}

.js-editor-body,
.js-output-body {
  padding: 0.85rem;
}

.js-editor-section {
  min-width: auto;
}

.js-editor {
  width: 100%;
  min-height: 100%;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-page);
}

.js-editor :deep(.cm-editor) {
  min-height: 100%;
  color: var(--sd-text-primary);
  background: var(--sd-bg-page);
  font-family: var(--sd-font-code);
}

.js-editor :deep(.cm-scroller) {
  font-family: var(--sd-font-code);
}

.js-editor :deep(.cm-content),
.js-editor :deep(.cm-gutters) {
  font-family: var(--sd-font-code);
}

.js-editor :deep(.cm-gutters) {
  color: var(--sd-text-muted);
  background: var(--sd-bg-elevated-soft);
  background: color-mix(in srgb, var(--sd-bg-elevated-soft), var(--sd-bg-page) 25%);
  border-right: 1px solid var(--sd-border-soft);
}

.js-editor :deep(.cm-activeLine),
.js-editor :deep(.cm-activeLineGutter) {
  background: var(--sd-bg-hover);
}

.js-editor :deep(.cm-selectionBackground),
.js-editor :deep(.cm-content ::selection) {
  background: var(--sd-bg-selected);
}

.js-editor :deep(.cm-cursor),
.js-editor :deep(.cm-dropCursor) {
  border-left-color: var(--sd-text-primary);
}

.js-console-tip {
  margin: 0;
  padding: 0;
}

.js-output-log {
  width: 100%;
  min-height: 100%;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-page);
  font-family: var(--sd-font-code);
}

.js-output-log :deep(.n-log) {
  background: transparent;
}

.js-output-log :deep(.n-scrollbar-container) {
  border-radius: var(--sd-radius-md);
}

.js-output-log :deep(.n-log-line) {
  color: var(--sd-text-primary);
}

.js-panel-footer {
  padding: 0 1rem 0.95rem;
}

@media (max-width: 960px) {
  .js-console-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .js-panel {
    min-height: 28rem;
  }

  .js-panel-header :deep(.n-flex) {
    align-items: flex-start;
  }
}

@media (max-width: 640px) {
  .js-page {
    padding: 0;
  }

  .js-tabs :deep(.n-tabs-nav-scroll-content) {
    min-width: max-content;
    justify-content: flex-start !important;
  }

  .js-panel-header :deep(.n-flex) {
    flex-wrap: wrap;
    gap: 0.75rem;
  }
}
</style>
