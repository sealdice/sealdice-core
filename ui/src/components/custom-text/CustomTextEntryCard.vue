<template>
  <n-form label-width="auto" label-position="top">
    <n-form-item class="w-full">
      <template #label>
        <div>
          <n-tag type="default" size="small" class="entry-tag" :bordered="false">
            {{ help?.subType || (help?.notBuiltin ? '旧版文本' : '其它') }}
          </n-tag>

          <span>
            <span>{{ keyName }}</span>
            <n-tooltip v-if="help?.extraText">
              <template #trigger>
                <n-icon><i-tabler-help-circle /></n-icon>
              </template>
              {{ help.extraText }}
            </n-tooltip>
          </span>

          <template v-if="help?.notBuiltin">
            <n-tooltip placement="bottom-end">
              <template #trigger>
                <n-button
                  class="entry-action-button"
                  quaternary
                  circle
                  size="tiny"
                  type="error"
                  aria-label="删除旧版文案"
                  @click="emit('deleteKey', category, keyName)"
                >
                  <template #icon><i-tabler-trash /></template>
                </n-button>
              </template>
              移除 - 这个文本在新版的默认配置中不被使用，<br />
              但升级而来时仍可能被使用，请确认无用后删除
            </n-tooltip>
          </template>

          <template v-if="help?.modified">
            <n-tooltip placement="bottom-end">
              <template #trigger>
                <n-button
                  class="entry-action-button"
                  quaternary
                  circle
                  size="tiny"
                  aria-label="重置文案为初始值"
                  @click="emit('resetKey', category, keyName)"
                >
                  <template #icon><i-tabler-restore /></template>
                </n-button>
              </template>
              重置为初始值
            </n-tooltip>
          </template>
        </div>
      </template>

      <n-flex vertical class="w-full">
        <n-text v-if="keyName === '戳一戳'" type="warning" class="mb-1 text-xs">
          请确认你使用的 QQ
          连接方式支持该功能，若不支持请于「基本设置」中关闭戳一戳来避免日志中出现相关报错。
        </n-text>

        <RepeatableList add-label="添加随机文案" @add="emit('addItem', keyName)">
          <RepeatableItem
            v-for="(item, index) in visibleItems"
            :key="textItemKeyOf(keyName, item)"
            :title="`文案 ${index + 1}`"
            :removable="items.length > 1"
            remove-label="删除随机文案"
            @remove="emit('removeItem', items, index)"
          >
            <div class="relative">
              <n-input
                v-model:value="item[0]"
                class="w-full sd-code-text"
                type="textarea"
                :autosize="{ minRows: 1, maxRows: 4 }"
                @update:value="emit('change', category, keyName)"
              />

              <div v-if="getPreview(keyName, item[0])" class="absolute bottom-0 right-1">
                <n-popover placement="bottom-start">
                  <template #trigger>
                    <span
                      v-if="getPreviewCheckErr(keyName, item[0])"
                      class="text-red-500 preview-icon"
                    >
                      <n-icon><i-tabler-circle-x-filled /></n-icon>
                    </span>
                    <n-flex v-else>
                      <span
                        v-if="getPreview(keyName, item[0])?.version === 'v2'"
                        class="text-blue-500 preview-icon"
                      >
                        <n-icon><i-tabler-circle-check-filled /></n-icon>
                      </span>
                      <span
                        v-if="getPreview(keyName, item[0])?.version === 'v1'"
                        class="text-yellow-500 preview-icon"
                      >
                        <n-icon><i-tabler-circle-check-filled /></n-icon>
                      </span>
                    </n-flex>
                  </template>

                  <CustomTextPreviewInfo
                    v-if="getPreview(keyName, item[0])"
                    :info="getPreview(keyName, item[0])!"
                  />
                </n-popover>
              </div>
            </div>
          </RepeatableItem>
        </RepeatableList>
        <n-button
          v-if="isLongList"
          text
          size="small"
          class="entry-expand-button"
          :aria-expanded="!collapsed"
          @click="collapsed = !collapsed"
        >
          {{ collapsed ? `展开其余 ${items.length - visibleItems.length} 条` : '收起多余条目' }}
        </n-button>
        <n-flex size="small" wrap>
          <n-tag v-for="item in help?.vars ?? []" :key="item" size="small" :bordered="false">
            {{ item }}
          </n-tag>
        </n-flex>
      </n-flex>
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { TextItemCompatibleInfo, Value } from '@/api';
import CustomTextPreviewInfo from './CustomTextPreviewInfo.vue';
import RepeatableItem from '@/components/shared/RepeatableItem.vue';
import RepeatableList from '@/components/shared/RepeatableList.vue';
import type { TextTemplateItem } from '@/features/customText/types';

const items = defineModel<TextTemplateItem[]>({ required: true });
const collapsed = ref(true);
const isLongList = computed(() => items.value.length > 3);
const visibleItems = computed(() =>
  isLongList.value && collapsed.value ? items.value.slice(0, 3) : items.value
);

defineProps<{
  category: string;
  keyName: string;
  help?: Value;
  getPreview: (keyName: string, text: string) => TextItemCompatibleInfo | undefined;
  getPreviewCheckErr: (keyName: string, text: string) => boolean;
  textItemKeyOf: (keyName: string, item: TextTemplateItem) => string;
}>();

const emit = defineEmits<{
  addItem: [keyName: string];
  removeItem: [items: TextTemplateItem[], index: number];
  change: [category: string, keyName: string];
  deleteKey: [category: string, keyName: string];
  resetKey: [category: string, keyName: string];
}>();
</script>

<style scoped>
.entry-tag {
  margin-right: 0.5rem;
}

.entry-action-button {
  float: right;
  margin-left: 1rem;
}

.entry-expand-button {
  align-self: flex-start;
}

.preview-icon {
  margin-left: 0.1rem;
  margin-top: 0.1rem;
}
</style>
