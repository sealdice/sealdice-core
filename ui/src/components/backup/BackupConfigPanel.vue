<template>
  <section class="backup-config-panel">
    <n-form ref="formRef" :model="config" :rules="rules" label-placement="top" :disabled="disabled">
      <SettingCategoryBox
        title="自动备份"
        padded
        :columns="2"
        :show-panel="config.autoBackupEnable"
      >
        <template #title-extra>
          <n-switch v-model:value="config.autoBackupEnable" aria-label="启用自动备份" />
        </template>

        <n-form-item path="autoBackupTime">
          <template #label>
            <span class="backup-config-panel__label">
              备份间隔
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon class="backup-config-panel__help">
                    <i-tabler-help-circle />
                  </n-icon>
                </template>
                备份间隔表达式使用 robfig/cron 格式，例如 @every 12h。
              </n-tooltip>
            </span>
          </template>
          <n-input
            v-model:value="config.autoBackupTime"
            class="sd-code-text"
            placeholder="@every 12h"
          />
        </n-form-item>

        <n-form-item path="autoBackupSelectionList" label="备份范围">
          <BackupSelectionGroup v-model:value="config.autoBackupSelectionList" />
        </n-form-item>

        <n-form-item label="备份文件名预览">
          <n-text code class="backup-config-panel__preview">
            {{ autoBackupPreview }}
          </n-text>
        </n-form-item>
      </SettingCategoryBox>

      <SettingCategoryBox title="自动清理" padded :columns="2" :show-panel="cleaningEnabled">
        <template #title-extra>
          <n-switch v-model:value="cleaningEnabled" aria-label="启用自动清理" />
        </template>

        <n-form-item path="backupCleanStrategy" label="保留规则">
          <n-radio-group v-model:value="config.backupCleanStrategy" size="small">
            <n-radio-button :value="1">按数量保留</n-radio-button>
            <n-radio-button :value="2">按时间保留</n-radio-button>
          </n-radio-group>
        </n-form-item>

        <n-form-item
          v-if="config.backupCleanStrategy === 1"
          path="backupCleanKeepCount"
          label="保留数量"
        >
          <n-input-number v-model:value="config.backupCleanKeepCount" :min="1" :step="1" />
        </n-form-item>

        <n-form-item v-if="config.backupCleanStrategy === 2" path="backupCleanKeepDur">
          <template #label>
            <span class="backup-config-panel__label">
              保留时间
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon class="backup-config-panel__help">
                    <i-tabler-help-circle />
                  </n-icon>
                </template>
                支持 h、m、s，例如 720h 表示保留 30 天内的备份。
              </n-tooltip>
            </span>
          </template>
          <n-input v-model:value="config.backupCleanKeepDur" placeholder="720h" />
        </n-form-item>

        <n-form-item path="backupCleanTriggers">
          <template #label>
            <span class="backup-config-panel__label">
              触发方式
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon class="backup-config-panel__help">
                    <i-tabler-help-circle />
                  </n-icon>
                </template>
                自动备份后会在每次自动备份完成后顺便清理；定时会按照 cron 表达式单独清理。
              </n-tooltip>
            </span>
          </template>
          <n-checkbox-group v-model:value="config.backupCleanTriggers">
            <n-flex align="center" wrap>
              <n-checkbox
                v-for="option in cleanTriggerOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </n-checkbox>
            </n-flex>
          </n-checkbox-group>
        </n-form-item>

        <n-form-item path="backupCleanCron" label="定时间隔">
          <n-input
            v-model:value="config.backupCleanCron"
            class="sd-code-text"
            :disabled="!config.backupCleanTriggers.includes('cron')"
            placeholder="0 0 * * *"
          />
        </n-form-item>
      </SettingCategoryBox>

      <n-alert type="info" :bordered="false">
        恢复备份时，将骰子彻底关闭，解压备份压缩包到骰子目录。若提示是否覆盖，选择全部即可。
      </n-alert>
    </n-form>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { FormInst, FormRules } from 'naive-ui';
import type { BackupCleanTriggerKey, BackupConfigDraft } from '@/features/backup/viewModel';
import { buildBackupConfigPayload, buildBackupFilenamePreview } from '@/features/backup/viewModel';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
import BackupSelectionGroup from './BackupSelectionGroup.vue';

const config = defineModel<BackupConfigDraft>('config', { required: true });

const props = defineProps<{
  dirty: boolean;
  saving: boolean;
  timestamp: string;
  disabled?: boolean;
}>();

const formRef = ref<FormInst | null>(null);
const lastCleaningStrategy = ref<1 | 2>(config.value.backupCleanStrategy === 2 ? 2 : 1);
const cleaningEnabled = computed({
  get: () => config.value.backupCleanStrategy !== 0,
  set: enabled => {
    if (enabled) {
      config.value.backupCleanStrategy = lastCleaningStrategy.value;
      return;
    }

    if (config.value.backupCleanStrategy === 1 || config.value.backupCleanStrategy === 2) {
      lastCleaningStrategy.value = config.value.backupCleanStrategy;
    }
    config.value.backupCleanStrategy = 0;
  },
});

watch(
  () => config.value.backupCleanStrategy,
  strategy => {
    if (strategy === 1 || strategy === 2) lastCleaningStrategy.value = strategy;
  }
);

const rules: FormRules = {
  autoBackupTime: {
    validator: () =>
      !config.value.autoBackupEnable ||
      config.value.autoBackupTime.trim().length > 0 ||
      new Error('请输入备份间隔'),
    trigger: ['blur', 'change'],
  },
  backupCleanKeepCount: {
    validator: () =>
      config.value.backupCleanStrategy !== 1 ||
      (Number.isInteger(config.value.backupCleanKeepCount) &&
        config.value.backupCleanKeepCount >= 1) ||
      new Error('保留数量至少为 1'),
    trigger: ['blur', 'change'],
  },
  backupCleanKeepDur: {
    validator: () =>
      config.value.backupCleanStrategy !== 2 ||
      /^[0-9]+(ns|us|ms|s|m|h)$/.test(config.value.backupCleanKeepDur.trim()) ||
      new Error('请输入有效时长，例如 720h'),
    trigger: ['blur', 'change'],
  },
  backupCleanTriggers: {
    validator: () =>
      config.value.backupCleanStrategy === 0 ||
      config.value.backupCleanTriggers.length > 0 ||
      new Error('至少选择一种触发方式'),
    trigger: ['change'],
  },
  backupCleanCron: {
    validator: () =>
      config.value.backupCleanStrategy === 0 ||
      !config.value.backupCleanTriggers.includes('cron') ||
      config.value.backupCleanCron.trim().length > 0 ||
      new Error('请输入定时间隔'),
    trigger: ['blur', 'change'],
  },
};

const cleanTriggerOptions: Array<{ value: BackupCleanTriggerKey; label: string }> = [
  { value: 'afterAutoBackup', label: '自动备份后' },
  { value: 'cron', label: '定时' },
];

const autoBackupPreview = computed(() =>
  buildBackupFilenamePreview(
    props.timestamp,
    buildBackupConfigPayload(config.value).autoBackupSelection,
    true
  )
);
</script>

<style scoped>
.backup-config-panel :deep(.n-form) {
  display: flex;
  flex-direction: column;
  gap: var(--sd-space-xs);
}

.backup-config-panel__label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.backup-config-panel__help {
  color: var(--sd-text-muted);
  cursor: help;
}

.backup-config-panel__preview {
  word-break: break-all;
}
</style>
