<template>
  <section class="story-painter-viewer">
    <header class="viewer-header">
      <n-flex align="center" justify="space-between" wrap>
        <div class="viewer-title">
          <n-button quaternary @click="emit('back')">
            <template #icon>
              <n-icon><i-carbon-chevron-left /></n-icon>
            </template>
            返回列表
          </n-button>
          <n-text tag="strong" class="viewer-title-text">{{ title }}</n-text>
        </div>
        <n-flex size="small" wrap>
          <n-button type="primary" secondary :loading="loading" @click="loadLog">
            <template #icon><n-icon><i-carbon-renew /></n-icon></template>
            重新加载
          </n-button>
          <n-button type="primary" text @click="painter.refreshSwatches">
            刷新色板
          </n-button>
        </n-flex>
      </n-flex>
    </header>

    <n-alert v-if="errorText" type="error" class="viewer-alert">
      {{ errorText }}
    </n-alert>

    <n-spin :show="loading">
      <div class="viewer-grid">
        <aside class="viewer-side">
          <StoryPainterOptionsPanel v-model:options="exportOptionsModel" v-model:editor-highlight="editorHighlightModel" />
          <StoryPainterCharacterPanel
            :chars="painter.chars.value"
            :swatches="painter.swatches.value"
            :disabled="loading"
            @update-char="painter.updateChar"
            @rename-char="painter.renameChar"
            @delete-char="painter.deleteChar"
          />
        </aside>

        <main class="viewer-main">
          <section class="viewer-toolbar">
            <n-flex align="center" justify="space-between" wrap>
              <n-flex size="small" align="center" wrap>
                <n-checkbox
                  v-for="item in modeOptions"
                  :key="item.value"
                  :checked="painter.mode.value === item.value"
                  :border="true"
                  @click="openMode(item.value)"
                >
                  {{ item.label }}
                </n-checkbox>
                <n-button
                  v-if="painter.mode.value !== 'editor'"
                  size="small"
                  type="primary"
                  text
                  @click="showEditor"
                >
                  编辑器
                </n-button>
              </n-flex>

              <n-flex size="small" wrap>
                <n-button size="small" type="primary" secondary :loading="exportBusy.raw" @click="withBusy('raw', exportRaw)">
                  下载原始文件
                </n-button>
                <n-button size="small" type="primary" secondary :loading="exportBusy.rawText" @click="withBusy('rawText', exportRawText)">
                  下载原始文本
                </n-button>
                <n-button size="small" type="primary" secondary :loading="exportBusy.doc" @click="withBusy('doc', () => exportDoc(false))">
                  下载带图 doc
                </n-button>
                <n-button size="small" type="primary" secondary :loading="exportBusy.talkDoc" @click="withBusy('talkDoc', () => exportDoc(true))">
                  下载对话 doc
                </n-button>
                <n-button
                  size="small"
                  type="primary"
                  secondary
                  :loading="exportBusy.docx"
                  :disabled="!docxSupported"
                  @click="withBusy('docx', exportDocx)"
                >
                  下载 docx
                </n-button>
                <n-dropdown
                  trigger="click"
                  :options="[
                    { label: '保存论坛代码', key: 'bbs' },
                    { label: '保存回声工坊', key: 'trg' },
                  ]"
                  @select="(key: string) => key === 'bbs' ? withBusy('forum', exportForumText) : withBusy('trg', exportTrgText)"
                >
                  <n-button size="small" type="primary" secondary>
                    更多导出
                  </n-button>
                </n-dropdown>
              </n-flex>
            </n-flex>
          </section>

          <StoryPainterEditor
            v-if="painter.mode.value === 'editor'"
            :items="editorItems"
            :chars="painter.chars.value"
            :options="painter.exportOptions"
            :highlight="painter.editorHighlight.value"
            :lazy="!painter.fullItemsLoaded.value"
            @load-full="showEditor"
            @update-items="painter.setAllItemsFromEditor"
          />
          <StoryPainterPreview
            v-else
            :mode="painter.mode.value"
            :items="painter.previewItems.value"
            :item-indexes="painter.visibleIndexes.value"
            :read-item="painter.readPreviewItem"
            :read-items="painter.readPreviewItems"
            :chars="painter.chars.value"
            :options="painter.exportOptions"
            :forum-options="painter.forumOptions"
            :add-voice-mark="painter.trgIsAddVoiceMark.value"
            :copy-text="buildCopyText"
            @copy="handleCopy"
            @update-add-voice-mark="value => { painter.trgIsAddVoiceMark.value = value; }"
            @update-forum-options="value => { Object.assign(painter.forumOptions, value); }"
          />
        </main>
      </div>
    </n-spin>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, shallowRef } from 'vue';
import type { StoryLogView } from '@/api';
import { copyText } from '@/features/clipboard';
import { getStoryPainterAdvancedModeSupport } from '@/features/storyPainter/compat';
import { fetchStoryLogParquet } from '@/features/storyPainter/api';
import { createStoryPainterParquetDataset } from '@/features/storyPainter/parquetDataset';
import { useStoryPainter } from '@/features/storyPainter/useStoryPainter';
import { renderPreviewHtml, renderRawText } from '@/features/storyPainter/renderers';
import { collectStoryPainterForumText, collectStoryPainterTrgText } from '@/features/storyPainter/textExport';
import {
  exportStoryPainterDoc,
  exportStoryPainterDocx,
  extractMessageLines,
  readElementColor,
  saveStoryPainterBlob,
  saveStoryPainterText,
  supportsStoryPainterDocxExport,
  type StoryPainterDocxEntry,
} from '@/features/storyPainter/exporter';
import type { StoryPainterOptions as StoryPainterOptionsModel, StoryPainterPreviewDisplayMode } from '@/features/storyPainter/types';
import { toggleStoryPainterMode } from '@/features/storyPainter/viewMode';
import StoryPainterCharacterPanel from './StoryPainterCharacterPanel.vue';
import StoryPainterEditor from './StoryPainterEditor.vue';
import StoryPainterOptionsPanel from './StoryPainterOptions.vue';
import StoryPainterPreview from './StoryPainterPreview.vue';

const props = defineProps<{
  log: StoryLogView;
}>();

const emit = defineEmits<{
  back: [];
}>();

const message = useMessage();
const painter = useStoryPainter();
const exportOptionsModel = computed<StoryPainterOptionsModel>({
  get: () => painter.exportOptions,
  set: value => Object.assign(painter.exportOptions, value),
});
const editorHighlightModel = computed<boolean>({
  get: () => painter.editorHighlight.value,
  set: value => {
    painter.editorHighlight.value = value;
  },
});
const loading = shallowRef(false);
const errorText = shallowRef('');
const sourceBlob = shallowRef<Blob | null>(null);
const exportBusy = reactive({
  raw: false,
  rawText: false,
  doc: false,
  talkDoc: false,
  docx: false,
  forum: false,
  trg: false,
});
const docxSupported = supportsStoryPainterDocxExport();

const modeOptions: Array<{ label: string; value: StoryPainterPreviewDisplayMode }> = [
  { label: '预览', value: 'preview' },
  { label: '论坛代码', value: 'bbs' },
  { label: '论坛代码(内容多行)', value: 'bbspineapple' },
  { label: '回声工坊', value: 'trg' },
];

const title = computed(() => `${props.log.name} (${props.log.groupId})`);
const editorItems = computed(() => painter.fullItemsLoaded.value ? painter.items.value : []);

async function loadLog(): Promise<void> {
  loading.value = true;
  errorText.value = '';
  try {
    const support = getStoryPainterAdvancedModeSupport();
    if (!support.supported) {
      throw new Error(support.reason ?? '当前浏览器不支持高级日志模式');
    }
    painter.mode.value = 'editor';
    const blob = await fetchStoryLogParquet(props.log.id);
    sourceBlob.value = blob;
    const dataset = await createStoryPainterParquetDataset(blob);
    await painter.setDataset(dataset);
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : '加载日志失败';
  } finally {
    loading.value = false;
  }
}

function handleCopy(text: string): void {
  void copyText(text).then(
    () => message.success('已复制'),
    () => message.error('复制失败'),
  );
}

function exportRaw(): void {
  if (sourceBlob.value) {
    void saveStoryPainterBlob(sourceBlob.value, `${props.log.name}.parquet`);
    return;
  }
  void saveStoryPainterText(renderRawText(painter.items.value), '跑团记录(未处理).txt');
}

async function exportRawText(): Promise<void> {
  const chunks: string[] = [];
  if (painter.dataset.value) {
    for await (const items of painter.dataset.value.iterRows({ chunkSize: 2000 })) {
      chunks.push(renderRawText(items));
    }
  } else {
    chunks.push(renderRawText(painter.items.value));
  }
  await saveStoryPainterText(chunks.join('\n'), '跑团记录(未处理).txt');
}

async function buildForumText(): Promise<string> {
  return await collectStoryPainterForumText({
    chunks: painter.iterPreviewChunks(),
    chars: painter.chars.value,
    exportOptions: painter.exportOptions,
    forumOptions: painter.forumOptions,
    pineapple: painter.mode.value === 'bbspineapple',
    colorByItem: painter.colorByItem,
  });
}

async function exportForumText(): Promise<void> {
  await saveStoryPainterText(await buildForumText(), '跑团记录(论坛代码).txt');
}

async function buildTrgText(): Promise<string> {
  return await collectStoryPainterTrgText({
    chunks: painter.iterPreviewChunks(),
    chars: painter.chars.value,
    exportOptions: painter.exportOptions,
    addVoiceMark: painter.trgIsAddVoiceMark.value,
  });
}

async function exportTrgText(): Promise<void> {
  await saveStoryPainterText(await buildTrgText(), '跑团记录(回声工坊).txt');
}

async function buildCopyText(): Promise<string> {
  if (painter.mode.value === 'trg') return await buildTrgText();
  if (painter.mode.value === 'bbs' || painter.mode.value === 'bbspineapple') return await buildForumText();
  return '';
}

async function exportDoc(talk = false): Promise<void> {
  const chunks: string[] = [];
  for await (const items of painter.iterPreviewChunks()) {
    chunks.push(...items.map((item) => {
      const color = painter.colorByItem(item);
      return talk
        ? `<tr><td style="color:#999;white-space:nowrap;padding-right:8px">${item.time}</td><td style="color:${color};white-space:nowrap;padding-right:8px">${item.nickname}</td><td style="color:${color}">${renderPreviewHtml(item, painter.chars.value, painter.exportOptions, color)}</td></tr>`
        : `<p>${renderPreviewHtml(item, painter.chars.value, painter.exportOptions, color)}</p>`;
    }));
  }
  const html = talk ? `<table style="border-collapse: collapse;"><tbody>${chunks.join('\n')}</tbody></table>` : chunks.join('\n');
  await exportStoryPainterDoc(html);
}

async function exportDocx(): Promise<void> {
  const entries: StoryPainterDocxEntry[] = [];
  for await (const items of painter.iterPreviewChunks()) {
    items.forEach((item) => {
      const color = painter.colorByItem(item);
      const mount = document.createElement('div');
      mount.innerHTML = renderPreviewHtml(item, painter.chars.value, painter.exportOptions, color);
      entries.push({
        time: (mount.querySelector('._time')?.textContent ?? '').trim(),
        timeColor: readElementColor(mount.querySelector('._time')),
        nickname: (mount.querySelector('._nickname')?.textContent ?? '').trim(),
        nicknameColor: readElementColor(mount.querySelector('._nickname')),
        messageLines: extractMessageLines(mount.querySelector('._message')),
        messageColor: readElementColor(mount.querySelector('._message')),
      });
    });
  }
  await exportStoryPainterDocx(entries);
}

async function openMode(mode: StoryPainterPreviewDisplayMode): Promise<void> {
  painter.mode.value = toggleStoryPainterMode(painter.mode.value, mode);
  if (painter.mode.value !== 'editor') {
    if (mode === 'bbs' || mode === 'bbspineapple' || mode === 'trg') {
      painter.exportOptions.imageHide = true;
    }
    await painter.refreshPreview();
  }
}

async function showEditor(): Promise<void> {
  loading.value = true;
  try {
    await painter.ensureFullItems();
    painter.mode.value = 'editor';
  } finally {
    loading.value = false;
  }
}

function withBusy(key: keyof typeof exportBusy, action: () => void | Promise<void>): void {
  exportBusy[key] = true;
  Promise.resolve(action())
    .catch((error) => message.error(error instanceof Error ? error.message : '导出失败'))
    .finally(() => {
      exportBusy[key] = false;
    });
}

onMounted(() => {
  void loadLog();
});
</script>

<style scoped>
.story-painter-viewer {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.viewer-header {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  padding: 0.8rem;
}

.viewer-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.viewer-title-text {
  overflow-wrap: anywhere;
}

.viewer-alert {
  margin-bottom: 0;
}

.viewer-grid {
  display: grid;
  grid-template-columns: minmax(18rem, 24rem) minmax(0, 1fr);
  gap: 1rem;
}

.viewer-side,
.viewer-main {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-width: 0;
}

.viewer-toolbar {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  padding: 0.65rem 0.8rem;
}

@media screen and (max-width: 1100px) {
  .viewer-grid {
    grid-template-columns: 1fr;
  }
}
</style>
