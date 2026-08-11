<template>
  <header class="sd-page-header">
    <div class="sd-page-header__copy">
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div v-if="$slots.default || unsavedScope" class="sd-page-header__actions">
      <template v-if="unsavedScope">
        <n-button v-if="dirty" secondary :disabled="!dirty || saving" @click="discard">
          <template #icon>
            <n-icon><i-ep-refresh-left /></n-icon>
          </template>
          放弃改动
        </n-button>
        <n-button
          v-if="dirty"
          type="primary"
          :loading="saving"
          :disabled="!dirty || saving"
          @click="save"
        >
          <template #icon>
            <n-icon><i-ep-document-checked /></n-icon>
          </template>
          保存设置
        </n-button>
      </template>
      <slot />
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useMessage } from 'naive-ui';
import { activeUnsavedChangesSource, saveActiveUnsavedChanges } from '@/features/unsavedChanges';

const props = defineProps<{
  title: string;
  description?: string;
  unsavedScope?: string;
}>();

const message = useMessage();
const activeSource = computed(() => (props.unsavedScope ? activeUnsavedChangesSource.value : null));
const dirty = computed(() =>
  Boolean(activeSource.value && activeSource.value.scope === props.unsavedScope)
);
const saving = computed(() =>
  Boolean(
    activeSource.value &&
      activeSource.value.scope === props.unsavedScope &&
      activeSource.value.saving
  )
);

async function save() {
  if (!props.unsavedScope) return;
  const saved = await saveActiveUnsavedChanges();
  if (!saved) {
    message.error('保存失败');
  }
}

async function discard() {
  message.info('请使用页面内的“放弃改动”按钮');
}
</script>

<style scoped>
.sd-page-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sd-space-md);
  margin-bottom: var(--sd-space-lg);
  padding-bottom: var(--sd-space-md);
  border-bottom: 1px solid var(--sd-border-soft);
}

.sd-page-header__copy {
  min-width: 0;
}

.sd-page-header h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.25;
}

.sd-page-header p {
  max-width: 60rem;
  margin: 0.4rem 0 0;
  color: var(--sd-text-secondary);
  line-height: 1.5;
}

.sd-page-header__actions {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: var(--sd-space-sm);
  flex-wrap: wrap;
}

@media (max-width: 640px) {
  .sd-page-header {
    grid-template-columns: 1fr;
    align-items: start;
  }

  .sd-page-header__actions {
    justify-content: flex-start;
  }
}
</style>
