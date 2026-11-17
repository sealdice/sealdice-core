<template>
  <TipBox type="info" class="help-collapse">
    <n-collapse :accordion="accordion">
      <n-collapse-item :name="name">
        <template #header>
          <n-text tag="strong">{{ title }}</n-text>
        </template>
        <slot />
      </n-collapse-item>
    </n-collapse>
  </TipBox>
</template>

<script setup lang="ts">
import TipBox from '@/components/shared/TipBox.vue';

withDefaults(
  defineProps<{
    title?: string;
    name?: string;
    accordion?: boolean;
  }>(),
  {
    title: '查看帮助',
    name: 'help',
    accordion: false,
  }
);
</script>

<style scoped>
/* 折叠内容自带上边距，去掉 naive 默认分割线以免通栏切断 TipBox 圆角。 */
.help-collapse :deep(.n-collapse-item) {
  margin-top: 0;
}

.help-collapse :deep(.n-collapse-item:not(:first-child)) {
  border-top: none;
}

.help-collapse :deep(.n-collapse-item__content-inner) {
  padding-top: var(--sd-space-xs);
}
</style>
