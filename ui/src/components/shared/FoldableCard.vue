<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';

const props = withDefaults(
  defineProps<{
    shadow?: 'always' | 'never' | 'hover';
    type?: 'card' | 'div' | string;
    errTitle?: string;
    errText?: string;
    defaultFold?: 'auto' | boolean;
    compact?: boolean;
  }>(),
  {
    shadow: 'hover',
    type: 'card',
    defaultFold: 'auto',
    compact: false,
  },
);

const folded = ref<boolean | undefined>(undefined);

const open = () => {
  folded.value = false;
};

const close = () => {
  folded.value = true;
};

const updateFolded = () => {
  if (props.defaultFold === 'auto') {
    folded.value = folded.value ?? !window.matchMedia('(min-width: 768px)').matches;
  } else {
    folded.value = folded.value ?? props.defaultFold;
  }
};

watch(
  () => props.defaultFold,
  () => updateFolded(),
);

window.addEventListener('resize', updateFolded);
onMounted(() => {
  updateFolded();
});

defineExpose({ open, close });
</script>

<template>
  <div
    v-if="type === 'card'"
    class="rounded-lg border border-gray-200 bg-white p-4"
    :class="{
      'shadow-sm': shadow === 'always',
      'shadow-sm hover:shadow-md transition-shadow': shadow === 'hover',
    }">
    <template v-if="!errText">
      <header :class="compact ? 'foldable-card-header foldable-card-header--compact' : 'foldable-card-header'">
        <div class="foldable-card-title-row">
          <div class="foldable-card-title">
            <slot name="title" />
          </div>
          <div class="foldable-card-title-actions">
            <div class="foldable-card-title-extra">
              <slot name="title-extra" />
            </div>
            <n-button text size="small" @click="folded = !folded">
              <template #icon>
                <span class="text-gray-400 text-xs">{{ folded ? '\u25B6' : '\u25BC' }}</span>
              </template>
            </n-button>
          </div>
        </div>

        <div v-if="!compact" class="foldable-card-secondary-row">
          <div class="foldable-card-description">
            <slot name="description" />
          </div>
          <div class="foldable-card-actions">
            <slot name="action" />
          </div>
        </div>
      </header>

      <template v-if="!folded">
        <div class="w-full">
          <slot name="default" />
        </div>
        <div class="w-full">
          <slot name="extra" />
        </div>
      </template>
      <div v-else class="w-full">
        <slot name="unfolded-extra" />
      </div>
    </template>

    <template v-else>
      <header class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <n-icon size="20" color="var(--n-error-color)">
            <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/><line x1="15" y1="9" x2="9" y2="15" stroke="currentColor" stroke-width="2"/><line x1="9" y1="9" x2="15" y2="15" stroke="currentColor" stroke-width="2"/></svg>
          </n-icon>
          <n-text type="error" :depth="1">
            <del>{{ errTitle }}</del>
          </n-text>
        </div>
        <div class="flex flex-wrap justify-end gap-2">
          <slot name="title-extra-error" />
        </div>
      </header>
      <div class="mt-2 flex items-start justify-between gap-2">
        <div class="whitespace-pre-line">
          <span class="text-sm text-gray-500">错误信息：</span>
          <n-text type="error">{{ errText }}</n-text>
        </div>
        <div class="flex items-center justify-end">
          <slot name="action-error" />
        </div>
      </div>
    </template>
  </div>

  <div v-else>
    <template v-if="!errText">
      <header :class="compact ? 'foldable-card-header foldable-card-header--compact' : 'foldable-card-header foldable-card-header--tight'">
        <div class="foldable-card-title-row">
          <div class="foldable-card-title">
            <slot name="title" />
          </div>
          <div class="foldable-card-title-actions">
            <div class="foldable-card-title-extra">
              <slot name="title-extra" />
            </div>
            <n-button text size="small" @click="folded = !folded">
              <template #icon>
                <span class="text-gray-400 text-xs">{{ folded ? '\u25B6' : '\u25BC' }}</span>
              </template>
            </n-button>
          </div>
        </div>

        <div v-if="!compact" class="foldable-card-secondary-row">
          <div class="foldable-card-description">
            <slot name="description" />
          </div>
          <div class="foldable-card-actions">
            <slot name="action" />
          </div>
        </div>
      </header>

      <template v-if="!folded">
        <div class="w-full">
          <slot name="default" />
        </div>
        <div class="w-full">
          <slot name="extra" />
        </div>
      </template>
      <div v-else class="w-full">
        <slot name="unfolded-extra" />
      </div>
    </template>

    <template v-else>
      <header class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <n-icon size="20" color="var(--n-error-color)">
            <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/><line x1="15" y1="9" x2="9" y2="15" stroke="currentColor" stroke-width="2"/><line x1="9" y1="9" x2="15" y2="15" stroke="currentColor" stroke-width="2"/></svg>
          </n-icon>
          <n-text type="error" :depth="1">
            <del>{{ errTitle }}</del>
          </n-text>
        </div>
        <div class="flex flex-wrap justify-end gap-2">
          <slot name="title-extra-error" />
        </div>
      </header>
      <div class="mt-2 flex items-start justify-between gap-2">
        <div class="whitespace-pre-line">
          <span class="text-sm text-gray-500">错误信息：</span>
          <n-text type="error">{{ errText }}</n-text>
        </div>
        <div class="flex items-center justify-end">
          <slot name="action-error" />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.foldable-card-header {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-width: 0;
}

.foldable-card-header--compact {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}

.foldable-card-header--tight {
  gap: 0.25rem;
}

.foldable-card-title-row,
.foldable-card-secondary-row,
.foldable-card-title-actions,
.foldable-card-title-extra,
.foldable-card-description,
.foldable-card-actions {
  display: flex;
  min-width: 0;
}

.foldable-card-title-row,
.foldable-card-secondary-row {
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.foldable-card-title {
  min-width: 0;
  margin-right: 0.5rem;
  overflow-wrap: anywhere;
}

.foldable-card-title-actions {
  flex: 0 0 auto;
  align-items: center;
  gap: 0.5rem;
}

.foldable-card-title-extra {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
}

.foldable-card-description {
  flex: 1 1 auto;
  overflow-wrap: anywhere;
}

.foldable-card-actions {
  flex: 0 1 auto;
  justify-content: flex-end;
  margin-left: auto;
  padding-right: 2.5rem;
}

@media screen and (max-width: 639.9px) {
  .foldable-card-header--compact,
  .foldable-card-title-row,
  .foldable-card-secondary-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .foldable-card-title-actions,
  .foldable-card-title-extra,
  .foldable-card-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .foldable-card-actions {
    margin-left: 0;
    padding-right: 0;
  }
}
</style>
