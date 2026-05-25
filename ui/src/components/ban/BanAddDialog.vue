<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="添加黑白名单条目"
    class="the-dialog"
    :mask-closable="false"
  >
    <n-form ref="formRef" :model="formModel" :rules="rules" label-placement="left" label-width="88">
      <n-form-item label="用户 ID" path="id">
        <n-input
          v-model:value="form.id"
          placeholder="例如 QQ:12345 或 QQ-Group:12345"
        />
      </n-form-item>
      <n-form-item label="名称">
        <n-input
          v-model:value="form.name"
          placeholder="可留空，后端会使用缓存名或未知名"
        />
      </n-form-item>
      <n-form-item label="原因">
        <n-input
          v-model:value="form.reason"
          placeholder="默认使用“骰主后台设置”"
        />
      </n-form-item>
      <n-form-item label="身份">
        <n-radio-group v-model:value="form.rank">
          <n-radio :value="-30">禁用</n-radio>
          <n-radio :value="30">信任</n-radio>
        </n-radio-group>
      </n-form-item>
    </n-form>
    <template #footer>
      <n-flex justify="end">
        <n-button @click="show = false">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="handleSubmit">
          保存
        </n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { FormInst, FormRules } from 'naive-ui';
import type { BanAddFormModel } from '@/features/ban/viewModel';

const show = defineModel<boolean>('show', { required: true });
const form = defineModel<BanAddFormModel>('form', { required: true });

defineProps<{
  submitting: boolean;
}>();

const emit = defineEmits<{
  submit: [];
}>();

const formRef = ref<FormInst | null>(null);

const formModel = computed(() => form.value);

const rules: FormRules = {
  id: [
    { required: true, message: '请输入帐号或群组 ID', trigger: ['blur', 'change'] },
  ],
};

async function handleSubmit() {
  await formRef.value?.validate();
  emit('submit');
}
</script>
