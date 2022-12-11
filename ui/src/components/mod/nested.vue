<template>
  <draggable
    class="dragArea"
    tag="div"
    :list="tasks"
    handle=".handle"
    :group="{ name: 'g1' }"
    item-key="name"
  >
    <template #item="{ element: el, index }">
      <li style="padding-right: .5rem; list-style: none; margin-bottom: 0.5rem;">
        <div style="display: flex; justify-content: space-between;">
          <el-checkbox v-model="el.enable">开启</el-checkbox>
          <div style="display: flex; align-items: center;">
            <el-icon v-if="!el.notCollapse" class="handle" style="padding: 0.2rem 0.7rem; font-size: 1.3rem; color: #999"><rank /></el-icon>
            <i class="fa fa-align-justify handle"></i>
            <el-button @click="el.notCollapse = !el.notCollapse">{{ el.notCollapse ? '收缩' : '展开'}}</el-button>
            <el-button @click="deleteItem(index)">删除</el-button>
          </div>
        </div>

        <template v-if="!el.notCollapse">
          <div style="padding-left: 1rem; border-left: .2rem solid orange;">
            <div v-for="(cond, index2) in (el.conditions || [])">
              <div v-if="cond.condType === 'textMatch'" style="display: flex;" class="mobile-changeline">
                文本匹配: {{ cond.value }}
              </div>
              <div v-else-if="cond.condType === 'exprTrue'" style="display: flex;" class="mobile-changeline">
                <div style="flex: 1">
                  表达式: {{ cond.value }}
                </div>
              </div>
              <div v-else-if="cond.condType === 'textLenLimit'" style="display: flex;" class="mobile-changeline">
                <div style="flex: 1">
                  长度: {{ cond.matchOp === 'ge' ? '大于等于' : '' }}{{ cond.matchOp === 'le' ? '小于等于' : '' }} {{ cond.value }}
                </div>
              </div>
            </div>
          </div>
        </template>
        <template v-else>
          <div>条件(需同时满足，即and): </div>
          <div style="padding-left: 1rem; border-left: .2rem solid orange;">
            <div v-for="(cond, index2) in (el.conditions || [])" style="border-left: .1rem solid #008; padding-left: .3rem; margin-bottom: .8rem;">
              <div style="display: flex; justify-content: space-between;">
                <el-select v-model="cond.condType">
                  <el-option
                    v-for="item in [{'label': '文本匹配', value: 'textMatch'}, {'label': '文本长度', value: 'textLenLimit'}, {'label': '表达式为真', value: 'exprTrue'}]"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-button @click="deleteAnyItem(el.conditions, index2)">删除条件</el-button>
              </div>

              <div v-if="cond.condType === 'textMatch'" style="display: flex;" class="mobile-changeline">
                <div style="width: 7rem; margin-right: 0.5rem;">
                  <div>方式:
                    <el-tooltip raw-content content="匹配方式一览:<br/>精确匹配: 完全相同时触发。<br/>包含文本: 包含此文本触发。<br/>不含文本: 不包含此文本触发。<br/>模糊匹配: 文本相似时触发<br/>正则匹配: 正则表达式匹配，语法请自行查阅<br/>前缀匹配: 文本以内容为开头<br/>后缀匹配: 文本以此内容为结尾">
                      <el-icon><question-filled /></el-icon>
                    </el-tooltip>
                  </div>
                  <el-select v-model="cond.matchType" placeholder="Select">
                    <el-option
                      v-for="item in [{'label': '精确匹配', value: 'matchExact'}, {'label': '包含文本', value: 'matchContains'}, {'label': '不含文本', value: 'matchNotContains'}, {'label': '模糊匹配', value: 'matchFuzzy'}, {'label': '正则匹配', value: 'matchRegex'}, {'label': '前缀匹配', value: 'matchPrefix'}, {'label': '后缀匹配', value: 'matchSuffix'}]"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </el-select>
                </div>

                <div style="flex: 1">
                  <div>内容:</div>
                  <el-input v-model="cond.value" />
                </div>
              </div>

              <div v-else-if="cond.condType === 'exprTrue'" style="display: flex;" class="mobile-changeline">
                <div style="flex: 1">
                  <div>表达式:
                    <el-tooltip raw-content content="举例：<br>$t1 == '张三' // 正则匹配的第一个组内容是张三<br>$m个人计数器 >= 10<br>友情提醒，匹配失败时无提示，请先自行在“指令测试”测好">
                      <el-icon><question-filled /></el-icon>
                    </el-tooltip>
                  </div>
                  <el-input type="textarea" :autosize="{ minRows: 1, maxRows: 10 }" v-model="cond.value" />
                </div>
              </div>
        
              <div v-else-if="cond.condType === 'textLenLimit'" style="display: flex;" class="mobile-changeline">
                <div style="width: 7rem; margin-right: 0.5rem;">
                  <div>方式:
                    <!-- <el-tooltip raw-content content="匹配方式一览:<br/>精确匹配: 完全相同时触发。<br/>包含文本: 包含此文本触发。<br/>不含文本: 不包含此文本触发。<br/>模糊匹配: 文本相似时触发<br/>正则匹配: 正则表达式匹配，语法请自行查阅<br/>前缀匹配: 文本以内容为开头<br/>后缀匹配: 文本以此内容为结尾">
                      <el-icon><question-filled /></el-icon>
                    </el-tooltip> -->
                  </div>
                  <el-select v-model="cond.matchOp" placeholder="Select">
                    <el-option
                      v-for="item in [{'label': '大于等于', value: 'ge'}, {'label': '小于等于', value: 'le'}]"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </el-select>
                </div>

                <div style="flex: 1">
                  <div>文本字数:</div>
                  <el-input v-model="cond.value" type="number" />
                </div>
              </div>
            </div>

            <el-button @click="addCond(el.conditions)">增加</el-button>
          </div>

          <div>结果(顺序执行)：</div>
          <div style="padding-left: 1rem; border-left: .2rem solid skyblue;">
            <div v-for="(i, index) in (el.results || [])" style="border-left: .1rem solid #008; padding-left: .3rem; margin-bottom: .8rem;">
              <div style="display: flex; justify-content: space-between;">
                <el-select v-model="i.resultType">
                  <el-option
                    v-for="item in [{'label': '回复', value: 'replyToSender'}, {'label': '私聊回复', value: 'replyPrivate'}, {'label': '群内回复', value: 'replyGroup'}]"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-button @click="deleteAnyItem(el.results, index)">删除结果</el-button>
              </div>

              <div v-if="['replyToSender', 'replyPrivate', 'replyGroup'].includes(i.resultType)">
                <div style="display: flex; justify-content: space-between;" class="mobile-changeline">
                  <div style="display: flex; align-items: center;">
                    <div>回复文本(随机选择):</div>
                  </div>
                  <div>
                    <div style="display: inline-block">延迟
                      <el-tooltip raw-content content="文本将在此延迟后发送，单位秒，可小数。<br />注意随机延迟仍会被加入，如果你希望保证发言顺序，记得考虑这点。">
                        <el-icon><question-filled /></el-icon>
                      </el-tooltip>
                    </div>
                    <el-input type="number" v-model="i.delay" style="width: 4rem"></el-input>
                  </div>
                </div>

                <div v-for="k2, index in i.message" style="width: 100%; margin-bottom: .5rem;">
                  <!-- 这里面是单条修改项 -->
                  <div style="display: flex;">
                    <div style="display: flex; align-items: center; width: 1.3rem; margin-left: .2rem;">
                      <el-tooltip :content="index === 0 ? '点击添加一个回复语，海豹将会随机抽取一个回复' : '点击删除你不想要的回复语'" placement="bottom-start">
                        <el-icon>
                          <circle-plus-filled v-if="index == 0" @click="addItem(i.message)" />
                          <circle-close v-else @click="removeItem(i.message, index)" />
                        </el-icon>
                      </el-tooltip>
                    </div>
                    <div style="flex:1">
                      <!-- :suffix-icon="Management" -->
                      <el-input type="textarea" class="reply-text" autosize v-model="k2[0]"></el-input> 
                    </div>
                  </div>
                </div>
                  <!-- <el-input type="textarea" autosize v-model="i.message"></el-input> -->
              </div>
            </div>
            <el-button @click="addResult(el.results)">增加</el-button>
          </div>        
          <!-- <nested-draggable :tasks="element.tasks" /> -->
        </template>
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
  Rank,
  BrushFilled
} from '@element-plus/icons-vue'
import draggable from "vuedraggable";
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps<{ tasks: Array<any>}>();

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
  i.push({"condType":"textMatch","matchType":"matchExact","value":"要匹配的文本"})
}

const addResult = (i: any) => {
  i.push({"resultType":"replyToSender","delay":0,"message":[ ["说点什么", 1] ]})
}

const addItem = (k: any) => {
  k.push(['怎么辉石呢', 1])
}

const removeItem = (v: any[], index: number | any) => {
  v.splice(index, 1)
}
</script>

<style scoped>
.dragArea {
  min-height: 50px;
  /* outline: 1px dashed; */
  padding-top: 1rem;
  padding-bottom: 1rem;
}

@media screen and (max-width: 700px) {
  .mobile-changeline {
    flex-direction: column;
  }
}
</style>
