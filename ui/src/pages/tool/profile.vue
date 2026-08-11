<template>
  <main class="pprof-page">
    <PageHeader
      title="性能分析"
      description="通过 V2 pprof 接口导出采样数据，下载后可用 `go tool pprof` / `go tool trace` 离线分析。"
    />

    <n-collapse arrow-placement="right" class="pprof-page__help">
      <n-collapse-item title="查看帮助" name="help">
        <n-space vertical size="small">
          <n-text>本页调用 Go 标准库 `net/http/pprof` 相关端点。</n-text>
          <n-text>CPU profile 和 trace 会持续采样，排障完成后应尽快停止，不建议长时间运行。</n-text>
          <n-text>文本视图适合快速浏览，二进制文件更适合交给本地工具深入分析。</n-text>
        </n-space>
      </n-collapse-item>
    </n-collapse>

    <n-grid cols="1 s:2 m:3" responsive="screen" :x-gap="16" :y-gap="16">
      <n-grid-item v-for="entry in entries" :key="entry.key">
        <n-card :title="entry.title" size="small" class="pprof-page__card">
          <template #header-extra>
            <n-tag v-if="entry.secondsModel" type="warning" size="small" :bordered="false">
              需时长
            </n-tag>
          </template>

          <n-space vertical size="small">
            <n-text depth="3">{{ entry.desc }}</n-text>

            <n-flex v-if="entry.secondsModel === 'profile'" align="center" wrap>
              <n-text depth="3">采样时长（秒）</n-text>
              <n-input-number v-model:value="profileSeconds" :min="1" :max="600" :step="5" />
            </n-flex>

            <n-flex v-if="entry.secondsModel === 'trace'" align="center" wrap>
              <n-text depth="3">采样时长（秒）</n-text>
              <n-input-number v-model:value="traceSeconds" :min="1" :max="60" :step="1" />
            </n-flex>

            <n-flex wrap>
              <n-button
                type="primary"
                size="small"
                :loading="isPending(entry.key)"
                :disabled="isPending(entry.key)"
                @click="handleDownload(entry)"
              >
                下载
              </n-button>
              <n-button
                v-if="buildPprofTextPath(entry)"
                size="small"
                :loading="isPending(entry.key)"
                :disabled="isPending(entry.key)"
                @click="handleView(entry)"
              >
                查看文本
              </n-button>
            </n-flex>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-drawer v-model:show="previewVisible" placement="right" :width="drawerWidth">
      <n-drawer-content :title="previewTitle" closable>
        <n-spin :show="previewLoading">
          <pre class="pprof-page__preview">{{ previewContent }}</pre>
        </n-spin>
      </n-drawer-content>
    </n-drawer>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { getApiBaseUrl } from '@/api';
import PageHeader from '@/components/shared/PageHeader.vue';
import { currentAccessToken } from '@/features/auth/state';
import { useResponsiveOverlayWidth } from '@/features/responsive/useResponsiveOverlayWidth';
import {
  buildPprofBinaryPath,
  buildPprofTextPath,
  createPprofEntries,
  type PprofEntry,
} from '@/features/pprof/model';

const message = useMessage();
const { width: drawerWidth } = useResponsiveOverlayWidth({ maxWidth: 880, gutter: 16 });

const entries = createPprofEntries();
const profileSeconds = ref(30);
const traceSeconds = ref(5);
const previewVisible = ref(false);
const previewTitle = ref('');
const previewContent = ref('');
const previewLoading = ref(false);
const pendingExpiry = ref<Record<string, number>>({});
const pendingStorageKey = 'sd-v2-pprof-pending';
const pprofBase = computed(() => `${getApiBaseUrl()}/sd-api/v2/pprof`);

function persistPending() {
  try {
    window.sessionStorage.setItem(pendingStorageKey, JSON.stringify(pendingExpiry.value));
  } catch {
    // ignore
  }
}

function clearPending(key: string) {
  if (!(key in pendingExpiry.value)) return;
  const next = { ...pendingExpiry.value };
  delete next[key];
  pendingExpiry.value = next;
  persistPending();
}

function markPending(key: string, durationMs: number) {
  pendingExpiry.value = { ...pendingExpiry.value, [key]: Date.now() + durationMs };
  persistPending();
  window.setTimeout(() => clearPending(key), durationMs);
}

function isPending(key: string) {
  const expiry = pendingExpiry.value[key];
  if (!expiry) return false;
  if (expiry <= Date.now()) {
    clearPending(key);
    return false;
  }
  return true;
}

async function fetchPprof(path: string) {
  const token = currentAccessToken();
  const headers = new Headers();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  const response = await fetch(`${pprofBase.value}${path}`, {
    headers,
    credentials: 'same-origin',
  });
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  return response;
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  window.URL.revokeObjectURL(url);
}

async function handleDownload(entry: PprofEntry) {
  if (isPending(entry.key)) return;

  const durationMs =
    entry.key === 'profile'
      ? profileSeconds.value * 1000
      : entry.key === 'trace'
        ? traceSeconds.value * 1000
        : 1500;
  markPending(entry.key, durationMs);

  try {
    const response = await fetchPprof(
      buildPprofBinaryPath(entry, {
        profileSeconds: profileSeconds.value,
        traceSeconds: traceSeconds.value,
      })
    );
    const blob = await response.blob();
    triggerBlobDownload(blob, entry.filename);
  } catch (error) {
    clearPending(entry.key);
    message.error(error instanceof Error ? error.message : String(error));
  }
}

async function handleView(entry: PprofEntry) {
  const textPath = buildPprofTextPath(entry);
  if (!textPath || isPending(entry.key)) return;

  markPending(entry.key, 5000);
  previewTitle.value = entry.title;
  previewContent.value = '';
  previewLoading.value = true;
  previewVisible.value = true;

  try {
    const response = await fetchPprof(textPath);
    previewContent.value = await response.text();
  } catch (error) {
    previewContent.value = `加载失败：${String(error)}`;
  } finally {
    previewLoading.value = false;
    clearPending(entry.key);
  }
}

onMounted(() => {
  try {
    const raw = window.sessionStorage.getItem(pendingStorageKey);
    if (!raw) return;
    const parsed = JSON.parse(raw) as Record<string, number>;
    const now = Date.now();
    const valid: Record<string, number> = {};
    for (const [key, expiry] of Object.entries(parsed)) {
      if (expiry > now) {
        valid[key] = expiry;
        window.setTimeout(() => clearPending(key), expiry - now);
      }
    }
    pendingExpiry.value = valid;
    persistPending();
  } catch {
    pendingExpiry.value = {};
  }
});
</script>

<style scoped>
.pprof-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.pprof-page__card {
  height: 100%;
}

.pprof-page__preview {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  line-height: 1.5;
}

.pprof-page__help {
  border: 1px solid var(--sd-border-soft);
  border-radius: 6px;
  background: var(--sd-bg-elevated-soft);
}
</style>
