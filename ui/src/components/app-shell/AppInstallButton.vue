<template>
  <n-tooltip :placement="props.collapsed ? 'right' : 'top'">
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

  <PwaInstallGuideModal v-model:show="guideVisible" :guide="installGuide" />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useMessage } from 'naive-ui';
import { buildPwaInstallGuide, getPwaInstallEnvironment } from '@/features/pwa/pwaInstallGuide';
import { usePwaInstall } from '@/features/pwa/usePwaInstall';
import PwaInstallGuideModal from './PwaInstallGuideModal.vue';

const props = withDefaults(defineProps<{ collapsed?: boolean }>(), { collapsed: false });

const message = useMessage();
const { canInstall, isInstalled, installing, install } = usePwaInstall();
const guideVisible = ref(false);
const installGuide = buildPwaInstallGuide(getPwaInstallEnvironment());

const buttonText = computed(() => {
  if (isInstalled.value) return '已作为应用运行';
  if (installing.value) return '正在安装';
  return '安装为应用';
});

const tooltipText = computed(() => {
  if (isInstalled.value) return '当前正在独立应用窗口中运行';
  if (canInstall.value) return '安装到桌面或启动器';
  return '查看当前浏览器的安装方式';
});

async function handleClick() {
  if (!canInstall.value) {
    guideVisible.value = true;
    return;
  }

  const outcome = await install();
  if (outcome === 'installed') {
    message.success('已安装到设备');
    return;
  }
  if (outcome === 'dismissed') {
    message.info('已取消安装');
    return;
  }
  guideVisible.value = true;
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
