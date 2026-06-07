<template>
  <header class="reply-meta-section">
    <n-flex align="center" justify="space-between" wrap>
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
          <n-icon><i-carbon-save /></n-icon>
        </template>
        保存
      </n-button>
    </n-flex>
  </header>
</template>

<script setup lang="ts">
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
  margin-bottom: 1rem;
}
</style>
