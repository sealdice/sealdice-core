<template>
  <div class="tip-box" :class="`tip-box--${props.type}`">
    <n-icon v-if="iconVisible" class="tip-box__icon" :size="16" aria-hidden="true">
      <i-tabler-alert-triangle v-if="props.type === 'warning'" />
      <i-tabler-alert-circle v-else-if="props.type === 'error'" />
      <i-tabler-circle-check v-else />
    </n-icon>

    <div class="tip-box__body">
      <slot :type="props.type" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  type?: 'default' | 'success' | 'info' | 'warning' | 'error';
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
});

// warning / error / success 必须有非颜色通道，颜色不能是区分状态的唯一方式。
// info 与 default 是解释性的，标题与正文已自解释，加图标只是装饰。
const iconVisible = computed(
  () => props.type === 'warning' || props.type === 'error' || props.type === 'success'
);
</script>

<style scoped>
.tip-box {
  display: flex;
  /* 图标相对整个提示框垂直居中。规范限定「一条提示只讲一个主题」，
     正文不会长到让居中的图标显得脱离文字。 */
  align-items: center;
  gap: var(--sd-space-xs);
  border-left: 3px solid var(--tip-box-accent);
  border-radius: var(--sd-radius-xs);
  background: var(--tip-box-surface);
  padding: var(--sd-space-sm) var(--sd-space-md);
  color: var(--sd-text-primary);
  line-height: 1.6;
}

.tip-box--default,
.tip-box--info {
  --tip-box-accent: var(--sd-info);
  --tip-box-surface: var(--sd-primary-soft);
}

.tip-box--success {
  --tip-box-accent: var(--sd-success);
  --tip-box-surface: var(--sd-success-soft);
}

.tip-box--warning {
  --tip-box-accent: var(--sd-warning);
  --tip-box-surface: var(--sd-warning-soft);
}

.tip-box--error {
  --tip-box-accent: var(--sd-error);
  --tip-box-surface: var(--sd-error-soft);
}

.tip-box__icon {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  color: var(--tip-box-accent);
}

.tip-box__body {
  min-width: 0;
  flex: 1 1 auto;
}

.tip-box__body :deep(p) {
  margin: 0;
}

.tip-box__body :deep(p + p),
.tip-box__body :deep(ul),
.tip-box__body :deep(ol) {
  margin-top: var(--sd-space-xs);
}
</style>
