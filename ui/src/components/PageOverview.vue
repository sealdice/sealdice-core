<template>
  <el-form label-width="120px" style="padding-bottom: 3rem">
    <h2>Master管理</h2>
    <el-form-item label="">
      <template #label>
        <div>
          <span>Master解锁码</span>
          <el-tooltip content="用于误操作或被抢占master。执行这条指令可以直接获取master权限并踢掉其他所有人，指令有效期20分钟">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <div>
        <el-button v-if="!isShowUnlockCode" @click="isShowUnlockCode = true">查看</el-button>
        <div v-else>.master unlock {{config.masterUnlockCode}}</div>
      </div>
    </el-form-item>

    <el-form-item label="Master列表">
      <div v-for="k2, index in config.diceMasters" style="width: 100%; margin-bottom: .5rem;">
        <!-- 这里面是单条修改项 -->
        <div style="display: flex;">
          <div>
            <!-- :suffix-icon="Management" -->
            <el-input v-model="config.diceMasters[index]" :autosize="true"></el-input> 
          </div>
          <div style="display: flex; align-items: center; width: 1.3rem; margin-left: 1rem;">
            <el-tooltip :content="index === 0 ? '点击添加项目' : '点击删除你不想要的项'" placement="bottom-start">
              <el-icon>
                <circle-plus-filled v-if="index == 0" @click="addItem(config.diceMasters)" />
                <circle-close v-else @click="removeItem(config.diceMasters, index)" />
              </el-icon>
            </el-tooltip>
          </div>
        </div>
      </div>
    </el-form-item>

    <h2>访问控制</h2>
    <el-form-item label="UI界面地址">
      <template #label>
        <div>
          <span>UI界面地址</span>
          <el-tooltip raw-content content="0.0.0.0:3211 主要用于服务器，代表可以在公网中用你的手机和电脑访问 <br>127.0.0.1:3211 主要用于自己的电脑/手机，只能在当前设备上管理海豹<br>注意：重启骰子后生效！<br>另，想开多个海豹必须修改端口号！">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-autocomplete v-model="config.serveAddress" name="setting" :fetch-suggestions="querySearch">
        <template #default="{ item }">
          <div class="value">{{ item.link }}</div>
        </template>
      </el-autocomplete>
    </el-form-item>

    <el-form-item label="UI界面密码">
      <template #label>
        <div>
          <span>UI界面密码</span>
          <el-tooltip content="公网用户一定要加，登录后会自动记住一段时间！">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>

      <el-input type="password" show-password v-model="config.uiPassword" style="width: auto;" />
    </el-form-item>

    <h2>其他</h2>
    <el-form-item label="QQ回复延迟(秒)">
      <el-input v-model="config.messageDelayRangeStart" type="number" style="width: 6rem;" />
      <span style="margin: 0 1rem">-</span>
      <el-input v-model="config.messageDelayRangeEnd" type="number" style="width: 6rem;" />
    </el-form-item>

    <el-form-item label="日志仅记录指令">
        <el-checkbox label="在群聊中" v-model="config.onlyLogCommandInGroup"/>
        <el-checkbox label="在私聊中" v-model="config.onlyLogCommandInPrivate"/>
    </el-form-item>

    <el-form-item label="指令前缀">
      <template #label>
        <div>
          <span>指令前缀</span>
          <el-tooltip content="举例：添加!作为指令前缀，运行 !r 可以骰点">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>

      <div v-for="k2, index in config.commandPrefix" style="width: 100%; margin-bottom: .5rem;">
        <!-- 这里面是单条修改项 -->
        <div style="display: flex;">
          <div>
            <!-- :suffix-icon="Management" -->
            <el-input v-model="config.commandPrefix[index]" :autosize="true"></el-input> 
          </div>
          <div style="display: flex; align-items: center; width: 1.3rem; margin-left: 1rem;">
            <el-tooltip :content="index === 0 ? '点击添加项目' : '点击删除你不想要的项目'" placement="bottom-start">
              <el-icon>
                <circle-plus-filled v-if="index == 0" @click="addItem(config.commandPrefix)" />
                <circle-close v-else @click="removeItem(config.commandPrefix, index)" />
              </el-icon>
            </el-tooltip>
          </div>
        </div>
      </div>
    </el-form-item>

    <el-form-item label="" style="margin-top: 3rem;" v-if="modified">
      <el-button type="danger" @click="submitGiveup">放弃改动</el-button>
      <el-button type="success" @click="submit">保存设置</el-button>
    </el-form-item>
  </el-form>

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
import { cloneDeep } from 'lodash-es';
import { computed, nextTick, onBeforeMount, onBeforeUnmount, ref, watch } from 'vue';
import { useStore } from '~/store';
import { objDiff, passwordHash } from '~/utils';
const store = useStore()

const config = ref<any>({})

const isShowUnlockCode = ref(false)
const modified = ref(false)

const addItem = (k: any) => {
  k.push('')
}

watch(() => config, (newValue, oldValue) => { //直接监听
  modified.value = true
}, {
  deep: true
});

const removeItem = (v: any[], index: number) => {
  v.splice(index, 1)
}

onBeforeMount(async () => {
  await store.diceConfigGet()
  config.value = cloneDeep(store.curDice.config)
  nextTick(() => {
    modified.value = false
  })
})

const submitGiveup = async () => {
  config.value = cloneDeep(store.curDice.config)
  modified.value = false
  nextTick(() => {
    modified.value = false
  })
}

onBeforeUnmount(() => {
  // clearInterval(timerId)
})

const submit = async () => {
  const mod = objDiff(config.value, store.curDice.config)
  if (mod.uiPassword) {
    mod.uiPassword = await passwordHash(store.salt, mod.uiPassword)
  }
  await store.diceConfigSet(mod)
  submitGiveup()
}

interface LinkItem {
  value: string
  link: string
}

const state = ref('')
const links = ref<LinkItem[]>([
  { link: '外网: 0.0.0.0:3211', value: '0.0.0.0:3211' },
  { link: '内网: 127.0.0.1:3211', value: '127.0.0.1:3211' },
])

const querySearch = (queryString: string, cb: any) => {
  const results = queryString
    ? links.value.filter(createFilter(queryString))
    : links.value
  // call callback function to return suggestion objects
  cb(results)
}
const createFilter = (queryString: string) => {
  return (restaurant: LinkItem) => {
    return (
      restaurant.value.toLowerCase().indexOf(queryString.toLowerCase()) === 0
    )
  }
}
</script>

<style scoped lang="scss">
.about {
  background-color: #fff;
  padding: 2rem;
  line-height: 2rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04)
}

.top-item {
  flex: 1 0 50%;
  flex-grow: 0;
}
</style>
