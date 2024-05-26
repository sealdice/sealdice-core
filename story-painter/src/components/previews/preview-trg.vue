<template>

  <div style="margin-bottom: .5rem" v-show="isShow">

    <div style="text-align: center; margin-bottom: 2rem; margin-top: 0.5rem;">
      <div>提示: 海豹骰与回声工坊达成了合作，<n-button type="primary" text tag="a" target="_blank" href="https://github.com/DanDDXuanX/TRPG-Replay-Generator">回声工坊</n-button>可以将海豹的log一键转视频哦！</div>
      <div>回声工坊的介绍和视频教程看这里：<n-button type="primary" text tag="a" target="_blank" href="https://www.bilibili.com/video/BV1PC4y1j7P2/">B站传送门</n-button></div>
    </div>

    <n-checkbox label="添加语音合成标记" v-model:checked="store.trgIsAddVoiceMark" />
  </div>

  <div class="preview" ref="preview" id="preview" v-show="isShow">
    <div style="position: absolute; right: 2rem; direction: rtl;">
      <n-button secondary type="primary" @click="copied" id="btnCopyPreviewTRG" style="z-index: 100">一键复制</n-button>
      <div class="mt-0.5 text-xs">注意: 长文本速度较慢</div>
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
import Item from './preview-trg-item.vue'
// @ts-ignore
import VirtualList from 'vue3-virtual-scroll-list';
import { useMessage } from 'naive-ui';

const props = defineProps<{
  isShow: boolean,
  previewItems: LogItem[],
}>();

const store = useStore();
const message = useMessage();
const isAddVoiceMark = ref(true)

const copied = () => {
  message.success('进行了复制！')
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
