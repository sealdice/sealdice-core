<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="设置帮助文档"
    class="the-dialog"
    :mask-closable="false"
  >
    <h4 class="config-title">分组别名</h4>
    <n-form label-placement="left">
      <n-form-item v-for="group in groups" :key="group.value" :label="group.label">
        <HelpConfigTags
          :group="group"
          :aliases="aliases"
          @add-alias="(groupKey, alias) => emit('addAlias', groupKey, alias)"
          @remove-alias="(groupKey, alias) => emit('removeAlias', groupKey, alias)"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-flex justify="end">
        <n-button @click="show = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="emit('save')">
          保存
        </n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import HelpConfigTags from './HelpConfigTags.vue';

const show = defineModel<boolean>('show', { required: true });

defineProps<{
  groups: { label: string; value: string }[];
  aliases: Map<string, string[]>;
  saving: boolean;
}>();

const emit = defineEmits<{
  save: [];
  addAlias: [group: string, alias: string];
  removeAlias: [group: string, alias: string];
}>();
</script>

<style scoped>
.config-title {
  margin: 0 0 0.75rem;
}
</style>
