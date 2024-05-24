<template>
  <el-space style="margin-left: 1rem; display: flex; justify-content: space-between; align-items: center">
    <el-text v-if="changed" type="primary" size="large">
      <el-icon>
        <QuestionFilled/>
      </el-icon>
      变更如下：
    </el-text>
    <el-text v-else type="info" size="large">
      <el-icon>
        <InfoFilled/>
      </el-icon>
      无变更
    </el-text>
    <el-space v-if="changed" direction="vertical" alignment="center" wrap>
      <el-switch v-model="split" active-text="双列" inactive-text="单列"/>
      <el-checkbox v-model="folding">折叠无变更</el-checkbox>
    </el-space>
  </el-space>
  <div v-show="split" style="display: flex; justify-content: space-around; align-items: center">
    <h3 style="padding-left: 2rem">原内容</h3>
    <el-icon>
      <ArrowRightBold/>
    </el-icon>
    <h3 style="padding-right: 2rem">新内容</h3>
  </div>
  <VueDiff
      v-if="changed"
      :mode="mode"
      theme="light"
      :language="props.lang"
      :folding="folding"
      :prev="props.old" :current="props.new"/>
</template>

<script lang="ts" setup>
import {
  InfoFilled,
  QuestionFilled,
  ArrowRightBold,
} from '@element-plus/icons-vue'
import {computed, ref} from "vue";

interface Props {
  old: string,
  new: string
  lang?: string,
}

const props = withDefaults(defineProps<Props>(), {
  lang: 'text',
  old: '',
  new: '',
})

const changed = computed(() => !(props.old === props.new))
const mode = computed(() => split.value ? 'split' : 'unified')
const split = ref<boolean>(false)
const folding = ref<boolean>(false)
</script>