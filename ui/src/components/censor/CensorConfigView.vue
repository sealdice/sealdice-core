<template>
  <header class="flex items-center">
    <n-button type="info" secondary :loading="saving" @click="emit('save')">
      <template #icon>
        <i-carbon-save />
      </template>
      保存设置
    </n-button>
    <n-text v-if="modified" type="error" tag="strong" class="ml-4 text-base">
      内容已修改，不要忘记保存！
    </n-text>
  </header>
  <n-form label-placement="left" label-width="auto">
    <h4>匹配选项</h4>
    <n-form-item>
      <template #label>
        <n-text>拦截范围</n-text>
        <n-tooltip>
          <template #trigger>
            <n-icon>
              <i-carbon-help-filled />
            </n-icon>
          </template>
          发出的消息： 拦截骰子发出的内容，进行检查。未通过检查，替换为
          <n-tag size="small" type="info" :bordered="false">拦截_完全拦截_发出的消息</n-tag>
          的内容。<br />
          收到的指令： 拦截骰子收到的命令文本进行检查，如收到「.rd
          进行一次骰点」时，会检查其中的「进行一次骰点」，未通过检查则发送
          <n-tag size="small" type="info" :bordered="false">拦截_完全拦截_收到的指令</n-tag>
          的内容<br />
          收到的所有消息： 会对所有收到的消息(所有群内聊天)进行检查，未通过检查默认不做响应，如
          <n-tag size="small" type="info" :bordered="false">拦截_完全拦截_收到的所有消息</n-tag>
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
      <n-checkbox v-model:checked="config.caseSensitive" label="开启" />
    </n-form-item>
    <n-form-item>
      <template #label>
        <n-text>匹配拼音</n-text>
        <n-tooltip>
          <template #trigger>
            <n-icon>
              <i-carbon-help-filled />
            </n-icon>
          </template>
          匹配敏感词拼音，勾选大小写敏感时该项无效。
        </n-tooltip>
      </template>
      <n-checkbox v-model:checked="config.matchPinyin" label="开启" />
    </n-form-item>
    <n-form-item>
      <template #label>
        <n-text>过滤字符正则</n-text>
        <n-tooltip>
          <template #trigger>
            <n-icon>
              <i-carbon-help-filled />
            </n-icon>
          </template>
          判断敏感词时，忽略过滤字符。如敏感词为 "114514"，指定过滤字符为空白，则 "114&nbsp;&nbsp;&nbsp;514" 也会命中敏感词。
        </n-tooltip>
      </template>
      <n-input v-model:value="config.filterRegex" placeholder="" style="width: 12rem" />
    </n-form-item>
  </n-form>

  <h4>响应设置</h4>
  <TipBox type="warning" class="my-4">
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
      <LevelConfigEditor v-model:config="config.levelConfig.notice" />
    </n-form-item>
    <n-form-item>
      <template #label>
        <CensorSensitiveTag :level="2" />
      </template>
      <LevelConfigEditor v-model:config="config.levelConfig.caution" />
    </n-form-item>
    <n-form-item>
      <template #label>
        <CensorSensitiveTag :level="3" />
      </template>
      <LevelConfigEditor v-model:config="config.levelConfig.warning" />
    </n-form-item>
    <n-form-item>
      <template #label>
        <CensorSensitiveTag :level="4" />
      </template>
      <LevelConfigEditor v-model:config="config.levelConfig.danger" />
    </n-form-item>
  </n-form>
</template>

<script setup lang="tsx">
import { defineComponent } from 'vue';
import type { CensorConfigBody, CensorLevelConfig } from '@/api';
import TipBox from '@/components/shared/TipBox.vue';
import CensorSensitiveTag from './CensorSensitiveTag.vue';
import { CENSOR_HANDLERS, CENSOR_MODES } from '@/features/censor/viewModel';

const config = defineModel<CensorConfigBody>('config', { required: true });

defineProps<{
  saving: boolean;
  modified: boolean;
}>();

const emit = defineEmits<{
  save: [];
}>();

const LevelConfigEditor = defineComponent({
  name: 'LevelConfigEditor',
  props: {
    config: {
      type: Object as () => CensorLevelConfig,
      required: true,
    },
  },
  emits: {
    'update:config': (_value: CensorLevelConfig) => true,
  },
  setup(props) {
    return () => (
      <n-flex align='center'>
        <n-flex align='center' wrap>
          <n-text>用户触发超过</n-text>
          <n-input-number
            v-model:value={props.config.threshold}
            class='w-28'
            size='small'
            step={1}
            min={0}
            precision={0}
          />
          <n-text>次时：</n-text>
        </n-flex>
        <n-flex justify='center' vertical wrap>
          <n-checkbox-group v-model:value={props.config.handlers}>
            {CENSOR_HANDLERS.map(handle => (
              <n-checkbox key={handle.key} value={handle.key}>
                {handle.name}
              </n-checkbox>
            ))}
          </n-checkbox-group>
          <n-flex align='center'>
            <n-text>怒气值</n-text>
            <n-input-number
              v-model:value={props.config.score}
              class='w-28'
              size='small'
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
