<script setup lang="ts">
import { computed } from 'vue';
import {
  DEFAULT_THEME_PALETTE,
  useAppTheme,
  type ThemeColorKey,
} from '@/features/theme';

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();

const visible = computed({
  get: () => props.show,
  set: value => emit('update:show', value),
});

const { resetThemePalette, setThemeColor, themePalette } = useAppTheme();

// 主题弹窗只负责浏览器本地的视觉偏好，不写入后端，也不替换任何业务设置页内容。
const colorFields: Array<{
  key: ThemeColorKey;
  label: string;
  description: string;
}> = [
  { key: 'primary', label: '主色', description: '按钮、选中态、重点操作' },
  { key: 'info', label: '信息色', description: '普通提示与信息标签' },
  { key: 'success', label: '成功色', description: '成功反馈与完成状态' },
  { key: 'warning', label: '警告色', description: '风险提示与需关注状态' },
  { key: 'error', label: '错误色', description: '错误反馈与危险操作' },
];

const swatches = Object.values(DEFAULT_THEME_PALETTE);

function updateColor(key: ThemeColorKey, value: string | null) {
  if (!value) return;
  setThemeColor(key, value);
}
</script>

<template>
  <n-modal
    v-model:show="visible"
    class="theme-palette-modal"
    preset="card"
    title="主题设置"
    :auto-focus="false"
  >
    <p class="theme-palette-intro">
      颜色会立即应用到 Naive UI 组件和项目语义色，并保存在当前浏览器。
    </p>

    <div class="theme-color-list">
      <label
        v-for="field in colorFields"
        :key="field.key"
        class="theme-color-row"
      >
        <span class="theme-color-copy">
          <span class="theme-color-label">{{ field.label }}</span>
          <span class="theme-color-description">{{ field.description }}</span>
        </span>
        <n-color-picker
          :value="themePalette[field.key]"
          :modes="['hex']"
          :show-alpha="false"
          :show-preview="false"
          :swatches="swatches"
          @update:value="updateColor(field.key, $event)"
        />
      </label>
    </div>

    <template #footer>
      <n-flex justify="space-between">
        <n-button secondary @click="resetThemePalette">
          恢复默认
        </n-button>
        <n-button type="primary" @click="visible = false">
          完成
        </n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<style scoped>
:global(.theme-palette-modal .n-card) {
  width: min(560px, calc(100vw - 32px));
  max-width: calc(100vw - 32px);
}

.theme-palette-intro {
  margin: 0 0 1rem;
  color: var(--sd-text-muted);
  line-height: 1.7;
}

.theme-color-list {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.theme-color-row {
  display: grid;
  align-items: center;
  gap: 1rem;
  grid-template-columns: minmax(8rem, 1fr) minmax(0, 260px);
}

.theme-color-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.15rem;
}

.theme-color-label {
  color: var(--sd-text-primary);
  font-weight: 600;
  line-height: 1.3;
}

.theme-color-description {
  color: var(--sd-text-muted);
  font-size: 0.82rem;
  line-height: 1.4;
}

@media screen and (max-width: 639.9px) {
  .theme-color-row {
    grid-template-columns: 1fr;
  }
}
</style>
