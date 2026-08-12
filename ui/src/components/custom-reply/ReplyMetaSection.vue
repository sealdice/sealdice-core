<template>
  <header class="reply-meta-section">
    <PageHeader title="自定义回复" description="管理回复文件、条件和回复规则。">
      <n-switch
        :value="replyEnabled"
        :loading="switchLoading"
        @update:value="emit('toggleReplyEnabled', $event)"
      >
        <template #checked>启用</template>
        <template #unchecked>关闭</template>
        总开关
      </n-switch>

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
