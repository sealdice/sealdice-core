import { LogItem } from "~/store"
import { LogImporter, TextInfo  } from "./importers/_logImpoter"
import { SealDiceLogImporter } from "./importers/SealDiceLogImporter";
import { QQExportLogImporter } from "./importers/QQExportLogImporter";
import { SinaNyaLogImporter } from "./importers/SinaNyaLogImporter";
import { EditLogExporter } from "./exporters/EditLogExporter";
import { Emitter } from "./event";
import { indexInfoListItem } from "./exporters/logExporter";
import { EditLogImporter } from "./importers/EditLogImporter";


const readDiceNum = (expr: string, defaultVal = 100) => {
  let diceNum = defaultVal // 如果读不到，当作100处理
  const m = /[dD](\d+)/.exec(expr)
  if (m) {
    diceNum = parseInt(m[1])
  }
  return diceNum
}

export const trgCommandSolve = (item: LogItem) => {
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


export class LogManager {
  ev = new Emitter<{
    'textSet': (text: string) => void,
    'parsed': (ti: TextInfo) => void;
  }>(this);

  importers = [
    ['sealDice', new SealDiceLogImporter(this)],
    ['editLog', new EditLogImporter(this)],
    ['qqExport', new QQExportLogImporter(this)],
    ['sinaNya', new SinaNyaLogImporter(this)],
  ]

  exporters = {
    editLog: new EditLogExporter()
  }

  constructor() {
  }

  /**
   * 解析log
   * @param text 文本
   * @param genFakeHeadItem 生成一个伪项，专门用于放在最前面，来处理用户在页面最前方防止非格式化文本的情况
   * @returns 
   */
  parse(text: string, genFakeHeadItem: boolean = false) {
    for (let [theName, _importer] of this.importers) {
      const importer = _importer as LogImporter;
      if (importer.check(text)) {
        const ret = importer.parse(text);
        if (genFakeHeadItem) {
          const item = {} as LogItem;
          item.isRaw = true;
          item.message = ret.startText
          ret.items = [item, ...ret.items];
        }

        this.ev.emit('parsed', ret);
        return ret;
      }
    }
  }

  flush() {
    const ret = this.exporters.editLog.doExport(this.curItems);
    if (ret) {
      const { text, indexInfoList} = ret;
      this.lastText = text;
      this.lastIndexInfoList = indexInfoList;
      this.ev.emit('textSet', text);
    }
  }

  lastText = '';
  curItems: LogItem[] = [];
  lastIndexInfoList: indexInfoListItem[] = [];

  syncChange(curText: string, r1: number[], r2: number[]) {
    if (curText === this.lastText) {
      return
    }

    if (!this.lastText) {
      const info = this.parse(curText, true);
      if (info) {
        this.curItems = info.items;
        const ret = this.exporters.editLog.doExport(info.items);
        if (ret) {
          const { text, indexInfoList} = ret;
          this.lastText = text;
          this.lastIndexInfoList = indexInfoList;
          this.ev.emit('textSet', text);
        }
      }
    } else {
      // 增删
      const influence = []
      const [a, b] = r1;
      let last: indexInfoListItem | undefined = undefined;
      if (this.lastIndexInfoList.length) {
        last = this.lastIndexInfoList[this.lastIndexInfoList.length - 1];
      }

      // flush 代表刷新当前editer的文本，使其与内部数据结构一致，用于导入非海豹文本格式日志
      let needFlush = false;

      for (let i of this.lastIndexInfoList) {
        if (a < i.indexEnd && b >= i.indexStart) {
          influence.push(i);
        }
        if (i == last) {
          if (last.indexEnd === r1[0]) {
            influence.push(i);
          }
        }
      }
      // console.log("H", this.lastIndexInfoList, r1, influence)
      console.log("TEST", this.lastIndexInfoList, r1, r2, influence);

      // 省事起见，不做精细控制，直接重建被影响的部分
      const replacePart = curText.slice(...r2)
      if (influence.length) {
        const li = influence[0];
        const ri = influence[influence.length-1];
        const partLeft = this.lastText.slice(li.indexStart, a);
        const partRight = this.lastText.slice(b, ri.indexEnd);

        // 这部分就是被影响被影响区间的新的文本
        const changedText = partLeft + replacePart + partRight;
        const liIndex = this.lastIndexInfoList.indexOf(li);
        const riIndex = this.lastIndexInfoList.indexOf(ri);

        const left = this.lastIndexInfoList.slice(0, liIndex);
        const right = this.lastIndexInfoList.slice(riIndex+1, this.lastIndexInfoList.length);

        const rInfo = this.parse(changedText, influence[0] === this.lastIndexInfoList[0]);
        console.log('changedText', [changedText], rInfo);
        const offset = r2[1] - r2[0] - (r1[1] - r1[0]);
  
        if (rInfo) {
          needFlush = rInfo?.exporter != 'editLog';
        }
        // console.log("C2", left, 'R', right, 'O', offset)

        if (rInfo) {
          // 检查左方是否有文本剩余
          if (rInfo.startText) {
            if (left.length !== 0) {
              // 如果左侧仍有内容，归并进去
              left[left.length-1].item.message += rInfo.startText;
              left[left.length-1].indexEnd += rInfo.startText.length;  
            } else {
              // 如果自己已经是最左方，左侧可以选择不管，因为上一步parse的时候应该会生成一个新的顶部节点
              // 但是右侧需要进行合并
              if (right.length) {
                // 合并到当前最后一个节点中
                if (right[0].item.isRaw) {
                  // 没有这种情况？？？为什么
                  console.log('XXXXXXXXXXX', right)
                }
              }
            }
          }

          const ret = this.exporters.editLog.doExport(rInfo.items, li.indexStart);

          if (ret) {
            const { text, indexInfoList} = ret;

            // TODO: left 的最后一个
            // console.log('CCC', liIndex, riIndex, 'X', this.lastIndexInfoList.length);
            // console.log('CCC', left, 'I', indexInfoList, 'R', right);

            // 将受影响部分的LogItems替换
            this.lastIndexInfoList = [...left, ...indexInfoList, ...right];
            // 依次推迟右侧区域offset
            this.curItems = [...left.map(i => i.item), ...rInfo.items, ...right.map(i => i.item)];
          }
        } else {
          // 不能再构成规范格式，比如被删掉一部分
          if (left.length !== 0) {
            // 如果在中间的被删除，这样处理
            left[left.length-1].item.message += changedText;
            left[left.length-1].indexEnd += changedText.length;
            this.lastIndexInfoList = [...left, ...right];
            this.curItems = [...left.map(i => i.item), ...right.map(i => i.item)];
          } else {
            // 如果在开头的受到影响
            this.curItems[0].message = changedText;
            this.lastIndexInfoList[0].indexEnd = changedText.length;
            this.lastIndexInfoList = [this.lastIndexInfoList[0], ...right];
            this.curItems = [this.curItems[0], ...right.map(i => i.item)];
          }
        }

        right.map(i => {
          i.indexStart += offset;
          i.indexContent += offset;
          i.indexEnd += offset;
        });

        // 封装最终文本
        const newText = this.lastText.slice(0, li.indexStart) + changedText + this.lastText.slice(ri.indexEnd, this.lastText.length)
        this.lastText = newText;
        if (needFlush) this.flush();
      }

      // 遍历受影响部分
      // for (let i of influence) {
      //   if (a < i.indexStart && b > i.indexEnd) {
      //     console.log('此段被删除', i);
      //   } else if (a < i.indexContent && b < i.indexContent) {
      //     // 只影响标题
      //     console.log('只影响标题', i);
      //   } else if (a < i.indexContent && b >= i.indexContent) {
      //     console.log('标题加正文', i);
      //   } else if (a >= i.indexContent) {
      //     console.log('仅正文', i);
      //   } else {
      //     console.log('????', i, r1)
      //   }
      // }

      // this.ev.emit('textSet', newText);
    }
    // console.log(333, textAll.slice(...r1), textAll.slice(...r2));
  }
}

export const logMan = new LogManager()

setTimeout(() => {
  (globalThis as any).logMan = logMan;
}, 1000)
