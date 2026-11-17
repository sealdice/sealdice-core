<template>
  <span
    class="store-package-icon"
    :style="{ '--store-package-icon-size': `${size}px` }"
    aria-hidden="true"
  >
    <span class="store-package-icon__fallback">{{ fallback }}</span>
    <img
      v-if="iconUrl && !imageFailed"
      :src="iconUrl"
      alt=""
      class="store-package-icon__image"
      loading="lazy"
      @error="imageFailed = true"
    />
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    name: string;
    iconUrl: string;
    content?: string;
    size?: number;
  }>(),
  {
    content: '',
    size: 34,
  }
);

const contentFallback: Record<string, string> = {
  all: 'EX',
  scripts: 'JS',
  decks: '牌',
  reply: '回',
  helpdoc: '文',
  templates: '模',
};

const imageFailed = ref(false);

watch(
  () => props.iconUrl,
  () => {
    imageFailed.value = false;
  }
);

const fallback = computed(
  () =>
    contentFallback[props.content || 'all'] ?? (props.name.trim().slice(0, 2).toUpperCase() || '?')
);
</script>

<style scoped>
.store-package-icon {
  display: inline-flex;
  width: var(--store-package-icon-size, 34px);
  height: var(--store-package-icon-size, 34px);
  flex: none;
  position: relative;
  overflow: hidden;
  align-items: center;
  justify-content: center;
  border-radius: var(--sd-radius-sm);
  background: var(--sd-primary-soft);
  color: var(--sd-primary);
  font-size: 0.75rem;
  font-weight: 700;
  line-height: 1;
}

.store-package-icon__fallback {
  user-select: none;
}

.store-package-icon__image {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
