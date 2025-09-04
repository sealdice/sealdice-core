import dayjs from "dayjs";
import { useStore } from "~/store";
import { CharItem, LogItem } from "../types";
import { LogImporter } from "./_logImpoter";

export const reQQExportLineTest = /^((\d{4}\/\d{2}\/\d{2}\s)?\d{2}:\d{2}:\d{2})?<(.+?)(\([^)]+\))?>:(.*)$/m;
export const reQQExport = new RegExp(reQQExportLineTest, 'gm')


export class RenderedLogImporter extends LogImporter {
  // 已导出的文本
  // 2022/08/18 19:43:21<Keeper(904002355)>:领航技能会判定自动失败
  // 2022/08/18 19:43:32<昆特·爱尔芙(562308967)>:唔……那我的信息有没有什么可以利用的）
  // 2022/08/18 19:43:57<Keeper(904002355)>:（罗克波特镇地图

  // 2022/08/18 19:43:21<Keeper>:领航技能会判定自动失败
  // 2022/08/18 19:43:32<昆特·爱尔芙>:唔……那我的信息有没有什么可以利用的）
  // 2022/08/18 19:43:57<Keeper>:（罗克波特镇地图

  // 19:43:21<Keeper>:领航技能会判定自动失败
  // 19:43:32<昆特·爱尔芙>:唔……那我的信息有没有什么可以利用的）
  // 19:43:57<Keeper>:（罗克波特镇地图

  check(text: string): boolean {
    if (reQQExportLineTest.test(text)) {
      return true;
    }
    return false;
  }

  get name() {
    return '染色器导出格式'
  }

  parse(text: string) {
    const store = useStore();

    reQQExport.lastIndex = 0; // 注: 默认值即为0 并非-1
    const charInfo = new Map<string, CharItem>();
    const items = [] as LogItem[];
    let lastItem: LogItem = null as any;
    let lastIndex = 0;
    let startText = '';

    while (true) {
      const m = reQQExport.exec(text);
      if (m) {
        if (lastItem) {
          lastItem.message += text.slice(lastIndex, m.index);
          lastItem.message = lastItem.message;
        } else {
          startText = text.slice(0, m.index);
        }

        const item = {} as LogItem;
        item.nickname = m[3];

        if (m[4]) {
          item.IMUserId = 'QQ:' + m[4].slice(1, -1);
        } else {
          item.IMUserId = this.getAutoIMUserId(store.pcList.length, item.nickname);
        }

        this.setCharInfo(charInfo, item);
        [item.time, item.timeText] = this.parseTime((m[1] || ''));
        item.message = m[5] + '\n';
        items.push(item);

        lastItem = item;
        lastIndex = m.index + m[0].length;
      } else {
        if (lastItem) {
          lastItem.message += text.slice(lastIndex, text.length);
          lastItem.message = lastItem.message;
        }
        break;
      }
    }

    return { items, charInfo, startText };
  }
}
