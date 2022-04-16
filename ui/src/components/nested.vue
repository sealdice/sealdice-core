<template>
  <draggable
    class="dragArea"
    tag="ul"
    :list="tasks"
    :group="{ name: 'g1' }"
    item-key="name"
  >
    <template #item="{ element: el }">
      <li style="padding-right: 1rem;">
        <el-checkbox v-model="el.enable">开启此项</el-checkbox>
        <div>条件: </div>
        <div style="padding-left: 1rem; border-left: .2rem solid orange;">
          <el-select v-model="el.condition.condType">
              <el-option
                v-for="item in [{'label': '文本匹配', value: 'match'}]"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>

          <div v-if="el.condition.condType === 'match'" style="display: flex;">
            <el-select v-model="el.condition.matchType" placeholder="Select">
              <el-option
                v-for="item in [{'label': '精确匹配', value: 'matchExact'}, {'label': '模糊匹配', value: 'matchFuzzy'}, {'label': '正则匹配', value: 'matchRegex'}, {'label': '前缀匹配', value: 'matchPrefix'}, {'label': '后缀匹配', value: 'matchSuffix'}]"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
            <el-input v-model="el.condition.value" />
          </div>
        </div>

        <div>结果：</div>
        <div style="padding-left: 1rem; border-left: .2rem solid skyblue;">
          <div v-for="i in el.results" style="border-left: .1rem solid #008; padding-left: .3rem; margin-bottom: .8rem;">
            <el-select v-model="i.resultType">
              <el-option
                v-for="item in [{'label': '回复', value: 'replyToSender'}, {'label': '私聊回复', value: 'replyPrivate'}, {'label': '群内回复', value: 'replyGroup'}]"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>

            <div v-if="['replyToSender', 'replyPrivate', 'replyGroup'].includes(i.resultType)" style="display: flex;">
              <div style="flex: 1; padding-right: 1rem;">
                <div>回复文本:</div>
                <el-input type="textarea" autosize v-model="i.message"></el-input>
              </div>
              <div>
                <div>延迟
                  <el-tooltip raw-content content="文本将在此延迟后发送，单位秒，可小数。<br />注意随机延迟仍会被加入，如果你希望保证发言顺序，记得考虑这点。">
                    <el-icon><question-filled /></el-icon>
                  </el-tooltip>
                </div>
                <el-input type="number" v-model="i.delay" style="width: 4rem"></el-input>
              </div>
            </div>
          </div>
          <el-button @click="addResult(el.results)">增加</el-button>
        </div>        
        <!-- <nested-draggable :tasks="element.tasks" /> -->
      </li>
    </template>
  </draggable>
</template>

<script lang="ts" setup>
import {
  Location,
  Document,
  Menu as IconMenu,
  Setting,
  CirclePlusFilled,
  CircleClose,
  QuestionFilled,
  BrushFilled
} from '@element-plus/icons-vue'
import draggable from "vuedraggable";

const props = defineProps<{ tasks: Array<any>}>();

const addResult = (i: any) => {
  i.push({"resultType":"replyToSender","delay":0,"message":"说点什么"})
}
</script>

<style scoped>
.dragArea {
  min-height: 50px;
  outline: 1px dashed;
  padding-top: 1rem;
  padding-bottom: 1rem;
}
</style>
