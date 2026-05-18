<script setup lang="ts">
import { useBaseOverview } from '@/features/base/useBaseOverview';

const props = withDefaults(
  defineProps<{
    collapsed?: boolean;
  }>(),
  {
    collapsed: false,
  },
);

const emit = defineEmits<{
  enableAdvancedConfig: [];
}>();

const { overview, appName, runtimeText } = useBaseOverview();
</script>

<template>
  <div class="sd-sidebar-brand" :class="{ collapsed: props.collapsed }">
    <button type="button" class="brand-mark" @click="emit('enableAdvancedConfig')">
      <span class="brand-mark-inner">S</span>
    </button>

    <div v-if="!props.collapsed" class="brand-info">
      <div class="brand-title-row">
        <button type="button" class="brand-title" @click="emit('enableAdvancedConfig')">
          {{ appName }}
        </button>
        <n-tooltip v-if="overview?.runtime.containerMode">
          <template #trigger>
            <n-button text size="tiny" class="container-indicator">
              <template #icon>
                <n-icon>
                  <i-carbon-container-software />
                </n-icon>
              </template>
            </n-button>
          </template>
          当前以容器模式启动，部分功能受到限制。
        </n-tooltip>
      </div>

      <div v-if="runtimeText" class="runtime-text">
        {{ runtimeText }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.sd-sidebar-brand {
  display: flex;
  min-height: 82px;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 0.85rem 0.8rem;
  color: var(--sd-text-inverse);
}

.sd-sidebar-brand.collapsed {
  justify-content: center;
  min-height: 70px;
  padding: 0.85rem 0.5rem;
}

.brand-mark {
  display: inline-flex;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 10px;
  background: linear-gradient(135deg, #22c55e, #0ea5e9 52%, #f97316);
  cursor: pointer;
  padding: 0;
}

.brand-mark-inner {
  color: var(--sd-text-inverse);
  font-size: 1.2rem;
  font-weight: 700;
  line-height: 1;
}

.brand-info {
  min-width: 0;
}

.brand-title-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.25rem;
}

.brand-title {
  overflow: hidden;
  border: 0;
  background: transparent;
  color: var(--sd-text-inverse);
  cursor: pointer;
  font: inherit;
  font-size: 1rem;
  line-height: 1.25;
  padding: 0;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-indicator {
  color: color-mix(in srgb, var(--sd-text-inverse), transparent 14%);
}

.runtime-text {
  overflow: hidden;
  color: color-mix(in srgb, var(--sd-text-inverse), transparent 32%);
  font-size: 0.7rem;
  line-height: 1.35;
  margin-top: 0.15rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
