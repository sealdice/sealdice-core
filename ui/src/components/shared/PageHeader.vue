<template>
  <header class="sd-page-header">
    <div class="sd-page-header__copy">
      <h1>{{ title }}</h1>
    </div>
    <div v-if="$slots.default || pendingActions.length" class="sd-page-header__actions">
      <div v-if="pendingActions.length" ref="anchorRef" class="sd-page-header__pending">
        <PendingActionRow
          v-for="action in pendingActions"
          :key="action.scope"
          :action="action"
          @run="handleRun(action.scope)"
        />
      </div>
      <slot />
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, useTemplateRef, watch } from 'vue';
import PendingActionRow from '@/components/shared/PendingActionRow.vue';
import {
  activePendingActions,
  runPendingAction,
  setPendingActionAnchor,
} from '@/features/unsavedChanges';

const props = defineProps<{
  title: string;
  /**
   * 声明本页承载待处理状态。传入后，标题右侧显示当前页的未保存/待重载状态与操作，
   * 并作为悬浮面板的可见性锚点 —— 此处可见时悬浮面板不出现。
   */
  unsavedScope?: string | string[];
}>();

const anchorRef = useTemplateRef<HTMLElement>('anchorRef');

const scopes = computed(() => {
  if (!props.unsavedScope) return null;
  return Array.isArray(props.unsavedScope) ? props.unsavedScope : [props.unsavedScope];
});

const pendingActions = computed(() => {
  const allowed = scopes.value;
  if (!allowed) return [];
  return activePendingActions.value.filter(action => allowed.includes(action.scope));
});

// 状态区随脏态出现或消失，锚点要跟着登记与注销。
watch(anchorRef, el => setPendingActionAnchor(el ?? null), { immediate: true });
onBeforeUnmount(() => setPendingActionAnchor(null));

async function handleRun(scope: string) {
  await runPendingAction(scope);
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
  font-size: var(--sd-page-title-size);
  font-weight: var(--sd-page-title-weight);
  line-height: var(--sd-page-title-line-height);
}

.sd-page-header__actions {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: var(--sd-space-sm);
  flex-wrap: wrap;
}

.sd-page-header__pending {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--sd-space-md);
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

  .sd-page-header__pending {
    width: 100%;
  }
}
</style>
