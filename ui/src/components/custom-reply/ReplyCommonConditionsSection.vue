<template>
  <section class="reply-section">
    <div class="section-head">
      <div>
        <h3>前置条件</h3>
        <p>该文件下所有规则执行前都需要先满足这些条件。</p>
      </div>
      <n-space size="small" align="center">
        <n-select
          class="vm-version-select"
          size="small"
          :value="vmVersion"
          :options="vmVersionOptions"
          :consistent-menu-width="false"
          @update:value="emit('updateVmVersion', $event as ReplyVMVersion)"
        />
        <n-button
          size="small"
          :type="fileEnabled ? 'success' : 'warning'"
          secondary
          @click="emit('toggleFileEnabled')"
        >
          <template #icon>
            <n-icon>
              <i-tabler-circle-check-filled v-if="fileEnabled" />
              <i-tabler-circle-x v-else />
            </n-icon>
          </template>
          {{ fileEnabled ? '文件已启用' : '文件未启用' }}
        </n-button>
        <n-button size="small" secondary @click="emit('add')">
          <template #icon>
            <n-icon><i-tabler-plus /></n-icon>
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
      <n-pagination v-model:page="pageModel" :page-size="pageSize" :item-count="total" simple />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ConditionBuilder from './ConditionBuilder.vue';
import type { ReplyCondition, ReplyVMVersion } from '@/features/customReply/model';

const conditions = defineModel<ReplyCondition[]>({ required: true });
const props = defineProps<{
  fileEnabled: boolean;
  vmVersion: ReplyVMVersion;
  page: number;
  pageSize: number;
  total: number;
}>();

const emit = defineEmits<{
  add: [];
  delete: [index: number];
  toggleFileEnabled: [];
  updateVmVersion: [value: ReplyVMVersion];
  updatePage: [page: number];
}>();

const vmVersionOptions = [
  { label: 'VM V2', value: 'v2' },
  { label: 'VM V1', value: 'v1' },
] satisfies Array<{ label: string; value: ReplyVMVersion }>;

const pageModel = computed({
  get: () => props.page,
  set: value => emit('updatePage', value),
});
</script>

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

.vm-version-select {
  width: 6.5rem;
}

@media screen and (max-width: 1023.9px) {
  .section-head {
    flex-direction: column;
  }
}
</style>
