<template>
  <el-affix :offset="60" v-if="modified">
    <div class="tip-danger">
      <el-text type="danger" size="large" tag="strong">内容已修改，不要忘记保存！</el-text>
    </div>
  </el-affix>

  <h2 class="h-2">高级设置</h2>
  <div class="tip-danger">
    <el-text>
      此处是面向开发者或进阶用户的隐藏设置页，下列的设置项可能会对海豹核心的功能造成重大影响。<br/>
      一些尚在测试的不稳定设置项，以及 <strong>普通骰主无需关注</strong> 的设置项会被放在此处。<br/>
      此处的设置项不保证稳定提供，在未来版本随时可能会被移除。<br/>
      <br/>
      <strong>除非你知道自己在做什么，否则不要修改此处的任何设置项！</strong><br/>
      <br/>
      <em>如果你误操作修改了此处设置，希望恢复默认，请手动删除 <code>data/default/advanced.yaml</code> 文件。</em>
    </el-text>
  </div>

  <el-form label-width="130px">
    <el-form-item label="显示高级设置页">
      <template #label>
        <span>显示高级设置页</span>
        <el-tooltip raw-content content="设置是否显示高级设置页，只影响展示">
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-switch v-model="config.show"/>
    </el-form-item>
    <el-form-item label="启用高级设置">
      <template #label>
        <span>启用高级设置</span>
        <el-tooltip raw-content content="设置是否启用高级设置，关闭时下列设置无效">
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-switch v-model="config.enable"/>
    </el-form-item>

    <h3>跑团日志</h3>
    <el-form-item label="自定义后端 URL">
      <template #label>
        <span>自定义后端 URL</span>
        <el-tooltip raw-content content="设置第三方跑团日志后端 URL">
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-input v-model="config.storyLogBackendUrl" style="width: 30rem"/>
    </el-form-item>
    <el-form-item label="API 版本">
      <template #label>
        <span>API 版本</span>
        <el-tooltip raw-content content="指定后端的 API 版本">
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-input v-model="config.storyLogApiVersion" style="width: 10rem"/>
    </el-form-item>
    <el-form-item label="Token">
      <template #label>
        <span>Token</span>
        <el-tooltip raw-content content="指定传递给后端的 token">
          <el-icon>
            <question-filled/>
          </el-icon>
        </el-tooltip>
      </template>
      <el-input v-model="config.storyLogBackendToken" style="width: 30rem"/>
    </el-form-item>

    <el-form-item label="" label-width="1rem" style="margin-top: 3rem;" v-if="modified">
      <el-button type="danger" @click="submitGiveup">放弃改动</el-button>
      <el-button type="success" @click="submit">保存设置</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import {QuestionFilled} from "@element-plus/icons-vue";
import {useStore} from "~/store";
import {cloneDeep} from "lodash-es";
import {AdvancedConfig} from "~/type";

const emit = defineEmits(['update:advanced-settings-show'])

const store = useStore()

const config = ref<AdvancedConfig>({
  show: false,
  enable: false,
  storyLogBackendUrl: "",
  storyLogApiVersion: "",
  storyLogBackendToken: "",
})

onBeforeMount(async () => {
  config.value = await store.diceAdvancedConfigGet()
  nextTick(() => {
    modified.value = false
  })
})

const modified = ref(false)
watch(() => config, (newValue, oldValue) => { //直接监听
  modified.value = true
}, {
  deep: true
});

const submit = async () => {
  await store.diceAdvancedConfigSet(config.value)
  config.value = await store.diceAdvancedConfigGet()
  modified.value = false
  emit('update:advanced-settings-show', config.value.show)
  nextTick(async () => {
    modified.value = false
  })
}

const submitGiveup = async () => {
  config.value = await store.diceAdvancedConfigGet()
  modified.value = false
  nextTick(() => {
    modified.value = false
  })
}
</script>

<style scoped lang="css">
</style>