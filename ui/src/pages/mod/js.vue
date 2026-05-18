<script setup lang="tsx">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { NButton, NFlex, NIcon, NTabs, NTabPane, NText, NSwitch, useMessage } from 'naive-ui';
import {
  getSdApiV2JsRecord,
  getSdApiV2JsStatusOptions,
  postSdApiV2JsExecute,
  postSdApiV2JsReload,
  postSdApiV2JsShutdown,
} from '@/api';
import TipBox from '@/components/shared/TipBox.vue';
import { hasAccessToken } from '@/features/auth/state';
import JsListView from '@/components/js/JsListView.vue';
import JsConfigView from '@/components/js/JsConfigView.vue';
import JsDataView from '@/components/js/JsDataView.vue';

const CodeMirror = defineAsyncComponent(() => import('vue-codemirror6'));

// JS 扩展页分为四个工作区：
// list 管理脚本文件，config 管理插件配置，data 管理插件 KV 存储，
// console 用于临时执行代码和查看输出。
const message = useMessage();
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
const editorExtensions = shallowRef<unknown[]>([]);
const editorReady = computed(() => editorExtensions.value.length > 0);

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
  { immediate: true },
);

async function doExecute() {
  if (!code.value.trim()) return;
  jsRunning.value = true;
  jsLines.value = [];
  try {
    const { data } = await postSdApiV2JsExecute({
      body: { body: { value: code.value } },
      throwOnError: true,
    });
    const item = data.item;
    if (item.outputs?.length) {
      jsLines.value.push(...item.outputs);
    }
    if (item.err) {
      jsLines.value.push(`[Error] ${item.err}`);
    } else if (item.ret !== undefined && item.ret !== null) {
      jsLines.value.push(String(item.ret));
    }
  } catch {
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
      const outputs = data.item.outputs ?? [];
      for (const line of outputs) {
        if (line) {
          jsLines.value.push(line);
        }
      }
    } catch {
      // ignore polling errors
    }
  }, 3000);
}

async function handleReload() {
  try {
    await postSdApiV2JsReload({ throwOnError: true });
    message.success('已重载');
    needReload.value = false;
    statusQuery.refetch();
  } catch {
    message.error('重载失败');
  }
}

async function handleShutdown() {
  try {
    await postSdApiV2JsShutdown({ throwOnError: true });
    jsLines.value = [];
    jsEnable.value = false;
    message.success('已关闭 JS 支持');
    statusQuery.refetch();
  } catch {
    message.error('关闭失败');
  }
}

async function handleJsEnableToggle(value: boolean) {
  if (value === jsEnable.value) return;
  jsSwitchBusy.value = true;
  try {
    if (value) {
      await handleReload();
      jsEnable.value = true;
    } else {
      await handleShutdown();
      jsEnable.value = false;
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
  await jsConfigViewRef.value.saveAll();
}

async function loadEditorExtensions() {
  // CodeMirror 体积较大，仅进入 console tab 时加载，避免拖慢 JS 管理页首屏。
  if (editorExtensions.value.length) return;
  const [{ basicSetup }, { oneDark }, { javascript }] = await Promise.all([
    import('codemirror'),
    import('@codemirror/theme-one-dark'),
    import('@codemirror/lang-javascript'),
  ]);
  editorExtensions.value = [basicSetup, oneDark, javascript()];
}

watch(tab, value => {
  if (value === 'console') {
    void loadEditorExtensions();
  }
}, { immediate: true });

onMounted(async () => {
  startRecordPolling();
});

onBeforeUnmount(() => {
  if (recordTimer) clearInterval(recordTimer);
});
</script>

<template>
  <main class="js-page">
    <header class="page-header">
      <n-flex align="center" justify="space-between" wrap>
        <n-switch
          :value="jsEnable"
          :loading="jsSwitchBusy"
          @update:value="handleJsEnableToggle"
        >
          <template #checked>启用</template>
          <template #unchecked>关闭</template>
        </n-switch>
        <n-button v-show="jsEnable" type="primary" @click="handleReload">
          <template #icon>
            <n-icon><i-carbon-renew /></n-icon>
          </template>
          重载 JS
        </n-button>
      </n-flex>
    </header>

    <n-affix v-if="needReload" :top="60">
      <TipBox type="error">
        <n-text type="error" class="text-base" tag="strong">存在修改，需要重载后生效！</n-text>
      </TipBox>
    </n-affix>

    <n-affix v-if="jsConfigEdited" :top="70">
      <TipBox type="error">
        <n-flex>
          <n-text type="error" tag="strong" class="text-base">配置内容已修改，不要忘记保存！</n-text>
          <n-button type="info" secondary :disabled="!jsConfigEdited" @click="saveJsConfig">
            <template #icon>
              <n-icon><i-carbon-save /></n-icon>
            </template>
            点我保存
          </n-button>
        </n-flex>
      </TipBox>
    </n-affix>

    <n-tabs v-model:value="tab" pane-class="mb-8" justify-content="space-evenly">
      <n-tab-pane tab="控制台" name="console">
        <header class="js-console-header">
          <n-flex align="center" justify="space-between">
            <n-text>JS 扩展执行环境</n-text>
            <n-flex size="small">
              <n-button type="info" secondary :disabled="!jsEnable || jsRunning" @click="doExecute">
                <template #icon><n-icon><i-carbon-play /></n-icon></template>
                执行代码
              </n-button>
            </n-flex>
          </n-flex>
        </header>

        <section class="js-editor-section">
          <CodeMirror
            v-if="editorReady"
            v-model="code"
            class="js-editor"
            :extensions="editorExtensions as never[]"
            :wrap="true"
          />
          <n-skeleton v-else text :repeat="8" />
          <n-text type="error" tag="p" class="js-console-tip">
            注意：延迟执行的代码，其输出不会立即出现
          </n-text>
        </section>

        <section class="js-output-section">
          <n-text depth="3" class="mb-2">输出</n-text>
          <div class="js-output-lines">
            <p v-for="(line, idx) in jsLines" :key="idx" class="js-output-line">
              <n-text code>{{ line }}</n-text>
            </p>
            <n-text v-if="!jsLines.length" depth="3">暂无输出</n-text>
          </div>
        </section>
      </n-tab-pane>

      <n-tab-pane tab="插件列表" name="list">
        <JsListView @mark-need-reload="handleMarkNeedReload" />
      </n-tab-pane>

      <n-tab-pane tab="插件设置" name="config">
        <JsConfigView
          ref="jsConfigViewRef"
          @dirty-change="handleConfigDirtyChange"
        />
      </n-tab-pane>

      <n-tab-pane tab="数据管理" name="data">
        <JsDataView />
      </n-tab-pane>
    </n-tabs>
  </main>
</template>

<style scoped>
.js-page {
  padding: 0 1rem;
}

.page-header {
  margin-bottom: 1rem;
}

.js-console-header {
  margin-bottom: 1rem;
}

.js-editor-section {
  margin-bottom: 1rem;
}

.js-editor {
  min-height: 20rem;
  border: 1px solid var(--sd-border);
}

.js-execute-btn {
  margin-top: 0.5rem;
}

.js-console-tip {
  padding: 1rem 0;
}

.js-output-section {
  margin-top: 1rem;
}

.js-output-lines {
  max-height: 30rem;
  overflow-y: auto;
  padding: 0.5rem;
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
}

.js-output-line {
  margin: 0.15rem 0;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
