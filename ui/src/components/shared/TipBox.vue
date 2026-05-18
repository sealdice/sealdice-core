<script setup lang="ts">
import { computed } from 'vue';
import { useThemeVars } from 'naive-ui';

interface Props {
  type?: 'default' | 'success' | 'info' | 'warning' | 'error';
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
});

const themeVars = useThemeVars();

const borderColor = computed(() => {
  switch (props.type) {
    case 'success':
      return themeVars.value.successColor;
    case 'info':
      return themeVars.value.infoColor;
    case 'warning':
      return themeVars.value.warningColor;
    case 'error':
      return themeVars.value.errorColor;
    default:
      return themeVars.value.primaryColor;
  }
});

const bgColor = computed(() => `${borderColor.value}10`);
</script>

<template>
  <div class="tip-box">
    <slot :type="props.type" />
  </div>
</template>

<style scoped>
.tip-box {
  border-left: 3px solid v-bind('borderColor');
  border-radius: 2px;
  background: v-bind('bgColor');
  padding: 1rem;
}
</style>
