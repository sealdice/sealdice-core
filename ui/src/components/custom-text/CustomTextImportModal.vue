<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="导入导出"
    :mask-closable="false"
    :close-on-esc="false"
    :closable="false"
    class="the-dialog custom-text-import-modal"
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

    <n-flex vertical class="import-body">
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
  flex: none;
}

/* 让文本框占满卡片剩余高度：内容再长也只在文本框内部滚动，
   避免自动高度把整个弹窗顶出视口、页脚按钮不可见。 */
.import-body {
  min-height: 0;
  flex: 1 1 auto;
}

.import-edit {
  min-height: 0;
  flex: 1 1 auto;
}

/* 卡片根节点由 naive-ui 渲染，作用域样式无法命中，使用全局选择器限定在当前弹窗。 */
:global(.custom-text-import-modal.n-card) {
  max-height: calc(100vh - 48px);
  max-height: calc(100dvh - 48px);
}

:global(.custom-text-import-modal > .n-card-content) {
  display: flex;
  min-height: 0;
  flex-direction: column;
}

:global(.custom-text-import-modal > .n-card-header),
:global(.custom-text-import-modal > .n-card__footer) {
  flex: none;
}
</style>
