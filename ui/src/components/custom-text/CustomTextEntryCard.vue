<script setup lang="ts">
import type { TextItemCompatibleInfo, Value } from '@/api';
import CustomTextPreviewInfo from './CustomTextPreviewInfo.vue';
import type { TextTemplateItem } from '@/features/customText/types';

const items = defineModel<TextTemplateItem[]>({ required: true });

const props = defineProps<{
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

<template>
  <n-form label-width="auto" label-position="top">
    <n-form-item class="w-full">
      <template #label>
        <div>
          <n-tag
            type="default"
            size="small"
            class="entry-tag"
            :bordered="false"
          >
            {{ help?.subType || (help?.notBuiltin ? '旧版文本' : '其它') }}
          </n-tag>

          <span>
            <span>{{ keyName }}</span>
            <n-tooltip v-if="help?.extraText">
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
              </template>
              {{ help.extraText }}
            </n-tooltip>
          </span>

          <template v-if="help?.notBuiltin">
            <n-tooltip placement="bottom-end">
              <template #trigger>
                <n-icon
                  class="entry-action-icon"
                  @click="emit('deleteKey', category, keyName)"
                >
                  <i-carbon-row-delete />
                </n-icon>
              </template>
              移除 - 这个文本在新版的默认配置中不被使用，<br />
              但升级而来时仍可能被使用，请确认无用后删除
            </n-tooltip>
          </template>

          <template v-if="help?.modified">
            <n-tooltip placement="bottom-end">
              <template #trigger>
                <n-icon
                  class="entry-action-icon"
                  @click="emit('resetKey', category, keyName)"
                >
                  <i-carbon-paint-brush />
                </n-icon>
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

        <div
          v-for="(item, index) in items"
          :key="textItemKeyOf(keyName, item)"
          class="entry-item"
        >
          <n-flex align="center">
            <div>
              <n-tooltip placement="bottom-start">
                <template #trigger>
                  <n-icon>
                    <i-carbon-add-filled v-if="index === 0" @click="emit('addItem', keyName)" />
                    <i-carbon-close-outline v-else @click="emit('removeItem', items, index)" />
                  </n-icon>
                </template>
                {{
                  index === 0
                    ? '点击添加一个回复语，SealDice 将会随机抽取一个回复'
                    : '点击删除你不想要的回复语'
                }}
              </n-tooltip>
            </div>
            <div class="relative flex-auto">
              <n-input
                v-model:value="item[0]"
                class="w-full"
                type="textarea"
                :autosize="{ minRows: 3 }"
                @update:value="emit('change', category, keyName)"
              />

              <div v-if="getPreview(keyName, item[0])" class="absolute bottom-0 right-1">
                <n-popover placement="bottom-start">
                  <template #trigger>
                    <span
                      v-if="getPreviewCheckErr(keyName, item[0])"
                      class="text-red-500 preview-icon"
                    >
                      <n-icon><i-carbon-close-filled /></n-icon>
                    </span>
                    <n-flex v-else>
                      <span
                        v-if="getPreview(keyName, item[0])?.version === 'v2'"
                        class="text-blue-500 preview-icon"
                      >
                        <n-icon><i-carbon-checkmark-filled /></n-icon>
                      </span>
                      <span
                        v-if="getPreview(keyName, item[0])?.version === 'v1'"
                        class="text-yellow-500 preview-icon"
                      >
                        <n-icon><i-carbon-checkmark-filled /></n-icon>
                      </span>
                    </n-flex>
                  </template>

                  <CustomTextPreviewInfo v-if="getPreview(keyName, item[0])" :info="getPreview(keyName, item[0])!" />
                </n-popover>
              </div>
            </div>
          </n-flex>
        </div>
        <n-flex size="small" wrap>
          <n-tag
            v-for="item in help?.vars ?? []"
            :key="item"
            size="small"
            type="info"
            :bordered="false"
          >
            {{ item }}
          </n-tag>
        </n-flex>
      </n-flex>
    </n-form-item>
  </n-form>
</template>

<style scoped>
.entry-tag {
  margin-right: 0.5rem;
}

.entry-action-icon {
  float: right;
  margin-left: 1rem;
}

.entry-item {
  width: 100%;
  margin-bottom: 0.5rem;
}

.preview-icon {
  margin-left: 0.1rem;
  margin-top: 0.1rem;
}
</style>
