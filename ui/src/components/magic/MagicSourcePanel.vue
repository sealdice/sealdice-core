<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import type { FormInst, FormRules } from 'naive-ui';
import { getErrorMessage } from '@/features/auth/error';
import { useMagicInspectMutation } from '@/features/magic/queries';
import {
  getMagicInspectSummary,
  normalizeMagicInspectResult,
  type MagicInspectSummaryInput,
} from '@/features/magic/viewModel';

const formRef = ref<FormInst | null>(null);
const message = useMessage();
const inspectMutation = useMagicInspectMutation();
const result = ref<MagicInspectSummaryInput | null>(null);

const form = reactive({
  kind: 'sqlite',
  sqlitePath: '',
  dsn: '',
});

const rules: FormRules = {
  kind: {
    required: true,
    message: '请选择源数据库类型',
    trigger: ['change'],
  },
  sqlitePath: {
    required: true,
    message: '请输入 SQLite 数据文件路径',
    trigger: ['blur', 'input'],
    validator: () => form.kind !== 'sqlite' || form.sqlitePath.trim().length > 0,
  },
  dsn: {
    required: true,
    message: '请输入数据库连接串',
    trigger: ['blur', 'input'],
    validator: () => form.kind === 'sqlite' || form.dsn.trim().length > 0,
  },
};

const summary = computed(() => (result.value ? getMagicInspectSummary(result.value) : null));

async function inspectSource() {
  await formRef.value?.validate();
  try {
    const apiResult = await inspectMutation.mutateAsync({
      kind: form.kind as 'sqlite' | 'mysql' | 'postgres',
      sqlitePath: form.sqlitePath,
      dsn: form.dsn,
    });
    result.value = normalizeMagicInspectResult(apiResult);
    message.success('源检查完成');
  } catch (error) {
    message.error(getErrorMessage(error, '源检查失败'));
  }
}
</script>

<template>
  <section class="magic-source-panel">
    <div class="magic-source-panel__intro">
      <n-tag :bordered="false" type="warning">Sealdice Magic</n-tag>
      <h2>源数据库检查</h2>
      <p>先识别当前数据库结构，再决定是否需要 SQLite 修复、1.5.0 中转升级和 Addax 迁移。</p>
    </div>

    <n-grid :cols="24" :x-gap="16" :y-gap="16">
      <n-form
        ref="formRef"
        class="magic-source-panel__form"
        :model="form"
        :rules="rules"
        label-placement="top"
      >
        <n-gi :span="24">
          <n-form-item path="kind" label="源数据库类型">
            <n-radio-group v-model:value="form.kind" name="magic-source-kind">
              <n-space>
                <n-radio value="sqlite">SQLite</n-radio>
                <n-radio value="mysql">MySQL</n-radio>
                <n-radio value="postgres">PostgreSQL</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item>
        </n-gi>

        <n-gi v-if="form.kind === 'sqlite'" :span="24">
          <n-form-item path="sqlitePath" label="SQLite 数据文件">
            <n-input
              v-model:value="form.sqlitePath"
              placeholder="例如 /data/default/data.db"
            />
          </n-form-item>
        </n-gi>

        <n-gi v-else :span="24">
          <n-form-item path="dsn" label="数据库连接串">
            <n-input
              v-model:value="form.dsn"
              type="textarea"
              :rows="4"
              placeholder="例如 postgres://user:pass@127.0.0.1:5432/sealdice?sslmode=disable"
            />
          </n-form-item>
        </n-gi>

        <n-gi :span="24">
          <n-space justify="end">
            <n-button type="primary" :loading="inspectMutation.isPending.value" @click="inspectSource">
              开始检查
            </n-button>
          </n-space>
        </n-gi>
      </n-form>

      <n-gi v-if="summary" :span="24">
        <n-alert :type="summary.tone" :bordered="false">
          <template #header>{{ summary.headline }}</template>
          <div class="magic-source-panel__summary">
            <p>阶段：{{ summary.stageText }}</p>
            <p>下一步：{{ summary.nextActionText }}</p>
            <p>识别表数：{{ summary.tableCount }}</p>
            <p>样例表：{{ summary.tablePreview || '无' }}</p>
          </div>
        </n-alert>
      </n-gi>

      <n-gi v-if="result?.messages.length" :span="24">
        <n-card size="small" :bordered="false" class="magic-source-panel__messages">
          <n-space vertical :size="10">
            <div v-for="message in result.messages" :key="message" class="magic-source-panel__message">
              {{ message }}
            </div>
          </n-space>
        </n-card>
      </n-gi>
    </n-grid>
  </section>
</template>

<style scoped>
.magic-source-panel {
  display: grid;
  gap: 16px;
}

.magic-source-panel__intro {
  display: grid;
  gap: 8px;
  padding: 22px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 22px;
  background:
    radial-gradient(circle at top right, color-mix(in srgb, var(--n-warning-color), transparent 78%), transparent 32%),
    linear-gradient(135deg, var(--sd-bg-elevated), var(--sd-bg-elevated-soft));
}

.magic-source-panel__intro h2 {
  margin: 0;
  font-size: 28px;
  line-height: 1.1;
}

.magic-source-panel__intro p {
  margin: 0;
  color: var(--n-text-color-2);
}

.magic-source-panel__form {
  display: grid;
  gap: 4px;
  padding: 20px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 20px;
  background: var(--sd-bg-elevated);
}

.magic-source-panel__summary {
  display: grid;
  gap: 6px;
}

.magic-source-panel__summary p,
.magic-source-panel__message {
  margin: 0;
}

.magic-source-panel__messages {
  border-radius: 18px;
  background: var(--sd-bg-elevated);
}
</style>
