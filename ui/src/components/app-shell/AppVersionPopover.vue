<template>
  <n-popover trigger="click" placement="bottom-end" :show-arrow="false">
    <template #trigger>
      <button type="button" class="version-trigger" aria-label="版本信息">
        <AppChannelTag />
      </button>
    </template>

    <div class="version-popover">
      <AppVersionDetails layout="inline" />
      <!-- 渠道说明只在来源或可信度可能被误解时出现，正式版没有这一行。 -->
      <p v-if="channelHint" class="version-popover__hint">{{ channelHint }}</p>
    </div>
  </n-popover>
</template>

<script setup lang="ts">
import { useBaseOverview } from '@/features/base/useBaseOverview';
import AppChannelTag from './AppChannelTag.vue';
import AppVersionDetails from './AppVersionDetails.vue';

const { channelHint } = useBaseOverview();
</script>

<style scoped>
/* badge 本身作为触发器，不再套一层按钮外观，避免出现两层可点击边界。 */
.version-trigger {
  display: inline-flex;
  min-height: 34px;
  align-items: center;
  border: 0;
  border-radius: var(--sd-radius-sm);
  background: transparent;
  cursor: pointer;
  font: inherit;
  padding: 0 0.15rem;
}

.version-popover {
  display: grid;
  max-width: 15rem;
  gap: 0.35rem;
}

.version-popover__hint {
  margin: 0;
  color: var(--sd-text-secondary);
  font-size: 0.75rem;
  line-height: 1.5;
}
</style>
