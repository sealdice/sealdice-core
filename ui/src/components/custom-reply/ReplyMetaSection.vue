<template>
  <header class="reply-meta-section">
    <PageHeader title="自定义回复">
      <n-flex align="center" size="small">
        <n-text depth="3">启用自定义回复</n-text>
        <n-switch
          :value="replyEnabled"
          :loading="switchLoading"
          aria-label="启用自定义回复"
          @update:value="emit('toggleReplyEnabled', $event)"
        />
      </n-flex>

      <n-button
        v-if="replyEnabled"
        type="primary"
        :loading="saveLoading"
        :disabled="saveDisabled"
        @click="emit('save')"
      >
        <template #icon>
          <n-icon><i-tabler-device-floppy /></n-icon>
        </template>
        保存
      </n-button>
    </PageHeader>
  </header>
</template>

<script setup lang="ts">
import PageHeader from '@/components/shared/PageHeader.vue';

defineProps<{
  replyEnabled: boolean;
  switchLoading: boolean;
  saveLoading: boolean;
  saveDisabled: boolean;
}>();

const emit = defineEmits<{
  toggleReplyEnabled: [value: boolean];
  save: [];
}>();
</script>

<style scoped>
.reply-meta-section {
  min-width: 0;
}
</style>
