<template>
  <div class="">
    <span style="color: #aaa" class="_time" v-if="!store.exportOptions.timeHide">[color={{ getTimeColor() }}]{{ timeSolve(source)
    }}[/color]</span>
    <span :style="{ 'color': colorByName(source) }">[color={{ colorByName(source) }}]
      <span class="_nickname">{{ nicknameSolve(source) }}</span>
      <span v-html="bbsMessageSolve(source)"></span>
      [/color]</span>
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs';
import { computed } from 'vue';
import { LogItem, packNameId } from '~/logManager/types';
import { useStore } from '~/store';
import { escapeHTML, msgCommandFormat, msgImageFormat, msgIMUseridFormat, msgOffTopicFormat } from '~/utils';

const store = useStore();

defineProps({
  source: {
    type: Object as () => LogItem,
    default: () => { },
  },
});

const getTimeColor = () => {
  if (store.bbsUseColorName) return 'silver';
  return '#aaaaaa'
}

const colorByName = (i: LogItem) => {
  // const info = store.pcMap.get(`${i.nickname}-`);
  const info = store.pcMap.get(packNameId(i));
  if (store.bbsUseColorName) {
    return store.colorHexToName(info?.color || '#ffffff');
  }
  return info?.color || '#ffffff';
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
  if (typeof i.time === 'number' && i.time !== 0) {
    timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
  } else {
    if (i.timeText) {
      timeText = i.timeText
    } else {
      timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
    }
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

// TODO: 当时写的时候没想太明白，应该写成tsx的
const bbsMessageSolve = (i: LogItem) => {
  const options = Object.assign({}, store.exportOptions)
  options.imageHide = true;
  const id = packNameId(i);
  if (store.pcMap.get(id)?.role === '隐藏') return '';

  let msg = msgImageFormat(escapeHTML(i.message), options);
  msg = msgOffTopicFormat(msg, store.exportOptions, i.isDice);
  msg = msgCommandFormat(msg, store.exportOptions);
  msg = msgIMUseridFormat(msg, store.exportOptions, i.isDice);
  msg = msgOffTopicFormat(msg, store.exportOptions, i.isDice); // 再过滤一次

  if (i.isDice) {
    msg = nameReplace(msg)
  }
  if (store.bbsUseSpaceWithMultiLine) {
    const toSpace = (text: string) => {
      const lst: string[] = [];
      for (let i = 0; i < text.length; i++) {
        if (text[i] === ':' || text[i] == '/' || text[i] === '[' || text[i] === ']') {
          lst.push('&nbsp;');
        } else {
          lst.push('&ensp;');
        }
      }
      lst.push('&nbsp;');
      return lst.join('');
      // return '&ensp;'.repeat(text.length + 1); // 这里总是有一个前置空格
    }
    return msg.trim().replaceAll('<br />', '\n').replaceAll('\n', '<br/><span class="lf">\n</span>' + (!store.exportOptions.timeHide ? `<span style='color:#aaa'>${toSpace(timeSolve(i))}</span>` : '&ensp;') + escapeHTML(nicknameSolve(i)))
    // return msg.trim().replaceAll('<br />', '\n').replaceAll('\n', '<br/><span class="lf">\n</span>' + (!store.exportOptions.timeHide ? `<span style='color:#aaa'>${toSpace('[color=#aaaaaa]' + timeSolve(i) + '[/color]')}</span>` : '') + '&ensp;'.repeat(`[color=${colorByName(i)}]`.length) + nicknameSolve(i))
  }
  return msg.trim().replaceAll('<br />', '\n').replaceAll('\n', '[/color]<br/><span class="lf">\n</span>' + (!store.exportOptions.timeHide ? `<span style='color:#aaa'>[color=${getTimeColor()}]${timeSolve(i)}[/color]</span>` : '') + `[color=${colorByName(i)}] ` + escapeHTML(nicknameSolve(i)))
}
</script>
