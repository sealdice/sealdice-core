<script setup lang="ts">
import { DocumentChecked } from "@element-plus/icons-vue";
import { getBanConfig, setBanConfig } from "~/api/banconfig";
import type { BanConfig } from "~/type";

const banConfig = ref<BanConfig>({} as BanConfig)
const modified = ref<boolean>(false)

const banConfigSave = async () => {
  await setBanConfig(banConfig.value)
  await configGet()
  ElMessage.success('已保存')
  modified.value = false
  await nextTick(() => {
    modified.value = false
  })
}

const configGet = async () => {
  banConfig.value = await getBanConfig()
}

onBeforeMount(async () => {
  await configGet()
  modified.value = false
})

watch(banConfig, () => {
  modified.value = true
}, {deep: true});

</script>

<template>
  <header>
    <el-button type="primary" :icon="DocumentChecked" @click="banConfigSave">保存设置</el-button>
    <el-text style="margin-left: 1rem" v-if="modified" type="danger" size="large" tag="strong">
      内容已修改，不要忘记保存！
    </el-text>
  </header>

  <h4>基本设置</h4>
  <el-space wrap>
    <el-text>黑名单惩罚：</el-text>
    <el-checkbox v-model="banConfig.banBehaviorRefuseReply">拒绝回复</el-checkbox>
    <el-checkbox v-model="banConfig.banBehaviorRefuseInvite">拒绝邀请</el-checkbox>
    <el-checkbox v-model="banConfig.banBehaviorQuitLastPlace">退出事发群</el-checkbox>
    <!-- <div>自动拉黑时长(分钟): <el-input style="max-width: 5rem;" type="number" v-model="banConfig.autoBanMinutes"></el-input></div> -->
    <el-checkbox v-model="banConfig.banBehaviorQuitPlaceImmediately">使用时立即退出群</el-checkbox>
    <el-checkbox v-model="banConfig.banBehaviorQuitIfAdmin">使用者为管理员立即退群，为普通群员进行通告</el-checkbox>
  </el-space>

  <h4>怒气值设置</h4>
  <div class="mt-4 mb-6">
    <el-text type="warning" tag="p">说明：海豹的黑名单使用积分制，每当用户做出恶意行为，其积分上涨一定数值，到达阈值后自动进入黑名单。会通知邀请者、通知列表、事发群（如果可能）。</el-text>
  </div>
  <el-form size="small">
    <el-form-item label="警告阈值">
      <el-input-number v-model="banConfig.thresholdWarn" :min="0" :step="1" step-strictly></el-input-number>
    </el-form-item>
    <el-form-item label="拉黑阈值">
      <el-input-number v-model="banConfig.thresholdBan" :min="0" :step="1" step-strictly></el-input-number>
    </el-form-item>

    <el-form-item class="mt-10" label="禁言增加">
      <el-input-number v-model="banConfig.scoreGroupMuted" :min="0" :step="1" step-strictly></el-input-number>
    </el-form-item>
    <el-form-item label="踢出增加">
      <el-input-number v-model="banConfig.scoreGroupKicked" :min="0" :step="1" step-strictly></el-input-number>
    </el-form-item>
    <el-form-item label="刷屏增加">
      <el-input-number v-model="banConfig.scoreTooManyCommand" :min="0" :step="1" step-strictly></el-input-number>
    </el-form-item>
    <el-form-item label="每分钟下降">
      <el-input-number v-model="banConfig.scoreReducePerMinute" :min="0" :step="1" step-strictly></el-input-number>
    </el-form-item>

    <el-form-item class="mt-10" label="群组连带责任">
      <el-input-number v-model="banConfig.jointScorePercentOfGroup" :min="0" :max="1" :step="0.1"></el-input-number>
    </el-form-item>
    <el-form-item label="邀请人连带责任">
      <el-input-number v-model="banConfig.jointScorePercentOfInviter" :min="0" :max="1" :step="0.1"></el-input-number>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="css">

</style>