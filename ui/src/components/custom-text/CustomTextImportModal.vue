<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="导入导出"
    :mask-closable="false"
    :close-on-esc="false"
    :closable="false"
    class="the-dialog"
  >
    <n-flex class="import-options" align="center" justify="space-between" size="small" wrap>
      <n-flex align="center" size="small" wrap>
        <n-text depth="3">文案范围</n-text>
        <n-radio-group v-model:value="onlyCurrent" size="small" aria-label="文案范围">
          <n-radio-button :value="false">全部文案</n-radio-button>
          <n-radio-button :value="true">仅当前页面</n-radio-button>
        </n-radio-group>
      </n-flex>
      <n-checkbox v-model:checked="compact">紧凑</n-checkbox>
    </n-flex>

    <n-flex vertical>
      <n-text tag="strong">以下为导出内容，可以复制给别人</n-text>
      <n-input
        v-model:value="content"
        placeholder="填入数据"
        type="textarea"
        :autosize="{ minRows: 4 }"
        class="import-edit sd-code-text"
        id="import-edit"
      />
    </n-flex>

    <template #footer>
      <n-flex>
        <n-button @click="show = false">返回</n-button>
        <n-button type="error" secondary @click="emit('clear')">清空</n-button>
        <n-button secondary @click="emit('copy')">复制</n-button>
        <n-button
          type="primary"
          :loading="saving"
          :disabled="content === ''"
          @click="emit('import')"
        >
          导入并保存
        </n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
const show = defineModel<boolean>('show', { required: true });
const content = defineModel<string>('content', { required: true });
const onlyCurrent = defineModel<boolean>('onlyCurrent', { required: true });
const compact = defineModel<boolean>('compact', { required: true });

defineProps<{
  saving: boolean;
}>();

const emit = defineEmits<{
  copy: [];
  clear: [];
  import: [];
}>();
</script>

<style scoped>
.import-options {
  margin-bottom: var(--sd-space-sm);
}

.import-edit :deep(textarea) {
  max-height: 65vh;
}
</style>
