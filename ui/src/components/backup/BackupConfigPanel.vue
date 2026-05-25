<template>
  <n-card class="backup-config-panel" :bordered="false">
    <template #header>
      <div class="backup-config-panel__head">
        <div>
          <h2>备份设置</h2>
          <p>配置自动备份、备份范围和历史备份清理策略。</p>
        </div>
        <n-button
          type="primary"
          :disabled="!dirty"
          :loading="saving"
          @click="emit('save')"
        >
          <template #icon>
            <n-icon>
              <i-carbon-save />
            </n-icon>
          </template>
          保存设置
        </n-button>
      </div>
    </template>

    <n-form :model="config" label-placement="top">
      <section class="backup-config-panel__section">
        <div class="backup-config-panel__section-title">
          <h3>自动备份</h3>
          <n-switch v-model:value="config.autoBackupEnable">
            <template #checked>开启</template>
            <template #unchecked>关闭</template>
          </n-switch>
        </div>

        <div v-show="config.autoBackupEnable" class="backup-config-panel__fields">
          <n-form-item path="autoBackupTime">
            <template #label>
              <span class="backup-config-panel__label">
                备份间隔
                <n-tooltip placement="top">
                  <template #trigger>
                    <n-icon class="backup-config-panel__help">
                      <i-carbon-help-filled />
                    </n-icon>
                  </template>
                  备份间隔表达式使用 robfig/cron 格式，例如 @every 12h。
                </n-tooltip>
              </span>
            </template>
            <n-input v-model:value="config.autoBackupTime" placeholder="@every 12h" />
          </n-form-item>

          <n-form-item path="autoBackupSelectionList" label="备份范围">
            <BackupSelectionGroup v-model:value="config.autoBackupSelectionList" />
          </n-form-item>

          <n-form-item label="备份文件名预览">
            <n-text type="info" class="backup-config-panel__preview">
              {{ autoBackupPreview }}
            </n-text>
          </n-form-item>
        </div>
      </section>

      <section class="backup-config-panel__section">
        <div class="backup-config-panel__section-title">
          <h3>自动清理</h3>
        </div>

        <n-form-item path="backupCleanStrategy" label="清理模式">
          <n-radio-group v-model:value="config.backupCleanStrategy" size="small">
            <n-radio-button :value="0">关闭</n-radio-button>
            <n-radio-button :value="1">保留一定数量</n-radio-button>
            <n-radio-button :value="2">保留一定时间内</n-radio-button>
          </n-radio-group>
        </n-form-item>

        <n-form-item v-if="config.backupCleanStrategy === 1" path="backupCleanKeepCount" label="保留数量">
          <n-input-number v-model:value="config.backupCleanKeepCount" :min="1" :step="1" />
        </n-form-item>

        <n-form-item v-if="config.backupCleanStrategy === 2" path="backupCleanKeepDur">
          <template #label>
            <span class="backup-config-panel__label">
              保留时间
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon class="backup-config-panel__help">
                    <i-carbon-help-filled />
                  </n-icon>
                </template>
                支持 h、m、s，例如 720h 表示保留 30 天内的备份。
              </n-tooltip>
            </span>
          </template>
          <n-input v-model:value="config.backupCleanKeepDur" placeholder="720h" />
        </n-form-item>

        <template v-if="config.backupCleanStrategy !== 0">
          <n-form-item path="backupCleanTriggers">
            <template #label>
              <span class="backup-config-panel__label">
                触发方式
                <n-tooltip placement="top">
                  <template #trigger>
                    <n-icon class="backup-config-panel__help">
                      <i-carbon-help-filled />
                    </n-icon>
                  </template>
                  自动备份后会在每次自动备份完成后顺便清理；定时会按照 cron 表达式单独清理。
                </n-tooltip>
              </span>
            </template>
            <n-checkbox-group v-model:value="config.backupCleanTriggers" :options="cleanTriggerOptions" />
          </n-form-item>

          <n-form-item path="backupCleanCron" label="定时间隔">
            <n-input
              v-model:value="config.backupCleanCron"
              :disabled="!config.backupCleanTriggers.includes('cron')"
              placeholder="0 0 * * *"
            />
          </n-form-item>
        </template>
      </section>

      <n-alert type="info" :bordered="false">
        恢复备份时，将骰子彻底关闭，解压备份压缩包到骰子目录。若提示是否覆盖，选择全部即可。
      </n-alert>
    </n-form>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { BackupCleanTriggerKey, BackupConfigDraft } from '@/features/backup/viewModel';
import { buildBackupConfigPayload, buildBackupFilenamePreview } from '@/features/backup/viewModel';
import BackupSelectionGroup from './BackupSelectionGroup.vue';

const config = defineModel<BackupConfigDraft>('config', { required: true });

const props = defineProps<{
  dirty: boolean;
  saving: boolean;
  timestamp: string;
}>();

const emit = defineEmits<{
  save: [];
}>();

const cleanTriggerOptions: Array<{ value: BackupCleanTriggerKey; label: string }> = [
  { value: 'afterAutoBackup', label: '自动备份后' },
  { value: 'cron', label: '定时' },
];

const autoBackupPreview = computed(() =>
  buildBackupFilenamePreview(
    props.timestamp,
    buildBackupConfigPayload(config.value).autoBackupSelection,
    true,
  ),
);
</script>

<style scoped>
.backup-config-panel__head,
.backup-config-panel__section-title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.backup-config-panel__head h2,
.backup-config-panel__section-title h3 {
  margin: 0;
}

.backup-config-panel__head p {
  margin: 4px 0 0;
  color: var(--sd-text-muted);
  font-size: 13px;
}

.backup-config-panel__section {
  padding-top: 18px;
  border-top: 1px solid var(--sd-border-soft);
}

.backup-config-panel__section:first-child {
  padding-top: 0;
  border-top: 0;
}

.backup-config-panel__fields {
  margin-top: 12px;
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

@media (max-width: 720px) {
  .backup-config-panel__head {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
