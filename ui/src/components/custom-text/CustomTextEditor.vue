<template>
  <main class="custom-text-page">
    <n-spin :show="editor.customTextQuery.isFetching.value && !editor.customTextQuery.data.value">
      <PageHeader :title="category" description="编辑当前分类的自定义回复文案。" />
      <CustomTextHelp />

      <CustomTextToolbar
        v-model:keyword="editor.currentFilterName.value"
        :preview-loading="editor.previewRefreshMutation.isPending.value"
        @refresh-preview="editor.refreshPreview"
        @open-import="editor.dialogImportVisible.value = true"
      />

      <CustomTextFilterBar
        v-model:mode="editor.filterMode.value"
        v-model:group="editor.currentFilterGroup.value"
        :groups="editor.filterGroups.value"
        @mode-change="editor.handleFilterModeChange"
      />

      <n-empty v-if="!editor.hasCategory.value" description="未找到当前文案分类" />

      <n-empty v-else-if="editor.sortedCategory.value.length === 0" description="没有匹配的文案">
        <template #extra>
          <n-button size="small" @click="editor.clearFilters">清除筛选</n-button>
        </template>
      </n-empty>

      <n-collapse v-else class="text-collapse" :default-expanded-names="['__others__']">
        <CustomTextBox
          v-for="[group, values] in editor.sortedCategory.value"
          :key="group"
          :group="group"
        >
          <template #values>
            <n-grid x-gap="24" y-gap="16" cols="1 m:2" responsive="screen">
              <n-grid-item
                v-for="[keyName] in values"
                :key="keyName"
                :span="isMultiValue(keyName) ? '1 m:2' : 1"
              >
                <CustomTextEntryCard
                  v-model="editor.texts.value[editor.category.value][keyName]"
                  :category="editor.category.value"
                  :key-name="keyName"
                  :help="editor.helpInfo.value[editor.category.value]?.[keyName]"
                  :get-preview="editor.getPreview"
                  :get-preview-check-err="editor.getPreviewCheckErr"
                  :text-item-key-of="editor.textItemKeyOf"
                  @add-item="editor.addItem"
                  @remove-item="editor.removeItem"
                  @change="editor.doChanged"
                  @delete-key="editor.askDeleteValue"
                  @reset-key="editor.askResetValue"
                />
              </n-grid-item>
            </n-grid>
          </template>
        </CustomTextBox>
      </n-collapse>

      <CustomTextImportModal
        v-model:show="editor.dialogImportVisible.value"
        v-model:content="editor.configForImport.value"
        v-model:only-current="editor.importOnlyCurrent.value"
        v-model:compact="editor.importImpact.value"
        :saving="editor.saveMutation.isPending.value"
        @copy="editor.copied"
        @clear="editor.configForImport.value = ''"
        @import="editor.doImport"
      />
    </n-spin>
  </main>
</template>

<script setup lang="ts">
import CustomTextBox from './CustomTextBox.vue';
import CustomTextEntryCard from './CustomTextEntryCard.vue';
import CustomTextFilterBar from './CustomTextFilterBar.vue';
import CustomTextHelp from './CustomTextHelp.vue';
import CustomTextImportModal from './CustomTextImportModal.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import CustomTextToolbar from './CustomTextToolbar.vue';
import { useCustomTextEditor } from '@/features/customText/useCustomTextEditor';

const props = defineProps<{
  category: string;
}>();

const editor = useCustomTextEditor(() => props.category);

function isMultiValue(keyName: string): boolean {
  return (editor.texts.value[editor.category.value]?.[keyName]?.length ?? 0) > 1;
}
</script>

<style scoped>
.custom-text-page {
  text-align: left;
}

.text-collapse {
  width: 100%;
}
</style>
