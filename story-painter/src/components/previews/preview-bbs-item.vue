<template>
  <div class="">
    <span style="color: #aaa" class="_time" v-if="!store.exportOptions.timeHide">[color=#silver]{{ timeSolve(source) }}[/color]</span>
    <span :style="{ 'color': colorByName(source) }">[color={{ colorByName(source) }}]
      <span class="_nickname">{{ nicknameSolve(source) }}</span>
      <span v-html="bbsMessageSolve(source)"></span>
      [/color]</span>
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs';
import { LogItem, packNameId } from '~/logManager/types';
import { useStore } from '~/store';
import { msgCommandFormat, msgImageFormat, msgIMUseridFormat, msgOffTopicFormat } from '~/utils';

const store = useStore();

defineProps({
  source: {
    type: Object as () => LogItem,
    default: () => { },
  },
});

const colorByName = (i: LogItem) => {
  // const info = store.pcMap.get(`${i.nickname}-`);
  const info = store.pcMap.get(packNameId(i));
  return info?.color || '#fff';
}

const nicknameSolve = (i: LogItem) => {
	let userid = '(' + i.IMUserId + ')'
	const options = store.exportOptions
	if (options.userIdHide) {
		userid = ''
	}
  return `<${i.nickname}${userid}>`
}


const timeSolve = (i: LogItem) => {
	let timeText = i.time.toString()
	const options = store.exportOptions
	if (typeof i.time === 'number') {
		timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
	}
	if (options.timeHide) {
		timeText = ''
	}
	return timeText
}

const nameReplace = (msg: string) => {
	for (let i of store.pcList) {
		msg = msg.replaceAll(`<${i.name}>`, `${i.name}`)
	}
	return msg
}

const bbsMessageSolve = (i: LogItem) => {
  const options = Object.assign({}, store.exportOptions)
  options.imageHide = true;
  const id = packNameId(i);
  if (store.pcMap.get(id)?.role === '隐藏') return '';

  let msg = msgImageFormat(i.message, options);
  msg = msgOffTopicFormat(msg, store.exportOptions, i.isDice);
  msg = msgCommandFormat(msg, store.exportOptions);
  msg = msgIMUseridFormat(msg, store.exportOptions, i.isDice);

  if (i.isDice) {
    msg = nameReplace(msg)
  }
  return msg.trim().replaceAll('<br />', '\n').replaceAll('\n', '[/color]<br /> ' + (!store.exportOptions.timeHide ? `<span style='color:#aaa'>[color=#silver]${timeSolve(i)}[/color]</span>` : '') + `[color=${colorByName(i)}] ` + nicknameSolve(i))
}
</script>
