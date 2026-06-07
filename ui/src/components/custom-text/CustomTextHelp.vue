<template>
  <TipBox type="info">
    <n-collapse>
      <n-collapse-item name="1">
        <template #header>
          <n-text tag="strong">查看帮助</n-text>
        </template>

        <n-text tag="p">
          <div>此处可以对骰子返回的文本进行修改。最终返回的文本将为多个条目中随机抽取的一个。</div>
          <div>
            随机文本：默认一种显示结果，如果需要多种反馈结果，使用＋添加条目，使用 - 删除条目
          </div>
          <div>
            遇到有此标记 (<n-icon><i-carbon-paint-brush /></n-icon>)
            的条目，说明和默认值不同，是一个自定义条目
          </div>
          <div class="help-block">
            文本下方的
            <n-tag size="small" :bordered="false">标签</n-tag>
            代表了被默认文本所使用的特殊变量，你可以使用 {变量名} 来插入他们，例如 {$t判定值}
          </div>
          <div>
            除此之外，这些变量可以在所有文本中使用：
            <n-flex size="small" wrap>
              <n-tag
                v-for="item in commonVars"
                :key="item"
                size="small"
                :bordered="false"
              >
                {{ item }}
              </n-tag>
            </n-flex>
          </div>
          <div>
            <span>以及，所有的自定义文本都可以嵌套使用，例如：</span>
            <div>
              <b>这里是{核心：骰子名字}，我是一个示例</b>
            </div>
            <div>默认会被解析为：</div>
            <div>
              <b>这里是海豹 bot，我是一个示例</b>
            </div>
            <div>注意！千万不要递归嵌套，会发生很糟糕的事情。</div>
          </div>

          <div class="help-block">
            <div>此外，支持插入图片，将图片放在骰子的适当目录，再写这样一句话即可：</div>
            <div><n-text code>[图:data/images/sealdice.png]</n-text></div>
            <div>可以参考 核心：骰子进群 词条</div>
            <div>同样的，可以使用 CQ 码插入图片和其他内容，关于 CQ 码，请参阅 onebot 项目文档</div>
          </div>

          <div class="help-block">
            <b>
              COC 的“判定 - 常规”和“判定 - 简短”主要区别是，多重检定会默认使用简短版本 (.ra
              3#射击)
            </b>
            <b>进行调整后，可以在左侧面板“指令测试”中进行测试！</b>
          </div>
        </n-text>
      </n-collapse-item>
    </n-collapse>
  </TipBox>
</template>

<script setup lang="ts">
import TipBox from '@/components/shared/TipBox.vue';

const commonVars = [
  '$t玩家',
  '$tQQ昵称',
  '$t个人骰子面数',
  '$tQQ',
  '$t骰子帐号',
  '$t骰子昵称',
  '$t群号',
  '$t群名',
];
</script>

<style scoped>
.help-block {
  margin-top: 1rem;
}
</style>
