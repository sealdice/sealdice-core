<script setup lang="ts">
import { type ResourceItem } from '@/api';
import { useResourcePreview } from '@/features/resource/useResourcePreview';

const props = withDefaults(
  defineProps<{
    item: ResourceItem;
    thumbnail?: boolean;
  }>(),
  {
    thumbnail: true,
  },
);

const { objectUrl, loading, failed } = useResourcePreview(
  () => props.item,
  () => props.thumbnail,
);
</script>

<template>
  <div class="resource-preview">
    <n-skeleton v-if="loading && !objectUrl" class="resource-preview__skeleton" />
    <n-image
      v-else-if="objectUrl"
      class="resource-preview__image"
      :src="objectUrl"
      :preview-src="objectUrl"
      :alt="item.name"
      object-fit="cover"
      lazy
    />
    <div v-else class="resource-preview__fallback" :class="{ 'resource-preview__fallback--failed': failed }">
      <n-icon size="22">
        <i-carbon-image />
      </n-icon>
    </div>
  </div>
</template>

<style scoped>
.resource-preview {
  display: grid;
  width: 72px;
  height: 72px;
  overflow: hidden;
  place-items: center;
  border: 1px solid var(--sd-border-soft);
  border-radius: 14px;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--sd-bg-control), transparent 10%), transparent),
    var(--sd-bg-elevated-soft);
}

.resource-preview__image {
  width: 72px;
  height: 72px;
}

.resource-preview__image :deep(img) {
  width: 72px;
  height: 72px;
  object-fit: cover;
}

.resource-preview__skeleton {
  width: 72px;
  height: 72px;
}

.resource-preview__fallback {
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  color: var(--sd-text-muted);
}

.resource-preview__fallback--failed {
  color: var(--sd-warning);
}
</style>
