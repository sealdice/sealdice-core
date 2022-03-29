<template>
  <div style="display: flex; flex-wrap: wrap; text-align: left; line-height: 2rem;">
    <div class="top-item"><span style="font-weight: 600;">当前版本: </span>{{ store.curDice.baseInfo.version }}</div>
    <div class="top-item"><span style="font-weight: 600;">官网: </span>www.sealdice.com(建设中)</div>
    <div class="top-item">
      <div style="font-weight: 600;">海洋馆工作人员: </div>
      <div>木落(fy0@qq.com)、Yumeno(184023393)、鹊鹊(1251343516)</div>
    </div>
  </div>

  <h3 style="text-align: center;">鸣谢</h3>
  <div class="about">
    <div>斯塔尼亚 - 塔系核心作者，指令实现过程中大量参考了塔系核心的指令表现</div>
    <div>Dice!核心的开发者们 - 同样的，在骰点格式和输出表现方面进行了参考</div>
    <div>赵喵喵 - ZhaoDice作者，指令参考之一</div>
    <div style="margin-top: 1rem;">感谢早期测试的参与者们</div>
    <div>V1.0版本</div>
    <div>Ariel船长</div>
    <div>Raycel</div>
    <div>kuma</div>
    <div>卟啵</div>
    <div>月森优姬</div>
  </div>

  <div class="tip">
    <p class="title">TIP</p>
    <div>特别注意：内测版意味着尚不完全稳定，配置文件有可能发生不兼容的变化。请谨慎使用。</div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount } from 'vue';
import { useStore } from '~/store';
const store = useStore()

onBeforeMount(async () => {
  await store.logFetchAndClear()
})

onBeforeUnmount(() => {
  // clearInterval(timerId)
})
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
