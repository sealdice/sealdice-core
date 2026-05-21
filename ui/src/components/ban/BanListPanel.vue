<script setup lang="ts">
import { computed } from 'vue';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import type { UploadCustomRequestOptions } from 'naive-ui';
import type { BanListInfoItem } from '@/api';
import {
  getBanRankMeta,
  type BanListQueryModel,
} from '@/features/ban/viewModel';

dayjs.extend(relativeTime);

const props = defineProps<{
  items: BanListInfoItem[];
  total: number;
  loading: boolean;
  query: BanListQueryModel;
  addPending: boolean;
  importPending: boolean;
}>();

const emit = defineEmits<{
  updateQuery: [patch: Partial<BanListQueryModel>];
  openAdd: [];
  delete: [item: BanListInfoItem];
  import: [file: File];
  export: [];
}>();

const rankValues = computed({
  get: () => props.query.ranks,
  set: value => emit('updateQuery', { ranks: [...value], page: 1 }),
});

const keyword = computed({
  get: () => props.query.keyword,
  set: value => emit('updateQuery', { keyword: value, page: 1 }),
});

const sortBy = computed({
  get: () => props.query.sortBy,
  set: value => emit('updateQuery', { sortBy: value, page: 1 }),
});

function updatePage(page: number) {
  emit('updateQuery', { page });
}

function updatePageSize(pageSize: number) {
  emit('updateQuery', { pageSize, page: 1 });
}

async function uploadBanFile(options: UploadCustomRequestOptions) {
  const file = options.file.file;
  if (!(file instanceof File)) {
    options.onError?.();
    return;
  }
  try {
    emit('import', file);
    options.onFinish?.();
  } catch {
    options.onError?.();
  }
}
</script>

<template>
  <section class="ban-list-panel">
    <header class="ban-list-panel__toolbar">
      <n-flex size="small" align="center" wrap>
        <n-text>搜索：</n-text>
        <n-input v-model:value="keyword" class="ban-list-panel__search" placeholder="按 ID 或名字筛选" clearable />
      </n-flex>

      <n-flex align="center" wrap>
        <n-button type="success" secondary :loading="addPending" @click="emit('openAdd')">
          <template #icon>
            <n-icon><i-carbon-add-large /></n-icon>
          </template>
          添加
        </n-button>
        <n-upload
          action=""
          accept=".json,application/json"
          :show-file-list="false"
          :custom-request="uploadBanFile"
        >
          <n-button type="info" secondary :loading="importPending">
            <template #icon>
              <n-icon><i-carbon-upload /></n-icon>
            </template>
            导入
          </n-button>
        </n-upload>
        <n-button type="info" secondary @click="emit('export')">
          <template #icon>
            <n-icon><i-carbon-download /></n-icon>
          </template>
          导出
        </n-button>
      </n-flex>
    </header>

    <n-flex align="center" wrap>
      <n-text>级别：</n-text>
      <n-checkbox-group v-model:value="rankValues">
        <n-space item-style="display:flex;">
          <n-checkbox :value="-30">拉黑</n-checkbox>
          <n-checkbox :value="-10">警告</n-checkbox>
          <n-checkbox :value="30">信任</n-checkbox>
          <n-checkbox :value="0">其它</n-checkbox>
        </n-space>
      </n-checkbox-group>
      <n-text>排序：</n-text>
      <n-radio-group v-model:value="sortBy" size="small">
        <n-radio-button value="time">按封禁时间</n-radio-button>
        <n-radio-button value="score">按怒气值</n-radio-button>
      </n-radio-group>
    </n-flex>

    <n-spin :show="loading">
      <n-list hoverable clickable class="ban-list-panel__list">
        <n-list-item v-for="item in items" :key="item.ID">
          <n-thing>
            <template #header>
              <n-flex size="small" align="center">
                <n-tag :type="getBanRankMeta(item.rank).tagType" :bordered="false">
                  {{ getBanRankMeta(item.rank).label }}
                </n-tag>
                <n-text tag="strong">{{ item.ID }}</n-text>
              </n-flex>
            </template>
            <template #header-extra>
              <n-button type="error" size="small" secondary @click="emit('delete', item)">
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                删除
              </n-button>
            </template>
            <template #description>
              <n-flex size="small" align="center" wrap>
                <n-text>「{{ item.name || '未命名' }}」</n-text>
                <n-text depth="3">怒气值：{{ item.score }}</n-text>
              </n-flex>
            </template>

            <n-flex vertical size="small" class="ban-list-panel__reasons">
              <div
                v-for="(reason, index) in item.reasons ?? []"
                :key="`${item.ID}-${index}`"
                class="ban-list-panel__reason-item"
              >
                <n-tooltip>
                  <template #trigger>
                    <n-tag size="small" type="info" :bordered="false">
                      {{ dayjs.unix(item.times?.[index] ?? item.banTime).fromNow() }}
                    </n-tag>
                  </template>
                  {{ dayjs.unix(item.times?.[index] ?? item.banTime).format('YYYY-MM-DD HH:mm:ss') }}
                </n-tooltip>
                <n-text>
                  在 &lt;{{ item.places?.[index] || '未知地点' }}&gt;，原因：「{{ reason }}」
                </n-text>
              </div>
            </n-flex>
          </n-thing>
        </n-list-item>
      </n-list>

      <n-empty v-if="!items.length" description="暂无黑白名单条目" class="ban-list-panel__empty" />
    </n-spin>

    <footer class="ban-list-panel__footer">
      <n-pagination
        :page="query.page"
        :page-size="query.pageSize"
        :item-count="total"
        show-size-picker
        :page-sizes="[10, 20, 30, 50]"
        @update:page="updatePage"
        @update:page-size="updatePageSize"
      />
    </footer>
  </section>
</template>

<style scoped>
.ban-list-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.ban-list-panel__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.ban-list-panel__search {
  width: min(20rem, 80vw);
}

.ban-list-panel__list {
  border-radius: 14px;
  background: var(--sd-bg-elevated);
}

.ban-list-panel__reasons {
  margin-top: 0.5rem;
}

.ban-list-panel__reason-item {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.ban-list-panel__empty {
  padding: 1.5rem 0;
}

.ban-list-panel__footer {
  display: flex;
  justify-content: center;
}
</style>
