<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="许可协议"
    :mask-closable="false"
    :close-on-esc="false"
    :closable="false"
    class="the-dialog"
  >
    <pre class="license-text">尊敬的用户，欢迎您选择由木落等研发的海豹骰点核心（SealDice），在您使用自定义功能前，请务必仔细阅读使用须知，当您使用我们提供的服务时，即代表您已同意使用须知的内容。

  您需了解，海豹核心官方版只支持 TRPG 功能，娱乐功能定制化请自便，和海豹无关。
  您清楚并明白您对通过骰子提供的全部内容负责，包括自定义回复、非自带的插件、牌堆。海豹骰不对非自身提供以外的内容合法性负责。您不得在使用海豹骰服务时，导入包括但不限于以下情形的内容：
  (1) 反对中华人民共和国宪法所确定的基本原则的；
  (2) 危害国家安全，泄露国家秘密，颠覆国家政权，破坏国家统一的;
  (3) 损害国家荣誉和利益的;
  (4) 煽动民族仇恨、民族歧视、破坏民族团结的；
  (5) 破坏国家宗教政策，宣扬邪教和封建迷信的;
  (6) 散布谣言，扰乱社会秩序，破坏社会稳定的；
  (7) 散布淫秽、色情、赌博、暴力、凶杀、恐怖或者教唆犯罪的；
  (8) 侮辱或者诽谤他人，侵害他人合法权益的；
  (9) 宣扬、教唆使用外挂、私服、病毒、恶意代码、木马及其相关内容的；
  (10) 侵犯他人知识产权或涉及第三方商业秘密及其他专有权利的；
  (11) 散布任何贬损、诋毁、恶意攻击海豹骰及开发人员、海洋馆工作人员、mod 编写人员、关联合作者的；
  (12) 含有中华人民共和国法律、行政法规、政策、上级主管部门下发通知中所禁止的其他内容的。

  一旦查实您有以上禁止行为，请立即停用海豹骰。同时我们也会主动对你进行举报。</pre>
    <template #footer>
      <n-flex>
        <n-button type="primary" :loading="loading" @click="emit('accept')">
          我同意
        </n-button>
        <n-button type="error" @click="emit('refuse')">我拒绝</n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
const show = defineModel<boolean>('show', { required: true });

defineProps<{
  loading: boolean;
}>();

const emit = defineEmits<{
  accept: [];
  refuse: [];
}>();
</script>

<style scoped>
.license-text {
  white-space: pre-wrap;
}
</style>
