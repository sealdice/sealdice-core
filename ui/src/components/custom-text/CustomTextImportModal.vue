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
    <template #header-extra>
      <n-flex>
        <n-switch v-model:value="onlyCurrent">
          <template #checked>仅当前页面</template>
          <template #unchecked>全部文案</template>
        </n-switch>
        <n-checkbox v-model:checked="compact">紧凑</n-checkbox>
      </n-flex>
    </template>
    <n-flex vertical>
      <n-text tag="strong">以下为导出内容，可以复制给别人</n-text>
      <n-input
        v-model:value="content"
        placeholder="填入数据"
        type="textarea"
        :autosize="{ minRows: 4 }"
        class="import-edit"
        id="import-edit"
      />
    </n-flex>

    <template #footer>
      <n-flex>
        <n-button @click="show = false">返回</n-button>
        <n-button type="warning" @click="emit('clear')">清空</n-button>
        <n-button type="info" @click="emit('copy')">复制</n-button>
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

<style scoped>
.import-edit :deep(textarea) {
  max-height: 65vh;
}
</style>
