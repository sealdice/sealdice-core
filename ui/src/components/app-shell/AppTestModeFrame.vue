<template>
  <div class="test-mode-frame" :class="{ 'test-mode-frame--active': active }">
    <div v-if="active" class="test-mode-frame__banner">
      <n-alert type="warning" :bordered="false" :show-icon="false">
        {{ bannerText }}
      </n-alert>
    </div>
    <div v-if="active" class="test-mode-frame__watermark" aria-hidden="true">
      <span
        v-for="row in watermarkRows"
        :key="row"
        class="test-mode-frame__watermark-row"
      >
        {{ row }}
      </span>
    </div>
    <div class="test-mode-frame__content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { getTestModeWatermarkRows } from './appTestModeFrame';

const props = defineProps<{
  active: boolean;
  bannerText: string;
  watermarkText: string;
}>();

const watermarkRows = computed(() => getTestModeWatermarkRows(props.watermarkText));
</script>

<style scoped>
.test-mode-frame {
  position: relative;
  min-height: 100%;
}

.test-mode-frame__content {
  position: relative;
  z-index: 1;
  min-height: 100%;
}

.test-mode-frame__banner {
  position: sticky;
  top: 0;
  z-index: 30;
  padding: 0.75rem 0.75rem 0;
}

.test-mode-frame__watermark {
  position: fixed;
  inset: 0;
  z-index: 0;
  display: grid;
  place-content: center;
  gap: 4rem;
  overflow: hidden;
  pointer-events: none;
  opacity: 0.08;
  transform: rotate(-24deg) scale(1.15);
}

.test-mode-frame__watermark-row {
  white-space: nowrap;
  color: var(--sd-text-primary);
  font-size: clamp(1.15rem, 1.8vw, 1.8rem);
  font-weight: 800;
  letter-spacing: 0.22em;
  text-transform: uppercase;
}

@media (max-width: 767.9px) {
  .test-mode-frame__banner {
    padding: 0.5rem 0.5rem 0;
  }

  .test-mode-frame__watermark {
    gap: 3rem;
    transform: rotate(-24deg) scale(1.3);
  }
}
</style>
