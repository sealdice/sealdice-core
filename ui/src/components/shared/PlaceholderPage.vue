<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();
const title = computed(() => {
  if (typeof route.meta.title === 'string') {
    return route.meta.title;
  }
  return '页面迁移中';
});
</script>

<template>
  <section class="placeholder-page">
    <n-card :bordered="false">
      <p class="eyebrow">SealDice UI</p>
      <h1>{{ title }}</h1>
      <p>此页面已接入旧版导航结构，后续将按 v2 API 覆盖情况迁移业务内容。</p>
    </n-card>
  </section>
</template>

<style scoped>
.placeholder-page {
  max-width: 920px;
}

.eyebrow {
  margin: 0 0 8px;
  color: var(--sd-primary);
  font-size: 12px;
}

h1 {
  margin: 0 0 12px;
  color: var(--sd-text-primary);
  font-size: 26px;
  font-weight: 700;
}

p {
  margin: 0;
  color: var(--sd-text-secondary);
  line-height: 1.8;
}
</style>
