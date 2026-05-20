<script setup lang="ts">
import type { BaseSettingExtDefaultSettingItem } from '@/api';

const model = defineModel<BaseSettingExtDefaultSettingItem[]>({ required: true });

function toggleDisabledCommand(itemIndex: number, command: string) {
  const next = structuredClone(model.value);
  next[itemIndex]!.disabledCommand[command] = !next[itemIndex]!.disabledCommand[command];
  model.value = next;
}

function updateAutoActive(itemIndex: number, value: boolean) {
  const next = structuredClone(model.value);
  next[itemIndex]!.autoActive = value;
  model.value = next;
}
</script>

<template>
  <n-list>
    <n-list-item v-for="(item, index) in model" :key="item.name">
      <n-flex vertical class="mx-4">
        <span>扩展：{{ item.name }}</span>
        <span>禁用指令</span>
        <n-flex size="small">
          <n-button
            v-for="(disabled, command) in item.disabledCommand"
            :key="command"
            :type="disabled ? 'error' : 'default'"
            size="tiny"
            @click="toggleDisabledCommand(index, String(command))"
          >
            {{ command }}
          </n-button>
        </n-flex>
        <div>
          <n-checkbox
            :checked="item.autoActive"
            @update:checked="updateAutoActive(index, $event)"
          >
            入群自动开启
          </n-checkbox>
        </div>
      </n-flex>
    </n-list-item>
  </n-list>
</template>
