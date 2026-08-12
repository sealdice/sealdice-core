<template>
  <n-popover trigger="click" placement="bottom-end" :show-arrow="false">
    <template #trigger>
      <n-button quaternary circle aria-label="更多操作">
        <template #icon>
          <n-icon size="1.2rem">
            <i-tabler-dots-vertical />
          </n-icon>
        </template>
      </n-button>
    </template>

    <div class="header-overflow" aria-label="更多操作">
      <AppInstallButton />
      <n-button tag="a" secondary block :href="oldUIUrl"> 回退老 UI </n-button>
      <n-divider class="header-overflow__divider" />
      <div class="header-overflow__version">
        <n-tag :bordered="false" :type="isStable ? 'success' : 'default'" size="small">
          {{ isStable ? '正式版' : '测试版' }}
        </n-tag>
        <n-tooltip placement="left">
          <template #trigger>
            <span class="header-overflow__version-text">
              {{ overview?.version.simple ?? '-' }}
            </span>
          </template>
          {{ overview?.version.value ?? '-' }}
        </n-tooltip>
      </div>
      <n-text v-if="hasNewVersion" depth="3" class="header-overflow__update">
        新版本 {{ overview?.version.latest }}
      </n-text>
    </div>
  </n-popover>
</template>

<script setup lang="ts">
import { resolveOldUIUrlFromLocation } from '@/api/config';
import { useBaseOverview } from '@/features/base/useBaseOverview';
import AppInstallButton from './AppInstallButton.vue';

const { overview, isStable, hasNewVersion } = useBaseOverview();
const oldUIUrl =
  typeof window !== 'undefined' ? resolveOldUIUrlFromLocation(window.location) : '/old-ui/';
</script>

<style scoped>
.header-overflow {
  display: grid;
  width: 13rem;
  gap: 0.75rem;
}

.header-overflow :deep(.install-trigger),
.header-overflow :deep(.install-button),
.header-overflow :deep(.install-tag) {
  width: 100%;
}

.header-overflow :deep(.install-label) {
  display: inline;
}

.header-overflow__divider {
  margin: 0;
}

.header-overflow__version {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.header-overflow__version-text,
.header-overflow__update {
  font-size: 0.82rem;
}

.header-overflow__version-text {
  color: var(--sd-text-secondary);
}
</style>
