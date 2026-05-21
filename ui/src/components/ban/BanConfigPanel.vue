<script setup lang="ts">
import type { BanConfig } from '@/api';
import TipBox from '@/components/shared/TipBox.vue';

const config = defineModel<BanConfig>('config', { required: true });

defineProps<{
  dirty: boolean;
  saving: boolean;
}>();

const emit = defineEmits<{
  save: [];
}>();
</script>

<template>
  <section class="ban-config-panel">
    <header class="ban-config-panel__header">
      <n-button type="primary" :loading="saving" @click="emit('save')">
        <template #icon>
          <n-icon><i-carbon-save /></n-icon>
        </template>
        保存设置
      </n-button>
    </header>

    <TipBox v-if="dirty" type="error" class="ban-config-panel__warning">
      <n-text type="error" tag="strong">内容已修改，不要忘记保存。</n-text>
    </TipBox>

    <section class="ban-config-panel__section">
      <h3>基本设置</h3>
      <n-flex wrap>
        <n-checkbox v-model:checked="config.banBehaviorRefuseReply">拒绝回复</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorRefuseInvite">拒绝邀请</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitLastPlace">退出事发群</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitPlaceImmediately">使用时立即退出群</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitIfAdmin">
          使用者为管理员立即退群，为普通群员进行通告
        </n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitIfAdminSilentIfNotAdmin">
          使用者为管理员立即退群，为普通群员仅拒绝回复
        </n-checkbox>
      </n-flex>
    </section>

    <section class="ban-config-panel__section">
      <h3>怒气值设置</h3>
      <TipBox type="info" class="ban-config-panel__tip">
        <n-text type="info">
          海豹的黑名单使用积分制。用户做出恶意行为时怒气值上涨，达到阈值后进入警告或黑名单。
        </n-text>
      </TipBox>

      <n-form size="small" label-placement="left" label-width="112" class="ban-config-panel__form">
        <n-form-item label="警告阈值">
          <n-input-number v-model:value="config.thresholdWarn" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="拉黑阈值">
          <n-input-number v-model:value="config.thresholdBan" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="自动拉黑时长(分钟)">
          <n-input-number v-model:value="config.autoBanMinutes" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="禁言增加">
          <n-input-number v-model:value="config.scoreGroupMuted" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="踢出增加">
          <n-input-number v-model:value="config.scoreGroupKicked" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="刷屏增加">
          <n-input-number v-model:value="config.scoreTooManyCommand" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="每分钟下降">
          <n-input-number v-model:value="config.scoreReducePerMinute" :min="0" :step="1" :precision="0" />
        </n-form-item>
        <n-form-item label="群组连带责任">
          <n-input-number v-model:value="config.jointScorePercentOfGroup" :min="0" :max="1" :step="0.1" />
        </n-form-item>
        <n-form-item label="邀请人连带责任">
          <n-input-number v-model:value="config.jointScorePercentOfInviter" :min="0" :max="1" :step="0.1" />
        </n-form-item>
      </n-form>
    </section>
  </section>
</template>

<style scoped>
.ban-config-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.ban-config-panel__section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.ban-config-panel__section h3 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 1rem;
}

.ban-config-panel__form {
  max-width: 32rem;
}
</style>
