<template>
  <div>
    <div class="img-box">
      <img :src="imgHaibao" />
    </div>
    <div style="text-align: left; line-height: 2rem; font-size: .8rem;">
      <div class="top-item"><span style="font-weight: 600;">当前版本: {{ store.curDice.baseInfo.version }}</span></div>
      <!-- <div class="top-item"><span style="font-weight: 600;">最新版本: {{ store.curDice.baseInfo.version }}</span></div> -->
      <div class="top-item" style="font-family: sans-serif;margin-top: 1rem;">官方网站: <el-link type="primary" src="https://www.sealdice.com">https://www.sealdice.com</el-link></div>
      <div class="top-item" style="font-family: sans-serif;">使用手册: <el-link type="primary" src="https://weizaima.com/manual/">https://weizaima.com/manual/</el-link></div>
      <div class="top-item">
        <div style="margin-top: 1rem; font-family: sans-serif;">
          <div>海洋馆工作人员: </div>
          <div class="hidden-xs-only">木落(fy0@qq.com)、 Yumeno(184023393)、 鹊鹊(1251343516)</div>
          <div class="hidden-sm-and-up">
            <div>木落(fy0@qq.com)</div>
            <div>Yumeno(184023393)</div>
            <div>鹊鹊(1251343516)</div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <div style="text-align: center; font-weight: bold; font-size: 1.3rem; margin-top: 4rem;">感谢 ♪(･ω･)ﾉ</div>
  <div style="text-align: center; font-size: small; margin-bottom: 2rem; font-family: sans-serif;">特别鸣谢参与测试、反馈问题，帮助完善海豹指令的各位！以下列出感谢名单（排名不分先后）</div>
  <div class="about">
    <div></div>
    <div class="subtitle">V1.0版本</div>
    <div>Ariel船长 - 早期测试参与者，协助解决登录流程问题</div>
    <div>Raycel - 早期测试参与者，协助解决登录流程问题</div>
    <div>kuma - 早期测试参与者，海豹的第一次全指令全流程测试</div>
    <div>卟啵 - 早期测试参与者，回报了中文路径和空格路径问题，协助解决了登录流程问题</div>
    <div>蜜瓜包 - 早期测试参与者，默认文档中“怪物之锤查询”的编纂者之一</div>
    <div>月森优姬 - 早期测试参与者，提出了大量各种各样建议和BUG反馈，纠正了一些违背规则书的问题，COC同义词和默认技能点数的编纂者</div>
    <div>清茶 - 在4月7日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令</div>
    <div>脑 - 在4月7日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令</div>
    <div>Greed锦鲤 - 在4月7日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令</div>
    <div>格莱德 - 在4月7日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令</div>

    <div style="margin-top: 2rem;" class="subtitle">参考</div>
    <div>赵喵喵 - ZhaoDice作者，主要指令参考之一</div>
    <div>Dice!核心的开发者们 - 同样的，在骰点格式和输出表现方面进行了参考</div>
    <div>斯塔尼亚 - 塔系核心作者，指令实现过程中部分参考了塔系核心的指令表现</div>
  </div>

  <!-- <div class="tip">
    <p class="title">TIP</p>
    <div>特别注意：内测版意味着尚不完全稳定，配置文件有可能发生不兼容的变化。请谨慎使用。</div>
  </div> -->
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount } from 'vue';
import { useStore } from '~/store';
import imgHaibao from '~/assets/haibao1.png'

const store = useStore()

onBeforeMount(async () => {
  await store.logFetchAndClear()
})

onBeforeUnmount(() => {
  // clearInterval(timerId)
})
</script>

<style scoped lang="scss">
.img-box {
  height: 250px;
  margin-right: 3rem;
  float: left;

  img {
    height: 200px;
  }
}

.about {
  background-color: #fff;
  padding: 2rem;
  line-height: 2rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04)
}

.subtitle {
  margin-bottom: 1rem;
  font-weight: bold;
}
</style>
