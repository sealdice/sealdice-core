<template>
  <RepeatableList add-label="添加通知目标" @add="addTarget">
    <RepeatableItem
      v-for="(target, index) in targets"
      :key="index"
      :title="`通知目标 ${index + 1}`"
      show-enabled
      :enabled="target.enabled"
      enabled-label="启用通知目标"
      remove-label="删除通知目标"
      @update:enabled="updateTarget(index, { enabled: $event })"
      @remove="removeTarget(index)"
    >
      <div class="notice-target-fields">
        <RepeatableFieldRow class="notice-target-field notice-target-field--id" label="ID">
          <n-input
            class="notice-target-id"
            :value="target.id"
            placeholder="平台:账号或群组 ID"
            @update:value="updateTarget(index, { id: $event })"
          />
        </RepeatableFieldRow>
        <RepeatableFieldRow class="notice-target-field notice-target-field--types" label="通知项">
          <n-select
            class="notice-target-types"
            :value="target.noticeTypes"
            :options="noticeTypeOptions"
            multiple
            clearable
            max-tag-count="responsive"
            @update:value="updateTarget(index, { noticeTypes: $event as NoticeType[] })"
          />
        </RepeatableFieldRow>
      </div>
    </RepeatableItem>
  </RepeatableList>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import RepeatableFieldRow from '@/components/settings-panel/RepeatableFieldRow.vue';
import RepeatableItem from '@/components/shared/RepeatableItem.vue';
import RepeatableList from '@/components/shared/RepeatableList.vue';
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

<style scoped>
.notice-target-fields {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sd-space-md) var(--sd-space-lg);
}

.notice-target-field--id {
  flex: 0 1 22.5rem;
  min-width: min(100%, 14rem);
}

.notice-target-field--types {
  flex: 1 1 22rem;
  max-width: 34rem;
  min-width: min(100%, 16rem);
}

.notice-target-id,
.notice-target-types {
  width: 100%;
}
</style>
