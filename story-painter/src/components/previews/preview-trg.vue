<template>

  <div style="margin-bottom: .5rem" v-show="isShow">

    <div style="text-align: center; margin-bottom: 2rem; margin-top: 0.5rem;">
      <div>提示: 海豹骰与回声工坊达成了合作，<el-link type="primary" target="_blank" href="https://github.com/DanDDXuanX/TRPG-Replay-Generator">回声工坊</el-link>可以将海豹的log一键转视频哦！</div>
      <div>回声工坊的介绍和视频教程看这里：<el-link type="primary" target="_blank" href="https://www.bilibili.com/video/BV1GY4y1H7wK/">B站传送门</el-link></div>
    </div>

    <el-checkbox :border="true" label="添加语音合成标记" v-model="store.trgIsAddVoiceMark" />
    <!-- <el-checkbox label="回声工坊" v-model="isShowPreviewTRG2" /> -->
  </div>

  <div class="preview" ref="preview" id="preview" v-show="isShow">
    <div style="position: absolute; right: 2rem; direction: rtl;">
      <el-button @click="copied" id="btnCopyPreviewTRG" size="large" style="z-index: 100">一键复制</el-button>
      <div style="font-size: 0.8rem;">注意: 长文本速度较慢</div>
      <!-- <div v-if="copyCount != 0">进度: {{ copyCount }} / {{ copyCountAll }}</div> -->
    </div>
    <div v-if="previewItems.length === 0">
      <div>染色失败，内容为空或无法识别此格式。</div>
      <div>已知支持的格式有: 海豹Log(json)、赵/Dice!原始文件、塔原始文件</div>
      <div>请先清空编辑框，再行复制</div>
    </div>

		<VirtualList
			class="list-dynamic scroll-touch scroller"
			:data-key="'index'"
			:data-sources="previewItems"
			:data-component="Item"
			:estimate-size="20"
			:item-class="''"
		/>
  </div>

</template>

<script setup lang="ts">
import ClipboardJS from 'clipboard';
import { h, nextTick, onMounted, ref, render, watch } from 'vue';
import { useStore } from '~/store';
import { LogItem, packNameId } from '~/logManager/types';
import { ElLoading, ElMessageBox, ElNotification, ElMessage, ElButton, ElCheckbox, ElLink } from "element-plus";
import Item from './preview-trg-item.vue'
// @ts-ignore
import VirtualList from 'vue3-virtual-scroll-list';

const props = defineProps<{
  isShow: boolean,
  previewItems: LogItem[],
}>();

const store = useStore();
const isAddVoiceMark = ref(true)

const copied = () => {
  ElMessage.success('进行了复制！')
}

let clip: ClipboardJS;
const copyCount = ref(0);
const copyCountAll = ref(1);

watch(() => props.isShow, (val: any) => {
  if (val) {
    store.exportOptions.imageHide = true

    nextTick(() => {
      if (clip) return;
      clip = new ClipboardJS('#btnCopyPreviewTRG', {
        text: () => {
          // 这个实现很好，很简单，可惜太慢
          // 先用着吧
          copyCountAll.value = props.previewItems.length || 1;
          copyCount.value = 0;
          const el = document.createElement('span');
          const items = [];
          for (let i of props.previewItems) {
            const html = h(Item, { source: i });
            render(html, el);
            items.push(el.textContent);
            copyCount.value += 1;
          }
          return items.join('\n');
        }
      })
    });
  }
})
</script>
