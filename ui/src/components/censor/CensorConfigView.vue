<template>
  <SettingCategoryBox title="匹配选项" padded>
    <n-form label-placement="left" label-width="auto">
      <n-form-item>
        <template #label>
          <n-text>拦截范围</n-text>
          <n-tooltip>
            <template #trigger>
              <n-icon>
                <i-tabler-help-circle />
              </n-icon>
            </template>
            发出的消息： 拦截骰子发出的内容，进行检查。未通过检查，替换为
            <n-tag size="small" :bordered="false">拦截_完全拦截_发出的消息</n-tag>
            的内容。<br />
            收到的指令： 拦截骰子收到的命令文本进行检查，如收到「.rd
            进行一次骰点」时，会检查其中的「进行一次骰点」，未通过检查则发送
            <n-tag size="small" :bordered="false">拦截_完全拦截_收到的指令</n-tag>
            的内容<br />
            收到的所有消息： 会对所有收到的消息(所有群内聊天)进行检查，未通过检查默认不做响应，如
            <n-tag size="small" :bordered="false">拦截_完全拦截_收到的所有消息</n-tag>
            不为空时会发送拦截提示。
          </n-tooltip>
        </template>
        <n-radio-group v-model:value="config.mode" size="small">
          <n-radio :value="CENSOR_MODES.replyOutput">发出的消息</n-radio>
          <n-radio :value="CENSOR_MODES.commandInput">收到的指令</n-radio>
          <n-radio :value="CENSOR_MODES.allInput">收到的所有消息(慎用)</n-radio>
        </n-radio-group>
      </n-form-item>
      <n-form-item label="大小写敏感">
        <n-checkbox v-model:checked="config.caseSensitive">开启</n-checkbox>
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-text>匹配拼音</n-text>
          <n-tooltip>
            <template #trigger>
              <n-icon>
                <i-tabler-help-circle />
              </n-icon>
            </template>
            匹配敏感词拼音，勾选大小写敏感时该项无效。
          </n-tooltip>
        </template>
        <n-checkbox v-model:checked="config.matchPinyin">开启</n-checkbox>
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-text>过滤字符正则</n-text>
          <n-tooltip>
            <template #trigger>
              <n-icon>
                <i-tabler-help-circle />
              </n-icon>
            </template>
            判断敏感词时，忽略过滤字符。如敏感词为 "114514"，指定过滤字符为空白，则
            "114&nbsp;&nbsp;&nbsp;514" 也会命中敏感词。
          </n-tooltip>
        </template>
        <n-input v-model:value="config.filterRegex" placeholder="" class="censor-regex-input" />
      </n-form-item>
    </n-form>
  </SettingCategoryBox>

  <SettingCategoryBox title="响应设置" padded>
    <TipBox type="warning">
      <n-text type="warning">
        <span>提示：</span>
        <ul class="ml-4 list-disc">
          <li><p>超过阈值时，对应用户该等级的计数会被清空重新计算。</p></li>
          <li>
            <p>
              增加怒气值时，会计算群组和邀请人的连带责任。连带责任比例在
              <strong>综合设置 > 黑白名单 > 设置选项</strong> 中调整。
            </p>
          </li>
        </ul>
      </n-text>
    </TipBox>

    <n-form label-placement="left" label-width="auto">
      <n-form-item>
        <template #label>
          <CensorSensitiveTag :level="1" />
        </template>
        <LevelConfigEditor v-model:level="config.levelConfig.notice" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <CensorSensitiveTag :level="2" />
        </template>
        <LevelConfigEditor v-model:level="config.levelConfig.caution" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <CensorSensitiveTag :level="3" />
        </template>
        <LevelConfigEditor v-model:level="config.levelConfig.warning" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <CensorSensitiveTag :level="4" />
        </template>
        <LevelConfigEditor v-model:level="config.levelConfig.danger" />
      </n-form-item>
    </n-form>
  </SettingCategoryBox>
</template>

<script setup lang="tsx">
import { defineComponent } from 'vue';
import type { CensorConfigBody, CensorLevelConfig } from '@/api';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
import TipBox from '@/components/shared/TipBox.vue';
import CensorSensitiveTag from './CensorSensitiveTag.vue';
import { CENSOR_HANDLERS, CENSOR_MODES } from '@/features/censor/viewModel';

const config = defineModel<CensorConfigBody>('config', { required: true });

// 内层编辑器的 prop 名与外层 defineModel 同名会让 lint 把模板里的 config
// 误判为 prop，因此这里改用 level。
const LevelConfigEditor = defineComponent({
  name: 'LevelConfigEditor',
  props: {
    level: {
      type: Object as () => CensorLevelConfig,
      required: true,
    },
  },
  emits: ['update:level'],
  setup(props) {
    return () => (
      <n-flex align="start" class="level-config-editor">
        <n-flex align="center" class="level-config-threshold">
          <n-text>用户触发超过</n-text>
          <n-input-number
            v-model:value={props.level.threshold}
            class="w-28"
            size="small"
            step={1}
            min={0}
            precision={0}
          />
          <n-text>次时：</n-text>
        </n-flex>
        <n-flex vertical class="level-config-actions">
          <n-checkbox-group v-model:value={props.level.handlers} class="level-config-handlers">
            {CENSOR_HANDLERS.map(handle => (
              <n-checkbox key={handle.key} value={handle.key}>
                {handle.name}
              </n-checkbox>
            ))}
          </n-checkbox-group>
          <n-flex align="center" class="level-config-score">
            <n-text>怒气值</n-text>
            <n-input-number
              v-model:value={props.level.score}
              class="w-28"
              size="small"
              step={1}
              min={0}
              precision={0}
            />
          </n-flex>
        </n-flex>
      </n-flex>
    );
  },
});
</script>

<style scoped>
.censor-regex-input {
  width: min(100%, 12rem);
}

.level-config-editor {
  display: grid !important;
  grid-template-columns: minmax(15rem, 18rem) minmax(18rem, 1fr);
  gap: 0.75rem 1.5rem;
  width: 100%;
}

.level-config-threshold,
.level-config-actions {
  min-width: 0;
}

.level-config-handlers {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr));
  gap: 0.35rem 0.75rem;
}

.level-config-score {
  margin-top: 0.35rem;
}

@media screen and (max-width: 639.9px) {
  .censor-regex-input {
    width: 100%;
  }

  .level-config-editor {
    grid-template-columns: 1fr;
  }
}
</style>
