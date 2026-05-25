<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="上传帮助文档"
    class="the-dialog"
    :mask-closable="false"
  >
    <n-alert v-show="group === 'default'" type="warning" class="mb-4">
      更具体的分组能提供组内搜索命令
      <n-tag size="small" :bordered="false">.find#&lt;分组&gt; &lt;搜索内容&gt;</n-tag>
      ，是否一定要选择默认分组？
    </n-alert>
    <n-form ref="formRef" :model="formModel" :rules="uploadRules" label-placement="left">
      <n-form-item label="分组" path="group">
        <n-select
          v-model:value="group"
          placeholder="选择分组"
          filterable
          clearable
          tag
          :options="groups"
        />
      </n-form-item>
      <n-form-item label="帮助文档">
        <n-upload
          v-model:file-list="fileList"
          :default-upload="false"
          :show-file-list="true"
          multiple
          accept=".json,.xlsx"
        >
          <n-button type="primary">
            <template #icon>
              <n-icon><i-carbon-upload /></n-icon>
            </template>
            选择文件
          </n-button>
        </n-upload>
      </n-form-item>
    </n-form>
    <template #footer>
      <n-flex justify="end">
        <n-button @click="closeDialog">取消</n-button>
        <n-button type="primary" :loading="busy" @click="handleSubmit">
          上传
        </n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { FormInst, FormRules, UploadFileInfo } from 'naive-ui';
import { isHelpdocUploadFileAccepted } from '@/features/helpdoc/viewModel';

const show = defineModel<boolean>('show', { required: true });
const group = defineModel<string>('group', { required: true });
const fileList = defineModel<UploadFileInfo[]>('fileList', { required: true });

defineProps<{
  groups: { label: string; value: string }[];
  busy: boolean;
}>();

const emit = defineEmits<{
  submit: [files: File[]];
}>();

const message = useMessage();
const formRef = ref<FormInst | null>(null);

const uploadRules: FormRules = {
  group: [
    { required: true, message: '请选择分组', trigger: ['blur', 'change'] },
    {
      validator: (_rule, value: string) => value !== 'builtin',
      message: '不能为内置分组',
      trigger: ['blur', 'change'],
    },
  ],
};

const formModel = computed(() => ({
  group: group.value,
}));

function closeDialog() {
  show.value = false;
}

async function handleSubmit() {
  await formRef.value?.validate();
  const files = fileList.value
    .map(item => item.file)
    .filter((file): file is File => Boolean(file));
  if (!files.length) {
    message.error('请选择文件');
    return;
  }
  if (files.some(file => !isHelpdocUploadFileAccepted(file.name))) {
    message.error('仅支持上传 .json 或 .xlsx 帮助文档');
    return;
  }
  emit('submit', files);
}
</script>
