<script setup lang="ts">
import { computed } from 'vue';
import ConditionBuilder from '@/components/shared/ConditionBuilder.vue';
import type { ReplyCondition } from '@/features/customReply/model';

const conditions = defineModel<ReplyCondition[]>({ required: true });
const props = defineProps<{
  fileEnabled: boolean;
  page: number;
  pageSize: number;
  total: number;
}>();

const emit = defineEmits<{
  add: [];
  delete: [index: number];
  toggleFileEnabled: [];
  updatePage: [page: number];
}>();

const pageModel = computed({
  get: () => props.page,
  set: value => emit('updatePage', value),
});
</script>

<template>
  <section class="reply-section">
    <div class="section-head">
      <div>
        <h3>前置条件</h3>
        <p>该文件下所有规则执行前都需要先满足这些条件。</p>
      </div>
      <n-space size="small" align="center">
        <n-button
          size="small"
          :type="fileEnabled ? 'success' : 'warning'"
          secondary
          @click="emit('toggleFileEnabled')"
        >
          <template #icon>
            <n-icon>
              <i-carbon-checkmark-filled v-if="fileEnabled" />
              <i-carbon-close-outline v-else />
            </n-icon>
          </template>
          {{ fileEnabled ? '文件已启用' : '文件未启用' }}
        </n-button>
        <n-button size="small" secondary @click="emit('add')">
          <template #icon>
            <n-icon><i-carbon-add-large /></n-icon>
          </template>
          添加条件
        </n-button>
      </n-space>
    </div>

    <div class="section-body">
      <ConditionBuilder
        v-if="conditions.length"
        v-model="conditions"
        @delete-condition="emit('delete', $event)"
      />
      <n-empty v-else description="当前无前置条件" size="small" />
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
