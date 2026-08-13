<template>
  <!-- 不支持安装且未安装时不占位：侧边栏页脚不放一个永远点不动的按钮。 -->
  <n-tooltip v-if="isSupported || isInstalled" :placement="props.collapsed ? 'right' : 'top'">
    <template #trigger>
      <n-button
        quaternary
        class="sd-sidebar-footer-action install-button"
        :loading="installing"
        :disabled="installing || isInstalled"
        :aria-label="buttonText"
        @click="handleClick"
      >
        <template #icon>
          <n-icon>
            <i-tabler-circle-check v-if="isInstalled" />
            <i-tabler-download v-else />
          </n-icon>
        </template>
        <span v-if="!props.collapsed" class="install-label">{{ buttonText }}</span>
      </n-button>
    </template>
    {{ tooltipText }}
  </n-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useMessage } from 'naive-ui';
import { usePwaInstall } from '@/features/pwa/usePwaInstall';

const props = withDefaults(defineProps<{ collapsed?: boolean }>(), { collapsed: false });

const message = useMessage();
const { isSupported, canInstall, isInstalled, installing, install } = usePwaInstall();

const buttonText = computed(() => {
  if (isInstalled.value) return '已安装 PWA';
  if (installing.value) return '正在安装';
  return '安装 PWA';
});

const tooltipText = computed(() => {
  if (isInstalled.value) return '已安装到当前设备';
  if (canInstall.value) return '安装到桌面或启动器';
  return '当前浏览器需从其菜单中安装';
});

async function handleClick() {
  const outcome = await install();
  if (outcome === 'installed') {
    message.success('已安装到设备');
    return;
  }
  if (outcome === 'dismissed') {
    message.info('已取消安装');
    return;
  }
  message.warning('当前浏览器不支持安装应用');
}
</script>

<style scoped>
.install-label {
  white-space: nowrap;
}

/* 已安装是状态而非可用操作，禁用后仍需保持页脚文字的可读性。 */
.install-button:disabled {
  --n-text-color-disabled: var(--sd-text-inverse-soft) !important;

  opacity: 0.75;
}
</style>
