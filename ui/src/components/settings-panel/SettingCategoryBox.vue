<template>
  <section :class="['setting-category-box', { 'setting-category-box-wide': wide }]">
    <n-thing class="setting-category-thing">
      <template #header>
        <span class="setting-category-heading">
          <span :id="headingId" class="setting-category-title">{{ title }}</span>
          <slot name="title-extra" />
        </span>
      </template>
      <template v-if="collapsible" #header-extra>
        <n-button
          text
          size="small"
          class="setting-category-toggle"
          :aria-label="expanded ? `收起${title}` : `展开${title}`"
          :aria-expanded="expanded"
          :aria-controls="panelId"
          @click="emit('toggle')"
        >
          {{ expanded ? '收起' : '展开' }}
        </n-button>
      </template>
    </n-thing>

    <div
      v-if="showPanel && (!collapsible || expanded)"
      :id="panelId"
      :class="['setting-category-panel', { 'setting-category-panel-padded': padded }]"
      role="region"
      :aria-labelledby="headingId"
    >
      <div v-if="$slots.notes" class="setting-category-notes">
        <slot name="notes" />
      </div>
      <SettingFieldLayout :columns="columns" :padded="padded">
        <slot />
      </SettingFieldLayout>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useId } from 'vue';
import SettingFieldLayout from './SettingFieldLayout.vue';

const panelId = `setting-category-panel-${useId()}`;
const headingId = `setting-category-heading-${useId()}`;

withDefaults(
  defineProps<{
    title: string;
    collapsible?: boolean;
    expanded?: boolean;
    wide?: boolean;
    padded?: boolean;
    columns?: 1 | 2;
    showPanel?: boolean;
  }>(),
  {
    showPanel: true,
  }
);

const emit = defineEmits<{
  toggle: [];
}>();
</script>

<style scoped>
.setting-category-box {
  padding: var(--sd-space-xs) 0 var(--sd-space-sm);
}

.setting-category-box-wide {
  grid-column: 1 / -1;
}

.setting-category-thing {
  margin: 0 var(--sd-space-md) var(--sd-space-xs);
}

.setting-category-title {
  color: var(--sd-text-primary);
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.35;
}

.setting-category-heading {
  display: inline-flex;
  align-items: center;
  gap: var(--sd-space-xs);
  min-width: 0;
}

.setting-category-toggle {
  flex: 0 0 auto;
}

.setting-category-panel {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  overflow: hidden;
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}

/*
 * 说明块与子面板内的字段左右对齐，且上下内边距必须与面板底部一致。
 * 底部内边距的来源随 padded 变化，因此这里的顶部内边距也要跟着变：
 * - 未 padded：字段由 SettingRow 自带 --sd-space-xs 纵向内边距，底部即 xs；
 * - 已 padded：由 SettingFieldLayout 提供 --sd-space-md 四边内边距，底部即 md。
 * 与字段之间的间隔由后续容器自身的顶部内边距提供，故这里 bottom 为 0。
 */
.setting-category-notes {
  display: flex;
  flex-direction: column;
  gap: var(--sd-space-xs);
  padding: var(--sd-space-xs) var(--sd-space-md) 0;
}

.setting-category-panel-padded .setting-category-notes {
  padding: var(--sd-space-md) var(--sd-space-md) 0;
}

@media (max-width: 639.9px) {
  .setting-category-box {
    padding-bottom: var(--sd-space-xs);
  }

  .setting-category-thing {
    margin: 0 var(--sd-space-xs) var(--sd-space-2xs);
  }

  /* 窄屏下字段内边距收窄为 sm，说明块跟随。 */
  .setting-category-notes {
    padding: var(--sd-space-xs) var(--sd-space-sm) 0;
  }

  .setting-category-panel-padded .setting-category-notes {
    padding: var(--sd-space-sm) var(--sd-space-sm) 0;
  }
}
</style>
