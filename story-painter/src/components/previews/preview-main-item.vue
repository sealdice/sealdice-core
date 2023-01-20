<template>
  <!-- 这句是为了防止空元素占行 -->
  <div class="list-item-dynamic" v-if="previewMessageSolve(source).trim() !== ''">
    <!-- {{  source  }} -->
    <span style="color: #aaa" class="_time" v-if="!store.exportOptions.timeHide">{{ timeSolve(source) }}</span>
    <span :style="{ 'color': colorByName(source) }" class="_nickname">{{ nicknameSolve(source) }}</span>
    <span :style="{ 'color': colorByName(source) }" v-html="previewMessageSolve(source)"></span>
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs';
import { LogItem, useStore } from '~/store';
import { getCanvasFontSize, getTextWidth, msgCommandFormat, msgImageFormat, msgIMUseridFormat, msgOffTopicFormat } from '~/utils';

const store = useStore();

defineProps({
  source: {
    type: Object as () => LogItem,
    default: () => { },
  },
});

const colorByName = (i: LogItem) => {
  // const info = store.pcMap.get(`${i.nickname}-`);
  const info = store.pcMap.get(`${i.nickname}-${i.IMUserId}`);
  return info?.color;
}


const nicknameSolve = (i: LogItem) => {
  let userid = '(' + i.IMUserId + ')'
  const options = store.exportOptions
  if (options.userIdHide) {
    userid = ''
  }
  return `<${i.nickname}${userid}>:`
}


const timeSolve = (i: LogItem) => {
  let timeText = i.time.toString()
  const options = store.exportOptions
  if (options.timeHide) {
    timeText = ''
  } else {
    if (typeof i.time === 'number') {
      timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
    }
  }
  return timeText
}

const nameReplace = (msg: string) => {
  for (let i of store.pcList) {
    msg = msg.replaceAll(`<${i.name}>`, `${i.name}`)
  }
  return msg
}

let canvasFontSize = '';

const previewMessageSolve = (i: LogItem) => {
  let msg = msgImageFormat(i.message, store.exportOptions, true);
  msg = msgOffTopicFormat(msg, store.exportOptions, i.isDice);
  msg = msgCommandFormat(msg, store.exportOptions);
  msg = msgIMUseridFormat(msg, store.exportOptions, i.isDice);

  const prefix = (!store.exportOptions.timeHide ? `${timeSolve(i)}` : '') + nicknameSolve(i)
  if (i.isDice) {
    msg = nameReplace(msg)
  }

  if (canvasFontSize === '') {
    // store.previewElement as any
    canvasFontSize = getCanvasFontSize(document.getElementById('preview') as any);
  }
  const length = getTextWidth(prefix, canvasFontSize);
  // return msg.replaceAll('<br />', '\n').replaceAll('\n', '<br /> ' + `<span style="color:white">${prefix}</span>`)
  return msg.replaceAll('<br />', '\n').replaceAll(/\n([^\n]+)/g, `<p style="margin-left: ${length}px; margin-top: 0; margin-bottom: 0">$1</p>`)
}
</script>
