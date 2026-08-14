<template>
  <n-modal
    :show="showDialog"
    preset="dialog"
    :closable="false"
    :mask-closable="false"
    :close-on-esc="false"
    title="输入密码解锁"
    positive-text="确认"
    :positive-button-props="{ loading: authSession.signinMutation.isPending.value }"
    @positive-click="doUnlock"
    @keyup.enter="doUnlock"
  >
    <template #icon>
      <n-icon>
        <i-tabler-lock />
      </n-icon>
    </template>

    <n-input
      v-model:value="password"
      type="password"
      show-password-on="mousedown"
      placeholder="请输入 UI 密码"
      :disabled="authSession.signinMutation.isPending.value"
    />
    <n-text v-if="errorText" class="unlock-error" type="error">
      {{ errorText }}
    </n-text>
  </n-modal>

  <n-modal
    v-model:show="dialogCheckPassword"
    :closable="false"
    preset="dialog"
    title="欢迎使用海豹核心"
    :mask-closable="false"
    transform-origin="center"
  >
    <n-text>
      如果您的服务开启在公网，为了保证您的安全性，请前往
      <strong>「综合设置」>「基本设置」</strong> 界面，设置
      <strong>UI 界面密码</strong>。或切换为只有本机可访问。<br />
    </n-text>
    <n-text type="warning" class="security-warning">
      如果您不了解上面在说什么，请务必设置一个密码！
    </n-text>

    <template #action>
      <n-button type="primary" @click="dialogCheckPassword = false"> 我已知晓！ </n-button>
    </template>
  </n-modal>
</template>

<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { useMessage } from 'naive-ui';
import { getSdApiV2BaseSecurityCheck } from '@/api';
import { getErrorMessage } from '@/features/auth/error';
import { useAuthSession } from '@/features/auth/useAuthSession';

const authSession = useAuthSession();
const message = useMessage();
const password = ref('');
const dialogCheckPassword = ref(false);
const canSkipSecurityDialog = ref(false);
const hasCheckedSecurity = ref(false);

const showDialog = computed(() => authSession.needsUnlock.value);
const canCheckSecurity = computed(
  () => authSession.hasAccessToken.value && !authSession.isInitializing.value
);
const errorText = computed(() =>
  authSession.signinMutation.isError.value
    ? getErrorMessage(authSession.signinMutation.error.value, '密码错误')
    : ''
);

async function doUnlock() {
  try {
    await authSession.signin({ password: password.value });
    password.value = '';
    message.success('登录成功，欢迎回来');
    await checkPasswordSecurity();
  } catch {
    message.error(errorText.value || '密码错误');
    password.value = '';
  }
}

async function checkPasswordSecurity() {
  if (hasCheckedSecurity.value || !canCheckSecurity.value) return;
  hasCheckedSecurity.value = true;
  const result = await getSdApiV2BaseSecurityCheck({ throwOnError: true });
  if (!result.data.item) {
    dialogCheckPassword.value = true;
    canSkipSecurityDialog.value = false;
  }
}

watch(canCheckSecurity, canAccess => {
  if (canAccess) {
    void checkPasswordSecurity();
  } else if (!authSession.hasAccessToken.value) {
    hasCheckedSecurity.value = false;
  }
}, { immediate: true });
</script>

<style scoped>
.unlock-error {
  display: block;
  margin-top: 0.75rem;
}

.security-warning {
  display: block;
  margin-top: 1rem;
}
</style>
