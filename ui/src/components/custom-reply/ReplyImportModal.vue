<script setup lang="ts">
const show = defineModel<boolean>('show', { required: true });
const content = defineModel<string>('content', { required: true });

defineProps<{
  disabled: boolean;
}>();

const emit = defineEmits<{
  import: [];
}>();
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="导入配置"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    class="the-dialog"
  >
    <n-input
      v-model:value="content"
      placeholder="支持格式: 关键字/回复语"
      class="reply-text"
      type="textarea"
      :autosize="{ minRows: 4, maxRows: 10 }"
    />
    <template #footer>
      <n-flex>
        <n-button @click="show = false">取消</n-button>
        <n-button type="primary" :disabled="disabled || content === ''" @click="emit('import')">
          下一步
        </n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<style scoped>
.reply-text :deep(textarea) {
  max-height: 65vh;
}
</style>
