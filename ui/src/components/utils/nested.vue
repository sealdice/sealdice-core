<template>
  <draggable class="dragArea" tag="div" :list="tasks" handle=".handle" :group="{ name: 'g1' }" item-key="name">
    <template #item="{ element: el, index }">
      <li class="reply-item list-none mb-2">
        <foldable-card type="div" :default-fold="true" compact>
          <template #title>
            <el-checkbox v-model="el.enable">开启</el-checkbox>
          </template>
          <template #title-extra>
            <el-space size="large" alignment="center">
              <el-icon class="handle">
                <rank />
              </el-icon>
              <el-button :icon="Delete" plain type="danger" size="small" @click="deleteItem(index)">删除</el-button>
            </el-space>
          </template>

          <template #unfolded-extra>
            <div class="pl-4 border-l-4 border-orange-500">
              <div v-for="(cond, index2) in (el.conditions || [])" :key="index2">
                <el-text size="large" v-if="cond.condType === 'textMatch'" style="display: flex;" class="mobile-changeline">
                  文本匹配：{{ cond.value }}
                </el-text>
                <el-text size="large" v-else-if="cond.condType === 'exprTrue'" style="display: flex;" class="mobile-changeline">
                  <div style="flex: 1">
                    表达式：{{ cond.value }}
                  </div>
                </el-text>
                <el-text size="large" v-else-if="cond.condType === 'textLenLimit'" style="display: flex;" class="mobile-changeline">
                  <div style="flex: 1">
                    长度：{{ cond.matchOp === 'ge' ? '大于等于' : '' }}{{ cond.matchOp === 'le' ? '小于等于' : '' }} {{ cond.value }}
                  </div>
                </el-text>
              </div>
            </div>
          </template>

          <el-text class="block mb-2" size="large">条件（需同时满足，即 and）</el-text>
          <div class="pl-4 border-l-4 border-orange-500">
            <custom-reply-condition v-for="(_, index2) in (el.conditions || [])" :key="index2"
                                    v-model="el.conditions[index2]" @delete="deleteAnyItem(el.conditions, index2)"/>

            <el-button type="success" size="small" :icon="Plus" @click="addCond(el.conditions)">增加</el-button>
          </div>

          <el-text class="block my-2" size="large">结果（顺序执行）</el-text>
          <div class="pl-4 border-l-4 border-blue-500">
            <div v-for="(i, index) in (el.results || [])" :key="index"
                 class="mb-3 pl-2 border-l-2 border-blue-500">
              <div style="display: flex; justify-content: space-between;">
                <el-space>
                  <el-text>模式</el-text>
                  <el-radio-group size="small" v-model="i.resultType">
                    <el-radio-button value="replyToSender" label="回复"/>
                    <el-radio-button value="replyPrivate" label="私聊回复"/>
                    <el-radio-button value="replyGroup" label="群内回复"/>
                  </el-radio-group>
                </el-space>

                <el-button type="danger" :icon="Delete" size="small" plain
                           @click="deleteAnyItem(el.results, index)">
                  <template #default v-if="notMobile">
                    删除结果
                  </template>
                </el-button>
              </div>

              <div v-if="['replyToSender', 'replyPrivate', 'replyGroup'].includes(i.resultType)">
                <div class="flex justify-between my-2 mobile-changeline">
                  <div style="display: flex; align-items: center;">
                    <el-text>回复文本（随机选择）</el-text>
                  </div>
                  <el-space>
                    <el-text>
                      延迟
                      <el-tooltip raw-content content="文本将在此延迟后发送，单位秒，可小数。<br />注意随机延迟仍会被加入，如果你希望保证发言顺序，记得考虑这点。">
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip>
                    </el-text>
                    <el-input size="small" type="number" v-model="i.delay" style="width: 4rem"></el-input>
                  </el-space>
                </div>

                <div v-for="(k2, index) in i.message" :key="index" class="w-full my-2">
                  <!-- 这里面是单条修改项 -->
                  <div style="display: flex;">
                    <div style="display: flex; align-items: center; width: 1.3rem; margin-left: .2rem;">
                      <el-tooltip :content="index === 0 ? '点击添加一个回复语，海豹将会随机抽取一个回复' : '点击删除你不想要的回复语'"
                                  placement="bottom-start">
                        <el-icon>
                          <circle-plus-filled v-if="index == 0" @click="addItem(i.message)" />
                          <circle-close v-else @click="removeItem(i.message, index)" />
                        </el-icon>
                      </el-tooltip>
                    </div>
                    <div style="flex:1">
                      <el-input type="textarea" class="reply-text" autosize v-model="k2[0]"></el-input>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <el-button type="success" size="small" :icon="Plus" @click="addResult(el.results)">增加</el-button>
          </div>
        </foldable-card>
      </li>
    </template>
  </draggable>
</template>

<script lang="ts" setup>
import {
  CirclePlusFilled,
  CircleClose,
  QuestionFilled,
  Rank,
  Delete,
  Plus
} from '@element-plus/icons-vue'
import draggable from "vuedraggable";
import { ElMessageBox } from 'element-plus'
import CustomReplyCondition from "~/components/utils/custom-reply-condition.vue";
import {breakpointsTailwind, useBreakpoints} from "@vueuse/core";

const props = defineProps<{ tasks: Array<any> }>();

const breakpoints = useBreakpoints(breakpointsTailwind)
const notMobile = breakpoints.greater('sm')

const deleteItem = (index: number) => {
  ElMessageBox.confirm(
    '确认删除此项？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    props.tasks.splice(index, 1);
  })
}

const deleteAnyItem = (lst: any[], index: number) => {
  lst.splice(index, 1);
}

const addCond = (i: any) => {
  i.push({ "condType": "textMatch", "matchType": "matchExact", "value": "要匹配的文本" })
}

const addResult = (i: any) => {
  i.push({ "resultType": "replyToSender", "delay": 0, "message": [["说点什么", 1]] })
}

const addItem = (k: any) => {
  k.push(['怎么辉石呢', 1])
}

const removeItem = (v: any[], index: number | any) => {
  v.splice(index, 1)
}
</script>

<style scoped lang="scss">
.dragArea {
  min-height: 50px;
  /* outline: 1px dashed; */
  padding-top: 1rem;
  padding-bottom: 1rem;

  .reply-item:not(:last-child) {
    border-bottom: 1px solid var(--el-border-color);
    padding-bottom: 1rem;
  }
}

@media screen and (max-width: 700px) {
  .mobile-changeline {
    flex-direction: column;
  }
}
</style>
