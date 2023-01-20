<template>
  <div class="" v-if="trgMessageSolve(source).trim() !== ''">
    <div :style="source.isDice ? 'margin-top: 16px; margin-bottom: 16px' : ''">
      <span :style="{ 'color': colorByName(source) }" v-if="source.isDice"># </span>
      <span :style="{ 'color': colorByName(source) }" class="_nickname">{{ nicknameSolve(source) }}</span>
      <span :style="{ 'color': colorByName(source) }" v-html="trgMessageSolve(source)"></span>
      <div v-if="source.commandInfo" style="white-space: pre-wrap;">{{
        trgCommandSolve(source)
      }}</div>
      <span v-if="store.trgIsAddVoiceMark && (!source.isDice)">{*}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LogItem, packNameId } from '~/logManager/types';
import { useStore } from '~/store';
import { msgCommandFormat, msgImageFormat, msgIMUseridFormat, msgOffTopicFormat } from '~/utils';

const store = useStore();

const props = defineProps({
  source: {
    type: Object as () => LogItem,
    default: () => { },
  }
});

const colorByName = (i: LogItem) => {
  // const info = store.pcMap.get(`${i.nickname}-`);
  const info = store.pcMap.get(packNameId(i));
  return info?.color || '#fff';
}

let findPC = (name: string) => {
  for (let i of store.pcList) {
    if (i.name === name) {
      return i
    }
  }
}

const nicknameSolve = (i: LogItem) => {
  const options = store.exportOptions
  const u = findPC(i.nickname)
  let kpFlag = u?.role === '主持人' ? ',KP' : ''
  return `[${i.nickname}${kpFlag}]:`
  // [张安翔]:最基本的对话行
}

const nameReplace = (msg: string) => {
  for (let i of store.pcList) {
    msg = msg.replaceAll(`<${i.name}>`, `${i.name}`)
  }
  return msg
}

const trgMessageSolve = (i: LogItem) => {
  const id = packNameId(i);
  if (store.pcMap.get(id)?.role === '隐藏') return '';

  let msg = msgImageFormat(i.message, store.exportOptions, true);
  msg = msgOffTopicFormat(msg, store.exportOptions, i.isDice);
  msg = msgCommandFormat(msg, store.exportOptions);
  msg = msgIMUseridFormat(msg, store.exportOptions, i.isDice);

  let extra = ''
  if (i.isDice) {
    msg = nameReplace(msg)
    extra = '# '
  }
  msg = msg.trim().replaceAll('"', '').replaceAll('\\', '') // 移除反斜杠和双引号
  const prefix = store.trgIsAddVoiceMark ? '{*}' : ''
  return msg.replaceAll('<br />', '\n').replaceAll('\n', prefix + '<br /> ' + extra + nicknameSolve(i))
}

const readDiceNum = (expr: string, defaultVal = 100) => {
  let diceNum = defaultVal // 如果读不到，当作100处理
  const m = /[dD](\d+)/.exec(expr)
  if (m) {
    diceNum = parseInt(m[1])
  }
  return diceNum
}

const trgCommandSolve = (item: LogItem) => {
  if (item.commandInfo) {
    const ci = item.commandInfo
    if (ci.rule === 'coc7') {
      switch (ci.cmd) {
        case 'ra': {        
          let items = []
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.expr1)
            items.push(`(${ci.pcName}的${i.expr2},${diceNum},${i.attrVal},${i.checkVal})`)
          }
          return `<dice>:${items.join(',')}`
          break
        }
        case 'st': {
          // { "cmd": "st", "items": [ { "attr": "hp", "isInc": false, "modExpr": "1d4", "type": "mod", "valNew": 63, "valOld": 65 } ], "pcName": "木落", "rule": "coc7" }
          let items = []
          for (let i of ci.items) {
            if (i.attr == 'hp') {
              let maxNow = Math.max(i.valOld, i.valNew)
              items.push(`<hitpoint>:(${ci.pcName},${maxNow},${i.valOld},${i.valNew})`)
            }
            // let diceNum = readDiceNum(i.exprs[0])
            // items.push(`(${ci.pcName}的${i.exprs[0]},${diceNum},${i.sanOld},${i.checkVal})`)
          }
          const tip = '# 请注意，当前版本需要手动调整下方最大生命值(第二项)\n'
          return tip + `${items.join('\n')}`
          break
        }
        case 'sc': {
          // { "cmd": "sc", "cocRule": 11, "items": [ { "checkVal": 55, "exprs": [ "d100", "0", "1" ], "rank": -2, "sanNew": 0, "sanOld": 0 } ], "pcName": "木落", "rule": "coc7" }
          let items = []
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.exprs[0])
            items.push(`(${ci.pcName}的${i.exprs[0]},${diceNum},${i.sanOld},${i.checkVal})`)
          }
          return `<dice>:${items.join(',')}`
          break
        }
      }
    }
    if (ci.rule === 'dnd5e') {
      switch (ci.cmd) {
        case 'st': {
          // {"cmd":"st","items":[{"attr":"hp","isInc":false,"modExpr":"3","type":"mod","valNew":7,"valOld":10}],"pcName":"海岸线","rule":"dnd5e"}
          let items = []
          let hasHp = false
          for (let i of ci.items || []) {
            if (i.attr == 'hp') {
              let maxNow = Math.max(i.valOld, i.valNew)
              items.push(`<hitpoint>:(${ci.pcName},${maxNow},${i.valOld},${i.valNew})`)
              hasHp = true
            }
          }
          let tip = ''
          if (hasHp) {
            let tip = '# 请注意，当前版本需要手动调整下方最大生命值(第二项)\n'
          }
          return tip + `${items.join('\n')}`
          break
        }
        case 'rc': {
          // {"cmd":"rc","items":[{"expr":"D20 + 体质豁免","reason":"体质豁免","result":15}],"pcName":"阿拉密尔•利亚顿","rule":"dnd5e"}
          let items = []

          let tip = ''
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.expr, 20)
            items.push(`(${ci.pcName}的${i.reason}检定,${diceNum},NA,${i.result})`)
            tip = '# 请注意，DND的最大面数可能为 D20+各种加值，需要手动二次调整\n'
          }
          return tip + `<dice>:${items.join(',')}`
          break
        }
      }
    }

    switch (ci.cmd) {
      case 'roll': {
          // { "cmd": "roll", "items": [ { "dicePoints": 100, "expr": "D100", "result": 30 } ], "pcName": "木落" }
          let items = []
          for (let i of ci.items) {
            let diceNum = readDiceNum(i.expr)
            items.push(`(${ci.pcName}的${i.expr},${diceNum},NA,${i.result})`)
          }
          return `<dice>:${items.join(',')}`
          break
        }
    }
    return ci
  }
}
</script>
