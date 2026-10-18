<template>
  <ListActions class="helpdoc-action-block">
    <n-button type="primary" secondary @click="emit('openUpload')">
      <template #icon>
        <n-icon><i-tabler-upload /></n-icon>
      </template>
      上传
    </n-button>
    <template #end>
      <n-button secondary @click="emit('openConfig')">
        <template #icon>
          <n-icon><i-tabler-settings /></n-icon>
        </template>
        设置
      </n-button>
      <n-button
        v-show="checkedKeys.length > 0"
        type="error"
        secondary
        :loading="deleting"
        @click="emit('deleteFiles')"
      >
        <template #icon>
          <n-icon><i-tabler-trash /></n-icon>
        </template>
        删除所选
      </n-button>
    </template>
  </ListActions>

  <n-spin :show="loading">
    <main class="helpdoc-file-block">
      <header class="file-tree-title">
        <n-text strong>文件名</n-text>
        <n-text strong>分组</n-text>
      </header>
      <ListEmptyState v-if="!loading && docTree.length === 0" description="暂无帮助文档文件" />
      <n-tree
        v-else
        v-model:checked-keys="checkedKeys"
        :data="docTree"
        label-field="label"
        block-line
        default-expand-all
        show-line
        cascade
        checkable
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
      />
    </main>
  </n-spin>
</template>

<script setup lang="tsx">
import { h } from 'vue';
import { NTag, NText } from 'naive-ui';
import ListActions from '@/components/shared/ListActions.vue';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
import type { HelpDocTreeOption } from '@/features/helpdoc/viewModel';

type TreeRenderContext = {
  option: Record<string, unknown>;
  checked: boolean;
  selected: boolean;
};

const checkedKeys = defineModel<Array<string | number>>('checkedKeys', { required: true });

defineProps<{
  docTree: HelpDocTreeOption[];
  loading: boolean;
  deleting: boolean;
}>();

const emit = defineEmits<{
  openUpload: [];
  openConfig: [];
  deleteFiles: [];
}>();

function renderPrefix({ option }: TreeRenderContext) {
  const raw = option as HelpDocTreeOption | undefined;
  const icon = raw?.icon;
  if (icon === 'folder') return <i-tabler-folder color="var(--sd-text-muted)" />;
  if (icon === 'json' || icon === 'xlsx') {
    return <i-tabler-file-text color="var(--sd-text-secondary)" />;
  }
  return <i-tabler-file-text />;
}

function renderLabel({ option }: TreeRenderContext) {
  const raw = option as HelpDocTreeOption | undefined;
  return h(
    NText,
    { class: raw?.raw.deleted ? 'del-line file-info' : 'file-info' },
    { default: () => option.label as string }
  );
}

function renderSuffix({ option }: TreeRenderContext) {
  const raw = option as HelpDocTreeOption | undefined;
  if (!raw?.tag) return null;
  return h(
    NTag,
    { size: 'small', type: raw.tag.type, bordered: false },
    { default: () => raw.tag?.label ?? '' }
  );
}
</script>

<style scoped>
.helpdoc-action-block {
  margin-bottom: 0.75rem;
}

.helpdoc-file-block {
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  color: var(--sd-text-primary);
  padding: 0.75rem;
}

.file-tree-title {
  display: flex;
  justify-content: space-between;
  padding: 0 23px 6px 50px;
  border-bottom: 1px solid var(--sd-border);
  margin-bottom: 0.5rem;
}

.file-info {
  flex: auto;
  width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.del-line {
  text-decoration: line-through;
}

@media screen and (max-width: 639.9px) {
  .file-tree-title {
    gap: 0.75rem;
    padding-left: 0;
  }
}
</style>
