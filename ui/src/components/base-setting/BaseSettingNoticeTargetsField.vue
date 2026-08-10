<script setup lang="ts">
import { ref, watch } from 'vue';
import {
  noticeTypeOptions,
  parseNoticeTarget,
  serializeNoticeTarget,
  type NoticeTargetModel,
  type NoticeType,
} from '@/features/baseSetting/noticeTargets';

const model = defineModel<string[]>({ required: true });

const targets = ref<NoticeTargetModel[]>(model.value.map(parseNoticeTarget));

watch(
  () => model.value,
  value => {
    if (JSON.stringify(value) === JSON.stringify(serializeTargets(targets.value))) return;
    targets.value = value.map(parseNoticeTarget);
  },
  { deep: true }
);

function serializeTargets(items: NoticeTargetModel[]) {
  return items.map(serializeNoticeTarget).filter(Boolean);
}

function commit(items: NoticeTargetModel[]) {
  targets.value = items;
  model.value = serializeTargets(items);
}

function updateTarget(index: number, patch: Partial<NoticeTargetModel>) {
  commit(
    targets.value.map((target, targetIndex) =>
      targetIndex === index ? { ...target, ...patch } : target
    )
  );
}

function addTarget() {
  commit([
    ...targets.value,
    { id: '', enabled: true, noticeTypes: noticeTypeOptions.map(option => option.value) },
  ]);
}

function removeTarget(index: number) {
  commit(targets.value.filter((_, targetIndex) => targetIndex !== index));
}
</script>

<template>
  <div class="notice-targets">
    <div v-for="(target, index) in targets" :key="index" class="notice-target-row">
      <n-switch
        :value="target.enabled"
        size="small"
        @update:value="updateTarget(index, { enabled: $event })"
      />
      <n-input
        class="notice-target-id"
        :value="target.id"
        placeholder="平台:账号或群组 ID"
        @update:value="updateTarget(index, { id: $event })"
      />
      <n-select
        class="notice-target-types"
        :value="target.noticeTypes"
        :options="noticeTypeOptions"
        multiple
        clearable
        max-tag-count="responsive"
        @update:value="updateTarget(index, { noticeTypes: $event as NoticeType[] })"
      />
      <n-tooltip>
        <template #trigger>
          <n-button
            quaternary
            circle
            type="error"
            aria-label="删除通知目标"
            @click="removeTarget(index)"
          >
            <template #icon><i-ep-delete /></template>
          </n-button>
        </template>
        删除通知目标
      </n-tooltip>
    </div>

    <n-button dashed class="notice-target-add" @click="addTarget">
      <template #icon><i-ep-plus /></template>
      添加通知目标
    </n-button>
  </div>
</template>

<style scoped>
.notice-targets {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 0.625rem;
}

.notice-target-row {
  display: grid;
  grid-template-columns: auto minmax(12rem, 0.9fr) minmax(18rem, 1.4fr) auto;
  align-items: center;
  gap: 0.625rem;
}

.notice-target-id,
.notice-target-types {
  min-width: 0;
}

.notice-target-add {
  align-self: flex-start;
}

@media screen and (max-width: 767.9px) {
  .notice-target-row {
    grid-template-columns: auto minmax(0, 1fr) auto;
  }

  .notice-target-types {
    grid-column: 2 / 4;
  }
}
</style>
