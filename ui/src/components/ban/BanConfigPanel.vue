<template>
  <section class="ban-config-panel">
    <header class="ban-config-panel__header">
      <n-button type="primary" :loading="saving" @click="emit('save')">
        <template #icon>
          <n-icon><i-tabler-device-floppy /></n-icon>
        </template>
        保存设置
      </n-button>
    </header>

    <TipBox v-if="dirty" type="warning" class="ban-config-panel__warning">
      <n-text type="warning" tag="strong">内容已修改，不要忘记保存。</n-text>
    </TipBox>

    <SettingCategoryBox title="基本设置" padded>
      <n-flex wrap class="ban-config-panel__checks">
        <n-checkbox v-model:checked="config.banBehaviorRefuseReply">拒绝回复</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorRefuseInvite">拒绝邀请</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitLastPlace">退出事发群</n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitPlaceImmediately"
          >使用时立即退出群</n-checkbox
        >
        <n-checkbox v-model:checked="config.banBehaviorQuitIfAdmin">
          使用者为管理员立即退群，为普通群员进行通告
        </n-checkbox>
        <n-checkbox v-model:checked="config.banBehaviorQuitIfAdminSilentIfNotAdmin">
          使用者为管理员立即退群，为普通群员仅拒绝回复
        </n-checkbox>
      </n-flex>
    </SettingCategoryBox>

    <SettingCategoryBox title="怒气值设置" padded>
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
          <n-input-number
            v-model:value="config.scoreGroupMuted"
            :min="0"
            :step="1"
            :precision="0"
          />
        </n-form-item>
        <n-form-item label="踢出增加">
          <n-input-number
            v-model:value="config.scoreGroupKicked"
            :min="0"
            :step="1"
            :precision="0"
          />
        </n-form-item>
        <n-form-item label="刷屏增加">
          <n-input-number
            v-model:value="config.scoreTooManyCommand"
            :min="0"
            :step="1"
            :precision="0"
          />
        </n-form-item>
        <n-form-item label="黑名单通报间隔">
          <n-flex align="center" wrap>
            <n-input-number
              v-model:value="config.banNotifyIntervalMinutes"
              :min="-1"
              :step="1"
              :precision="0"
            />
            <n-tooltip trigger="hover">
              <template #trigger>
                <n-text depth="3">分钟</n-text>
              </template>
              -1 表示每次通报；0 表示使用默认的 20 分钟；正整数表示自定义冷却分钟数。
            </n-tooltip>
          </n-flex>
        </n-form-item>
        <n-form-item label="每分钟下降">
          <n-input-number
            v-model:value="config.scoreReducePerMinute"
            :min="0"
            :step="1"
            :precision="0"
          />
        </n-form-item>
        <n-form-item label="群组连带责任">
          <n-input-number
            v-model:value="config.jointScorePercentOfGroup"
            :min="0"
            :max="1"
            :step="0.1"
          />
        </n-form-item>
        <n-form-item label="邀请人连带责任">
          <n-input-number
            v-model:value="config.jointScorePercentOfInviter"
            :min="0"
            :max="1"
            :step="0.1"
          />
        </n-form-item>
      </n-form>
    </SettingCategoryBox>
  </section>
</template>

<script setup lang="ts">
import type { BanConfig } from '@/api';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
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

<style scoped>
.ban-config-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.ban-config-panel__checks {
  align-items: flex-start;
}

.ban-config-panel__form {
  max-width: 32rem;
}
</style>
