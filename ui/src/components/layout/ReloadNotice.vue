<template>
  <n-affix v-if="show" :top="top">
    <TipBox type="error">
      <slot>
        <n-text type="error" class="text-base" tag="strong">{{ text }}</n-text>
      </slot>
    </TipBox>
  </n-affix>
</template>

<script setup lang="ts">
import TipBox from '@/components/shared/TipBox.vue';

withDefaults(defineProps<{
  show: boolean;
  text?: string;
  top?: number;
}>(), {
  text: '存在修改，需要重载后生效！',
  top: 60,
});
</script>
