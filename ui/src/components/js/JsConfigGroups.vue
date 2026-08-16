<template>
  <ResponsiveTabs
    v-if="groups.length"
    v-model:value="activeGroup"
    class="js-config-groups"
    :compact-at="600"
    :options="groupOptions"
  >
    <template #panel="{ option }">
      <JsConfigItemEditor
        v-for="item in findGroup(option.value)?.items ?? []"
        :key="item.key"
        :item="item"
        :plugin-name="pluginName"
        :error-text="configErrors[buildConfigErrorKey(pluginName, item.key)]"
        :checking="!!checkingKeys[buildConfigErrorKey(pluginName, item.key)]"
        @change="forwardChange"
        @reset="forwardReset"
        @validate="forwardValidate"
      />
    </template>
  </ResponsiveTabs>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { ConfigItem } from '@/api';
import JsConfigItemEditor from '@/components/js/JsConfigItemEditor.vue';
import ResponsiveTabs from '@/components/shared/ResponsiveTabs.vue';
import {
  buildConfigErrorKey,
  groupPluginConfigItems,
  type JsConfigErrorMap,
} from '@/features/js/configModel';

const props = defineProps<{
  pluginName: string;
  items: ConfigItem[];
  configErrors: JsConfigErrorMap;
  checkingKeys: Record<string, boolean>;
}>();

const emit = defineEmits<{
  change: [pluginName: string, key: string, value: unknown];
  reset: [pluginName: string, key: string];
  validate: [pluginName: string, key: string, value: string, type: string];
}>();

const groups = computed(() => groupPluginConfigItems(props.items));
const groupOptions = computed(() =>
  groups.value.map(group => ({
    label: group.name,
    value: group.name,
  }))
);
const activeGroup = ref('');

watch(
  groups,
  nextGroups => {
    if (!nextGroups.some(group => group.name === activeGroup.value)) {
      activeGroup.value = nextGroups[0]?.name ?? '';
    }
  },
  { immediate: true }
);

function findGroup(name: string) {
  return groups.value.find(group => group.name === name);
}

function forwardChange(pluginName: string, key: string, value: unknown) {
  emit('change', pluginName, key, value);
}

function forwardReset(pluginName: string, key: string) {
  emit('reset', pluginName, key);
}

function forwardValidate(pluginName: string, key: string, value: string, type: string) {
  emit('validate', pluginName, key, value, type);
}
</script>
