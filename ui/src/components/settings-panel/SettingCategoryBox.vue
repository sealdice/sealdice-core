<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
  collapsible?: boolean;
  expanded?: boolean;
  wide?: boolean;
}>();

const emit = defineEmits<{
  toggle: [];
}>();
</script>

<template>
  <section :class="['setting-category-box', { 'setting-category-box-wide': wide }]">
    <n-thing class="setting-category-thing">
      <template #header>
        <span class="setting-category-title">{{ title }}</span>
      </template>
      <template v-if="description" #description>
        <span class="setting-category-description">{{ description }}</span>
      </template>
      <template v-if="collapsible" #header-extra>
        <n-button
          text
          size="small"
          class="setting-category-toggle"
          @click="emit('toggle')"
        >
          {{ expanded ? '收起' : '展开' }}
        </n-button>
      </template>
    </n-thing>

    <div v-if="!collapsible || expanded" class="setting-category-panel">
      <slot name="notes" />
      <slot />
    </div>
  </section>
</template>

<style scoped>
.setting-category-box {
  padding: 0.45rem 0 0.8rem;
}

.setting-category-box-wide {
  grid-column: 1 / -1;
}

.setting-category-thing {
  margin: 0 1rem 0.45rem;
}

.setting-category-title {
  color: var(--sd-text-primary);
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.35;
}

.setting-category-description {
  color: var(--sd-text-muted);
  font-size: 0.82rem;
  line-height: 1.45;
}

.setting-category-toggle {
  flex: 0 0 auto;
}

.setting-category-panel {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  overflow: hidden;
  border-radius: 8px;
  background: var(--sd-bg-elevated);
}

@media (max-width: 639.9px) {
  .setting-category-box {
    padding-bottom: 0.6rem;
  }

  .setting-category-thing {
    margin: 0 0.45rem 0.35rem;
  }
}
</style>
