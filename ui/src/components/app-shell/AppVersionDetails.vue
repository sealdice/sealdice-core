<template>
  <div class="version-details" :class="`version-details--${props.layout}`">
    <span class="version-details__version">{{ displayVersion }}</span>
    <span class="version-details__runtime">{{ runtimeText }}</span>
    <span v-if="hasNewVersion" class="version-details__update">
      新版本 {{ overview?.version.latest }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { useBaseOverview } from '@/features/base/useBaseOverview';

const props = withDefaults(
  defineProps<{
    /** stacked 用于顶栏两行排布，inline 用于浮窗内的紧凑单行。 */
    layout?: 'stacked' | 'inline';
  }>(),
  { layout: 'stacked' }
);

const { overview, displayVersion, runtimeText, hasNewVersion } = useBaseOverview();
</script>

<style scoped>
.version-details {
  display: grid;
  min-width: 0;
  cursor: default;
}

/* 版本号是主信息，运行环境是它的补充，用字号与色阶区分而非两种颜色。 */
.version-details__version {
  color: var(--sd-text-primary);
  font-size: 0.82rem;
  line-height: 1.25;
  white-space: nowrap;
}

.version-details__runtime {
  color: var(--sd-text-muted);
  font-size: 0.72rem;
  line-height: 1.25;
  white-space: nowrap;
}

/* 发现新版本是信息而非风险，但仍需引起注意，沿用顶栏既有的 warning 色。 */
.version-details__update {
  color: var(--sd-warning);
  font-size: 0.72rem;
  line-height: 1.25;
  white-space: nowrap;
}

.version-details--stacked {
  justify-items: end;
  text-align: right;
}

.version-details--inline {
  justify-items: start;
  gap: 0.1rem;
  text-align: left;
}

.version-details--inline .version-details__version {
  font-size: 0.85rem;
}
</style>
