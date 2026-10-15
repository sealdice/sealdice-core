<template>
  <main class="custom-text-page sd-page-flow">
    <n-spin :show="editor.customTextQuery.isFetching.value && !editor.customTextQuery.data.value">
      <PageHeader :title="category" />
      <HelpCollapse>
        <div class="custom-text-help">
          <div>此处可以对骰子返回的文本进行修改。最终返回的文本将为多个条目中随机抽取的一个。</div>
          <div>
            随机文本：默认一种显示结果，如果需要多种反馈结果，使用＋添加条目，使用 - 删除条目
          </div>
          <div>
            遇到有此标记 (<n-icon><i-tabler-brush /></n-icon>)
            的条目，说明和默认值不同，是一个自定义条目
          </div>
          <div class="help-block">
            文本下方的
            <n-tag size="small" :bordered="false">标签</n-tag>
            代表了被默认文本所使用的特殊变量，你可以使用
            <span class="sd-code-text">{变量名}</span> 来插入他们，例如
            <span class="sd-code-text">{$t判定值}</span>
          </div>
          <div>
            除此之外，这些变量可以在所有文本中使用：
            <n-flex size="small" wrap>
              <n-tag
                v-for="item in commonVars"
                :key="item"
                size="small"
                :bordered="false"
                class="sd-code-text"
              >
                {{ item }}
              </n-tag>
            </n-flex>
          </div>
          <div>
            <span>以及，所有的自定义文本都可以嵌套使用，例如：</span>
            <div>
              <b>这里是<span class="sd-code-text">{核心：骰子名字}</span>，我是一个示例</b>
            </div>
            <div>默认会被解析为：</div>
            <div>
              <b>这里是海豹 bot，我是一个示例</b>
            </div>
            <div>注意！千万不要递归嵌套，会发生很糟糕的事情。</div>
          </div>

          <div class="help-block">
            <div>此外，支持插入图片，将图片放在骰子的适当目录，再写这样一句话即可：</div>
            <div><n-text code class="sd-code-text">[图:data/images/sealdice.png]</n-text></div>
            <div>可以参考 核心：骰子进群 词条</div>
            <div>同样的，可以使用 CQ 码插入图片和其他内容，关于 CQ 码，请参阅 onebot 项目文档</div>
          </div>

          <div class="help-block">
            <b>
              COC 的「判定 - 常规」和「判定 - 简短」主要区别是，多重检定会默认使用简短版本
              <span class="sd-code-text">(.ra 3#射击)</span>
            </b>
            <b>进行调整后，可以在左侧面板「指令测试」中进行测试！</b>
          </div>
        </div>
      </HelpCollapse>

      <div class="custom-text-toolbar">
        <n-flex align="center" size="small" class="custom-text-search">
          <n-text>搜索：</n-text>
          <span class="custom-text-search-input">
            <n-input v-model:value="editor.currentFilterName.value" size="small" clearable>
              <template #prefix>
                <n-icon><i-tabler-search /></n-icon>
              </template>
            </n-input>
          </span>
        </n-flex>
        <n-flex align="center" size="small" class="custom-text-actions">
          <n-button
            secondary
            :loading="editor.previewRefreshMutation.isPending.value"
            @click="editor.refreshPreview()"
          >
            刷新预览
          </n-button>
          <n-button secondary @click="editor.dialogImportVisible.value = true">导入/导出</n-button>
        </n-flex>
      </div>

      <n-flex class="custom-text-filter-row" align="center" wrap>
        <n-radio-group
          v-model:value="editor.filterMode.value"
          @update:value="editor.handleFilterModeChange($event as CustomTextFilterMode)"
        >
          <n-radio
            v-for="item of filterModes"
            :key="item.value"
            :value="item.value"
            :label="item.desc"
          />
        </n-radio-group>
        <n-flex
          v-if="editor.filterMode.value === 'group'"
          align="center"
          class="custom-text-group-filter"
        >
          <n-text>分组：</n-text>
          <span class="custom-text-group-select">
            <n-select
              v-model:value="editor.currentFilterGroup.value"
              filterable
              tag
              :options="editor.filterGroups.value.map(item => ({ label: item, value: item }))"
            />
          </span>
        </n-flex>
      </n-flex>

      <n-empty v-if="!editor.hasCategory.value" description="未找到当前文案分类" />

      <n-empty v-else-if="editor.sortedCategory.value.length === 0" description="没有匹配的文案">
        <template #extra>
          <n-button size="small" @click="editor.clearFilters">清除筛选</n-button>
        </template>
      </n-empty>

      <n-collapse v-else class="text-collapse" :default-expanded-names="['__others__']">
        <n-collapse-item
          v-for="[group, values] in editor.sortedCategory.value"
          :key="group"
          :title="group === '__others__' ? '无分组' : group"
          :name="group"
        >
          <n-grid :cols="1" :x-gap="20">
            <n-grid-item>
              <div class="entry-columns">
                <div v-for="[keyName] in values" :key="keyName" class="entry-column-item">
                  <CustomTextEntryCard
                    :items="editor.texts.value[editor.category.value][keyName]"
                    :category="editor.category.value"
                    :key-name="keyName"
                    :help="editor.helpInfo.value[editor.category.value]?.[keyName]"
                    :previews="editor.previewInfo.value[`${editor.category.value}:${keyName}`]"
                    @add-item="editor.addItem"
                    @remove-item="editor.removeItem"
                    @update-item="editor.updateItem"
                    @delete-key="editor.askDeleteValue"
                    @reset-key="editor.askResetValue"
                  />
                </div>
              </div>
            </n-grid-item>
          </n-grid>
        </n-collapse-item>
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
import HelpCollapse from '@/components/shared/HelpCollapse.vue';
import type { CustomTextFilterMode } from '@/features/customText/viewModel';
import CustomTextEntryCard from '@/components/custom-text/CustomTextEntryCard.vue';
import CustomTextImportModal from '@/components/custom-text/CustomTextImportModal.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import { useCustomTextEditor } from '@/features/customText/useCustomTextEditor';

import { computed } from 'vue';
import { useRoute } from 'vue-router';

const props = defineProps<{ category?: string }>();
const route = useRoute();
const category = computed(() => {
  const routeParams = route.params as Record<string, string | string[] | undefined>;
  const routeCategory = routeParams.category;
  const fallback = Array.isArray(routeCategory) ? routeCategory[0] : routeCategory;
  return props.category ?? String(fallback ?? '');
});

const editor = useCustomTextEditor(category);

const commonVars = [
  '$t玩家',
  '$tQQ昵称',
  '$t个人骰子面数',
  '$tQQ',
  '$t骰子帐号',
  '$t骰子昵称',
  '$t群号',
  '$t群名',
];
const filterModes = [
  { value: 'all', desc: '全部' },
  { value: 'unmodified', desc: '默认文案' },
  { value: 'modified', desc: '修改过' },
  { value: 'group', desc: '指定分组' },
  { value: 'deprecated', desc: '旧版文本' },
] satisfies Array<{ value: CustomTextFilterMode; desc: string }>;
</script>

<style scoped>
.custom-text-page {
  text-align: left;
}

.custom-text-page > :deep(.n-spin-container > .n-spin-content) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--sd-page-section-gap);
}

.text-collapse {
  width: 100%;
}

.entry-columns {
  column-count: 1;
  column-gap: var(--sd-space-xl);
}

.entry-column-item {
  display: inline-block;
  width: 100%;
  margin-bottom: var(--sd-space-md);
  break-inside: avoid;
}

@media (min-width: 1024px) {
  .entry-columns {
    column-count: 2;
  }
}
.custom-text-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--sd-space-md);
}

@media screen and (max-width: 767.9px) {
  .custom-text-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .custom-text-search,
  .custom-text-actions,
  .custom-text-search-input {
    width: 100%;
  }

  .custom-text-actions {
    justify-content: flex-start;
  }

  .custom-text-search-input :deep(.n-input) {
    width: 100%;
  }
}
@media screen and (max-width: 767.9px) {
  .custom-text-filter-row,
  .custom-text-group-filter,
  .custom-text-group-select {
    width: 100%;
  }

  .custom-text-group-select :deep(.n-select) {
    width: 100%;
  }
}
.help-block {
  margin-top: 1rem;
}
</style>
