import dayjs from "dayjs";
import { useStore } from "~/store";
import { CharItem, LogItem } from "../types";
import { LogImporter, TextInfo } from "./_logImpoter";

export const reSinaNyaLineTest = /^<(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d)>\s+\[?([^\]]+)\]?:\s+([^\n]+)$/m
export const reSinaNya = new RegExp(reSinaNyaLineTest, 'gm')


export class SinaNyaLogImporter extends LogImporter {
  // <2022-03-15 20:02:30.0>	[月歌]:	“锁上了么...”扭头看了看周围，看到了个在看假草的牧野，偷偷掏出螺丝刀尝试撬锁
  check(text: string): boolean {
    if (reSinaNyaLineTest.test(text)) {
      return true;
    }
    return false;
  }

  parse(text: string): TextInfo {
    const store = useStore();

    reSinaNya.lastIndex = 0; // 注: 默认值即为0 并非-1
    const charInfo = new Map<string, CharItem>();
    const items = [] as LogItem[];
    let lastItem: LogItem = null as any;
    let lastIndex = 0;
    let startText = '';

    while (true) {
      const m = reSinaNya.exec(text);
      if (m) {
        if (lastItem) {
          lastItem.message += text.slice(lastIndex, m.index);
          lastItem.message = lastItem.message;
        } else {
          startText = text.slice(0, m.index);
        }

        const item = {} as LogItem;
        item.nickname = m[2];
        item.IMUserId = this.getAutoIMUserId(store.pcList.length, item.nickname);
        this.setCharInfo(charInfo, item);
        item.time = dayjs(m[1]).unix();
        item.message = m[3];
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

    for (let i of items) {
      // 比实际多一个\n，纯粹排版用
      i.message += '\n';
    }

    return { items, charInfo, startText };
  }
}
