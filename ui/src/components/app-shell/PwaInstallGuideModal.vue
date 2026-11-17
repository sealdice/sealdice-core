<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="安装为应用"
    :bordered="false"
    :segmented="{ content: true }"
    style="width: min(32rem, calc(100vw - 2rem))"
  >
    <n-flex vertical :size="16">
      <div>
        <n-h3 prefix="bar" class="guide-title">{{ guide.title }}</n-h3>
        <n-text depth="3">{{ guide.description }}</n-text>
      </div>

      <ol class="guide-steps">
        <li v-for="step in guide.steps" :key="step">{{ step }}</li>
      </ol>

      <n-alert v-if="guide.warning" type="warning" :show-icon="true">
        {{ guide.warning }}
      </n-alert>
    </n-flex>

    <template #footer>
      <n-flex justify="end">
        <n-button type="primary" @click="show = false">知道了</n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import type { PwaInstallGuide } from '@/features/pwa/pwaInstallGuide';

const show = defineModel<boolean>('show', { required: true });

defineProps<{
  guide: PwaInstallGuide;
}>();
</script>

<style scoped>
.guide-title {
  margin: 0 0 0.5rem;
}

.guide-steps {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  margin: 0;
  padding-left: 1.4rem;
}

.guide-steps li {
  padding-left: 0.2rem;
}
</style>
