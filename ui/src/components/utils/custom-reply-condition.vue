<script setup lang="ts">
import {Delete, QuestionFilled} from "@element-plus/icons-vue";

const condModel = defineModel();
const cond: any = condModel.value

const props = defineProps<{
  key: number
}>();

const emit = defineEmits(['delete']);

const deleteSelf = () => {
  emit('delete', props.key)
}

</script>

<template>
  <div style="border-left: .1rem solid #008; padding-left: .3rem; margin-bottom: .8rem;">
    <div style="display: flex; justify-content: space-between;">
      <el-select v-model="cond.condType">
        <el-option
            v-for="item in [{ 'label': '文本匹配', value: 'textMatch' }, { 'label': '文本长度', value: 'textLenLimit' }, { 'label': '表达式为真', value: 'exprTrue' }]"
            :key="item.value" :label="item.label" :value="item.value"/>
      </el-select>
      <el-button type="danger" :icon="Delete" size="small" plain
                 @click="deleteSelf">删除条件
      </el-button>
    </div>

    <div v-if="cond.condType === 'textMatch'" style="display: flex;" class="mobile-changeline">
      <div style="width: 7rem; margin-right: 0.5rem;">
        <div>方式：
          <el-tooltip raw-content
                      content="匹配方式一览:<br/>精确匹配: 完全相同时触发。<br/>任意相符: 如aa|bb，则aa或bb都能触发。<br/>包含文本: 包含此文本触发。<br/>不含文本: 不包含此文本触发。<br/>模糊匹配: 文本相似时触发<br/>正则匹配: 正则表达式匹配，语法请自行查阅<br/>前缀匹配: 文本以内容为开头<br/>后缀匹配: 文本以此内容为结尾">
            <el-icon>
              <question-filled/>
            </el-icon>
          </el-tooltip>
        </div>
        <el-select v-model="cond.matchType" placeholder="Select">
          <el-option
              v-for="item in [{ 'label': '精确匹配', value: 'matchExact' }, { 'label': '任意相符', value: 'matchMulti' }, { 'label': '包含文本', value: 'matchContains' }, { 'label': '不含文本', value: 'matchNotContains' }, { 'label': '模糊匹配', value: 'matchFuzzy' }, { 'label': '正则匹配', value: 'matchRegex' }, { 'label': '前缀匹配', value: 'matchPrefix' }, { 'label': '后缀匹配', value: 'matchSuffix' }]"
              :key="item.value" :label="item.label" :value="item.value"/>
        </el-select>
      </div>

      <div style="flex: 1">
        <div>内容：</div>
        <el-input v-model="cond.value"/>
      </div>
    </div>

    <div v-else-if="cond.condType === 'exprTrue'" style="display: flex;" class="mobile-changeline">
      <div style="flex: 1">
        <div>表达式：
          <el-tooltip raw-content
                      content="举例：<br>$t1 == '张三' // 正则匹配的第一个组内容是张三<br>$m个人计数器 >= 10<br>友情提醒，匹配失败时无提示，请先自行在“指令测试”测好">
            <el-icon>
              <question-filled/>
            </el-icon>
          </el-tooltip>
        </div>
        <el-input type="textarea" :autosize="{ minRows: 1, maxRows: 10 }" v-model="cond.value"/>
      </div>
    </div>

    <div v-else-if="cond.condType === 'textLenLimit'" style="display: flex;" class="mobile-changeline">
      <div style="width: 7rem; margin-right: 0.5rem;">
        <div>方式：</div>
        <el-select v-model="cond.matchOp" placeholder="Select">
          <el-option v-for="item in [{ 'label': '大于等于', value: 'ge' }, { 'label': '小于等于', value: 'le' }]"
                     :key="item.value" :label="item.label" :value="item.value"/>
        </el-select>
      </div>

      <div style="flex: 1">
        <div>文本字数：</div>
        <el-input v-model="cond.value" type="number"/>
      </div>
    </div>
  </div>

</template>

<style scoped lang="scss">
@media screen and (max-width: 700px) {
  .mobile-changeline {
    flex-direction: column;
  }
}
</style>