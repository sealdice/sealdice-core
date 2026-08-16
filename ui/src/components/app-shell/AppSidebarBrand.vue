<template>
  <div class="sd-sidebar-brand" :class="{ collapsed: props.collapsed }">
    <button
      type="button"
      class="brand-mark"
      aria-label="启用高级配置"
      @click="emit('enableAdvancedConfig')"
    >
      <img :src="sealdiceD20" alt="" class="brand-mark-image" />
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
                  <i-tabler-package />
                </n-icon>
              </template>
            </n-button>
          </template>
          当前以容器模式启动，部分功能受到限制。
        </n-tooltip>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import sealdiceD20 from '@/assets/sealdice-d20.svg';
import { useBaseOverview } from '@/features/base/useBaseOverview';

const props = withDefaults(
  defineProps<{
    collapsed?: boolean;
  }>(),
  {
    collapsed: false,
  }
);

const emit = defineEmits<{
  enableAdvancedConfig: [];
}>();

const { overview, appName } = useBaseOverview();
</script>

<style scoped>
.sd-sidebar-brand {
  display: flex;
  min-height: 70px;
  align-items: center;
  gap: 0.65rem;
  margin-bottom: 0.65rem;
  padding: 1rem 1rem 0.75rem;
  border-bottom: 1px solid var(--sd-border-sidebar);
  color: var(--sd-text-inverse);
  line-height: 1;
}

.sd-sidebar-brand.collapsed {
  justify-content: center;
  min-height: 70px;
  margin-bottom: 0.65rem;
  padding: 1rem 0.5rem 0.75rem;
}

.brand-mark {
  display: inline-flex;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--sd-radius-sm);
  background: var(--sd-primary);
  background: linear-gradient(
    145deg,
    color-mix(in srgb, var(--sd-primary) 76%, var(--sd-bg-elevated)),
    color-mix(in srgb, var(--sd-primary) 54%, var(--sd-bg-elevated))
  );
  cursor: pointer;
  margin: 0;
  padding: 0;
  transition:
    background-color var(--sd-transition-fast),
    filter var(--sd-transition-fast),
    transform var(--sd-transition-fast);
}

.brand-mark:hover {
  filter: brightness(1.05);
}

.brand-mark:active {
  transform: translateY(1px);
}

.brand-mark-image {
  display: block;
  width: 30px;
  height: 30px;
}

.brand-info {
  flex: 1;
  min-width: 0;
}

.brand-title-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.2rem;
}

/* 产品名与 logo 同为品牌标识，字号取到与图标块同一视觉层级。 */
.brand-title {
  overflow: hidden;
  border: 0;
  background: transparent;
  color: var(--sd-text-inverse);
  cursor: pointer;
  font: inherit;
  font-size: 1.3rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.1;
  padding: 0;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-indicator {
  color: var(--sd-text-inverse-soft);
}

@supports (color: color-mix(in srgb, white, black)) {
  .container-indicator {
    color: color-mix(in srgb, var(--sd-text-inverse), transparent 14%);
  }
}
</style>
