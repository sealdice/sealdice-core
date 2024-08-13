<script setup lang="ts">
import {Delete, QuestionFilled} from "@element-plus/icons-vue";
import {breakpointsTailwind} from "@vueuse/core";

const listModel = defineModel();
const data = listModel.value;

const breakpoints = useBreakpoints(breakpointsTailwind)
const notMobile = breakpoints.greater('sm')

const deleteByIndex = (index) => {
  listModel.value.splice(index, 1)
}

</script>

<template>
  <div v-for="(cond, index) in data" :key="index"
       class="mb-3 pl-2 border-l-2 border-orange-500">
    <div class="pb-2 flex justify-between border-b">
      <el-space>
        <el-text>模式</el-text>
        <el-radio-group size="small" v-model="cond.condType">
          <el-radio-button value="textMatch" label="文本匹配"/>
          <el-radio-button value="textLenLimit" label="文本长度"/>
          <el-radio-button value="exprTrue" label="表达式为真"/>
        </el-radio-group>
      </el-space>
      <el-button type="danger" :icon="Delete" size="small" plain
                 @click="deleteByIndex(index)">
        <template #default v-if="notMobile">
          删除条件
        </template>
      </el-button>
    </div>

    <div v-if="cond.condType === 'textMatch'" style="display: flex;" class="mobile-changeline">
      <div style="width: 7rem; margin-right: 0.5rem;">
        <el-text>方式
          <el-tooltip raw-content
                      content="匹配方式一览:<br/>精确匹配: 完全相同时触发。<br/>任意相符: 如aa|bb，则aa或bb都能触发。<br/>包含文本: 包含此文本触发。<br/>不含文本: 不包含此文本触发。<br/>模糊匹配: 文本相似时触发<br/>正则匹配: 正则表达式匹配，语法请自行查阅<br/>前缀匹配: 文本以内容为开头<br/>后缀匹配: 文本以此内容为结尾">
            <el-icon>
              <question-filled/>
            </el-icon>
          </el-tooltip>
        </el-text>
        <el-select size="small" v-model="cond.matchType" placeholder="选择方式">
          <el-option
              v-for="item in [{ 'label': '精确匹配', value: 'matchExact' }, { 'label': '任意相符', value: 'matchMulti' }, { 'label': '包含文本', value: 'matchContains' }, { 'label': '不含文本', value: 'matchNotContains' }, { 'label': '模糊匹配', value: 'matchFuzzy' }, { 'label': '正则匹配', value: 'matchRegex' }, { 'label': '前缀匹配', value: 'matchPrefix' }, { 'label': '后缀匹配', value: 'matchSuffix' }]"
              :key="item.value" :label="item.label" :value="item.value"/>
        </el-select>
      </div>

      <div style="flex: 1">
        <el-text>内容</el-text>
        <el-input v-model="cond.value"/>
      </div>
    </div>

    <div v-else-if="cond.condType === 'exprTrue'" style="display: flex;" class="mobile-changeline">
      <div style="flex: 1">
        <el-text>表达式
          <el-tooltip raw-content
                      content="举例：<br>$t1 == '张三' // 正则匹配的第一个组内容是张三<br>$m个人计数器 >= 10<br>友情提醒，匹配失败时无提示，请先自行在“指令测试”测好">
            <el-icon>
              <question-filled/>
            </el-icon>
          </el-tooltip>
        </el-text>
        <el-input type="textarea" :autosize="{ minRows: 1, maxRows: 10 }" v-model="cond.value"/>
      </div>
    </div>

    <el-space v-else-if="cond.condType === 'textLenLimit'" class="mt-2 flex">
      <el-radio-group size="small" v-model="cond.matchOp">
        <el-radio-button value="ge" label="大于等于"/>
        <el-radio-button value="le" label="小于等于"/>
      </el-radio-group>
      <el-input-number v-model="cond.value" :min="0"/>
    </el-space>
  </div>

</template>

<style scoped lang="css">
@media screen and (max-width: 700px) {
  .mobile-changeline {
    flex-direction: column;
  }
}
</style>