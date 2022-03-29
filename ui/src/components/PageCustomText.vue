<script setup lang="ts">
import { ref, reactive, onBeforeMount, onDeactivated, watch } from "vue";
import { ElMessage, ElMessageBox } from 'element-plus'
import { useStore } from '~/store'
import { Management } from '@element-plus/icons-vue'

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
import { cloneDeep } from "lodash-es";

const store = useStore()
const props = defineProps<{ category: string }>();

watch(() => props.category, (newValue, oldValue) => { //直接监听
  modified.value = false
});

const addItem = (k: any) => {
  store.curDice.customTexts[props.category][k].push(['', 1 as any])
  modified.value = true
}

const doChanged = (category: string, keyName: string) => {
  modified.value = true
  const helpInfo = store.curDice.customTextsHelpInfo[category][keyName]
  helpInfo.modified = true
}

const removeItem = (v: any[], index: number) => {
  v.splice(index, 1)
  modified.value = true
}

const save = async () => {
  modified.value = false
  await store.customTextSave(props.category)
  await store.getCustomText()
  ElMessage.success('已保存')
}

const resetValue = async (category: string, keyName: string) => {
  const helpInfo = store.curDice.customTextsHelpInfo[category][keyName]
  store.curDice.customTexts[category][keyName] = cloneDeep(helpInfo.origin)
  helpInfo.modified = false
  modified.value = true
}

const askResetValue = async (category: string, keyName: string) => {
  ElMessageBox.confirm(
    '重置这条文本回默认状态，确定吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    resetValue(category, keyName)
    ElMessage({
      type: 'success',
      message: '成功!',
    })
  })
}

const modified = ref(false)

onBeforeMount(async () => {
  modified.value = false
})
</script>

<template>
  <el-affix :offset="60" v-if="modified">
    <div class="tip">
      <!-- <p class="title">TIP</p> -->
      <div style="display: flex; justify-content: space-between; align-content: center; align-items: center">
        <span>内容已修改，不要忘记保存</span>
        <el-button class="button" type="primary" @click="save" :disabled="!modified">点我保存</el-button>
      </div>
    </div>
  </el-affix>

  <div style="margin-bottom: 2rem;">
    <el-collapse>
        <el-collapse-item name="1">
          <template #title>
            <div style="padding: 0 1rem">查看帮助<el-icon><question-filled /></el-icon></div>
          </template>

          <div style="padding: 0 1rem;">
            <div>此处可以对骰子返回的文本进行修改。最终返回的文本将为多个条目中随机抽取的一个。</div>
            <div>随机文本:默认一种显示结果，如果需要多种反馈结果，使用＋添加条目，使用-删除条目</div>
            <!-- 权重选择:默认1，权重—致则没有优先级。数字越小，优先级越高 -->
            <!-- <div>文件备份:已修改的指令统一存在于路径/路径1/路径2/文件名，如有需要替换文件即可</div> -->
            <div>遇到有此标记(<el-icon><brush-filled /></el-icon>)的条目，说明和默认值不同，是一个自定义条目</div>
            <div style="margin-top: 1rem;">文本下方的<el-tag>标签</el-tag>代表了被默认文本所使用的特殊变量，你可以使用 {变量名} 来插入他们，例如 {$t判定值} </div>
            <div>除此之外，这些变量可以在所有文本中使用: 
              <el-space wrap>
                <el-tag v-for="i in ['$t玩家', '$tQQ昵称', '$t个人骰子面数', '$tQQ', '$t骰子帐号', '$t骰子昵称', '$t群号', '$t群名']">{{i}}</el-tag>
              </el-space>
            </div>
            <div>
              <span>以及，所有的自定义文本都可以嵌套使用，例如：</span>
              <div>
                <b>这里是{核心:骰子名字}，我是一个示例</b>
              </div>
              <div>默认会被解析为:</div>
              <div>
                <b>这里是海豹bot，我是一个示例</b>
              </div>
              <div>注意！千万不要递归嵌套，会发生很糟糕的事情</div>
            </div>

            <div style="margin-top: 1rem;">
              <div>此外，支持插入图片，将图片放在骰子的适当目录，再写这样一句话即可:</div>
              <div><b>[图:data/images/sealdice.png]</b></div>
              <div>可以参考 核心:骰子进群 词条</div>
              <div>同样的，可以使用CQ码插入图片和其他内容，关于CQ码，请参阅onebot项目文档</div>
            </div>

          </div>
        </el-collapse-item>
    </el-collapse>
  </div>

  <el-row :gutter="20">
    <el-col :xs="24" :span="12" v-for="v, k in reactive(store.curDice.customTexts[category])">
      <el-form ref="form" label-width="auto" label-position="top">
        <el-form-item>
          <template #label>
            <div>
              {{ k.toString() }}

              <template v-if="store.curDice.customTextsHelpInfo[category][k.toString()].modified">
                <el-tooltip content="重置为初始值" placement="bottom-end">
                  <el-icon style="float: right;" @click="askResetValue(category, k.toString())">
                    <brush-filled />
                  </el-icon>
                </el-tooltip>
              </template>
            </div>
          </template>
          
          <div v-for="k2, index in v" style="width: 100%; margin-bottom: .5rem;">
            <!-- 这里面是单条修改项 -->
            <el-row>
              <el-col :span="2">
                <el-tooltip :content="index === 0 ? '点击添加一个回复语，SealDice将会随机抽取一个回复' : '点击删除你不想要的回复语'" placement="bottom-start">
                  <el-icon>
                    <circle-plus-filled v-if="index == 0" @click="addItem(k)" />
                    <circle-close v-else @click="removeItem(v, index)" />
                  </el-icon>
                </el-tooltip>
              </el-col>
              <el-col :span="22">
                <!-- :suffix-icon="Management" -->
                <el-input v-model="k2[0]" :autosize="true" @change="doChanged(category, k.toString())"></el-input> 
              </el-col>
            </el-row>
          </div>
          <div>
            <el-tag v-for="i in store.curDice.customTextsHelpInfo[category][k.toString()].vars">{{i}}</el-tag>
            <!-- {{ store.curDice.customTextsHelpInfo[category][k.toString()] }} -->
          </div>
        </el-form-item>
      </el-form>
    </el-col>
  </el-row>
</template>
