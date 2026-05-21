<script setup lang="ts">
import { computed } from 'vue';
import NestedRuleEditor from '@/components/shared/NestedRuleEditor.vue';
import type { ReplyTask } from '@/features/customReply/model';

const rules = defineModel<ReplyTask[]>({ required: true });
const props = defineProps<{
  startIndex: number;
  page: number;
  pageSize: number;
  total: number;
}>();

const emit = defineEmits<{
  add: [];
  change: [];
  delete: [index: number];
  updatePage: [page: number];
}>();

const pageModel = computed({
  get: () => props.page,
  set: value => emit('updatePage', value),
});
</script>

<template>
  <section class="reply-section reply-section--rules">
    <div class="section-head">
      <div>
        <h3>规则列表</h3>
        <p>从上到下匹配。当前页只显示部分规则，但保存会提交整份文件。</p>
      </div>
      <n-space size="small" align="center">
        <n-button size="small" secondary @click="emit('add')">
          <template #icon>
            <n-icon><i-carbon-add-large /></n-icon>
          </template>
          添加规则
        </n-button>
      </n-space>
    </div>

    <div class="section-body">
      <NestedRuleEditor
        :tasks="rules"
        :start-index="startIndex"
        @change="emit('change')"
        @delete-rule="emit('delete', $event)"
      />
    </div>

    <div class="section-footer">
      <n-pagination
        v-model:page="pageModel"
        :page-size="pageSize"
        :item-count="total"
        simple
      />
    </div>
  </section>
</template>

<style scoped>
.reply-section {
  flex: 0 0 auto;
  border: 0;
  border-bottom: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
}

.reply-section--rules {
  flex: 1 1 auto;
  min-height: 0;
  border-bottom: 0;
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--sd-border-soft);
  padding: 1rem;
}

.section-head h3 {
  margin: 0;
  font-size: 1rem;
}

.section-head p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.85rem;
}

.section-body {
  padding: 1rem;
  min-width: 0;
}

.section-footer {
  border-top: 1px solid var(--sd-border-soft);
  padding: 0.75rem 1rem;
}

@media screen and (max-width: 1023.9px) {
  .section-head {
    flex-direction: column;
  }
}
</style>
