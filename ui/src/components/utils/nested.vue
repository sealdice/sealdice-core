<template>
  <draggable class="dragArea" tag="div" :list="tasks" handle=".handle" :group="{ name: 'g1' }" item-key="name">
    <template #item="{ element: el, index }">
      <li class="reply-item" style="padding-right: .5rem; list-style: none; margin-bottom: 0.5rem;">
        <div style="display: flex; justify-content: space-between;">
          <el-checkbox v-model="el.enable">开启</el-checkbox>
          <div style="display: flex; align-items: center;">
            <el-icon v-if="!el.notCollapse" class="handle" style="padding: 0.2rem 0.7rem; font-size: 1.3rem; color: #999">
              <rank />
            </el-icon>
            <i class="fa fa-align-justify handle"></i>
            <el-button size="small" plain @click="el.notCollapse = !el.notCollapse">
              {{ el.notCollapse ? '收缩' : '展开' }}
            </el-button>
            <el-button :icon="Delete" plain type="danger" size="small" @click="deleteItem(index)">删除</el-button>
          </div>
        </div>

        <template v-if="!el.notCollapse">
          <div style="padding-left: 1rem; border-left: .2rem solid orange;">
            <div v-for="(cond, index2) in (el.conditions || [])" :key="index2">
              <div v-if="cond.condType === 'textMatch'" style="display: flex;" class="mobile-changeline">
                文本匹配: {{ cond.value }}
              </div>
              <div v-else-if="cond.condType === 'exprTrue'" style="display: flex;" class="mobile-changeline">
                <div style="flex: 1">
                  表达式：{{ cond.value }}
                </div>
              </div>
              <div v-else-if="cond.condType === 'textLenLimit'" style="display: flex;" class="mobile-changeline">
                <div style="flex: 1">
                  长度：{{ cond.matchOp === 'ge' ? '大于等于' : '' }}{{ cond.matchOp === 'le' ? '小于等于' : '' }} {{ cond.value }}
                </div>
              </div>
            </div>
          </div>
        </template>
        <template v-else>
          <div>条件（需同时满足，即and）：</div>
          <div style="padding-left: 1rem; border-left: .2rem solid orange;">
            <custom-reply-condition v-for="(_, index2) in (el.conditions || [])" :key="index2"
                                    v-model="el.conditions[index2]" @delete="deleteAnyItem(el.conditions, index2)"/>

            <el-button type="success" size="small" :icon="Plus" @click="addCond(el.conditions)">增加</el-button>
          </div>

          <div>结果（顺序执行）：</div>
          <div style="padding-left: 1rem; border-left: .2rem solid skyblue;">
            <div v-for="(i, index) in (el.results || [])" :key="index"
              style="border-left: .1rem solid #008; padding-left: .3rem; margin-bottom: .8rem;">
              <div style="display: flex; justify-content: space-between;">
                <el-select v-model="i.resultType">
                  <el-option
                    v-for="item in [{ 'label': '回复', value: 'replyToSender' }, { 'label': '私聊回复', value: 'replyPrivate' }, { 'label': '群内回复', value: 'replyGroup' }]"
                    :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
                <el-button type="danger" :icon="Delete" size="small" plain
                  @click="deleteAnyItem(el.results, index)">删除结果</el-button>
              </div>

              <div v-if="['replyToSender', 'replyPrivate', 'replyGroup'].includes(i.resultType)">
                <div style="display: flex; justify-content: space-between;" class="mobile-changeline">
                  <div style="display: flex; align-items: center;">
                    <div>回复文本（随机选择）：</div>
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

                <div v-for="k2, index in i.message" :key="index" style="width: 100%; margin-bottom: .5rem;">
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
        </template>
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

const props = defineProps<{ tasks: Array<any> }>();

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
