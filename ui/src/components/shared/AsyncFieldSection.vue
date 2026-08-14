<template>
  <div class="async-field-section">
    <!-- 常驻错误块必须带重试，否则用户只能看着它停在页面上。 -->
    <TipBox v-if="props.error" type="error" class="section-alert">
      <n-flex justify="space-between" align="center" :wrap="false">
        <span>{{ props.error }}</span>
        <n-button size="small" tertiary @click="emit('retry')">
          {{ props.retryText }}
        </n-button>
      </n-flex>
    </TipBox>

    <n-spin :show="props.loading">
      <div class="section-body">
        <n-text v-if="props.message" depth="3" class="section-message">
          {{ props.message }}
        </n-text>
        <slot />
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import TipBox from '@/components/shared/TipBox.vue';

const props = withDefaults(
  defineProps<{
    loading?: boolean;
    message?: string;
    error?: string;
    retryText?: string;
  }>(),
  {
    loading: false,
    message: '',
    error: '',
    retryText: '重试',
  }
);

const emit = defineEmits<{
  retry: [];
}>();
</script>

<style scoped>
.async-field-section {
  width: 100%;
}

.section-alert {
  margin-bottom: 0.75rem;
}

.section-body {
  width: 100%;
}

.section-message {
  display: block;
  font-size: 0.82rem;
  line-height: 1.45;
  margin-bottom: 0.5rem;
}
</style>
