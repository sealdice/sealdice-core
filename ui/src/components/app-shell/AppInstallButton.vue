<script setup lang="ts">
import { computed } from 'vue';
import { useMessage } from 'naive-ui';
import { usePwaInstall } from '@/features/pwa/usePwaInstall';

const message = useMessage();
const { canInstall, isInstalled, installing, install } = usePwaInstall();

const buttonText = computed(() => {
  if (isInstalled.value) return '已安装';
  if (installing.value) return '正在安装';
  return '安装应用';
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
  message.warning('当前浏览器不支持一键安装，请使用浏览器菜单安装');
}
</script>

<template>
  <n-tooltip>
    <template #trigger>
      <span class="install-trigger">
        <n-button
          v-if="!isInstalled"
          class="install-button"
          secondary
          :type="canInstall ? 'primary' : 'default'"
          :loading="installing"
          :disabled="installing"
          @click="handleClick"
        >
          <template #icon>
            <n-icon size="1.05rem">
              <i-carbon-download />
            </n-icon>
          </template>
          <span class="install-label">{{ buttonText }}</span>
        </n-button>
        <n-tag
          v-else
          :bordered="false"
          class="install-tag"
          size="small"
          type="success"
        >
          已安装
        </n-tag>
      </span>
    </template>
    {{ isInstalled ? '已安装到当前设备' : canInstall ? '安装到桌面 / 启动器' : '浏览器菜单也可以安装' }}
  </n-tooltip>
</template>

<style scoped>
.install-trigger {
  display: inline-flex;
  align-items: center;
}

.install-button {
  white-space: nowrap;
}

.install-tag {
  white-space: nowrap;
}

@media screen and (max-width: 639.9px) {
  .install-label {
    display: none;
  }
}
</style>
