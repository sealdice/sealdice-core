import dayjs from "dayjs";
import { useStore } from "~/store";
import { CharItem, LogItem } from "../types";
import { LogImporter } from "./_logImpoter";

export const reQQExportLineTest = /^(\d{4}-\d{2}-\d{2} \d{1,2}:\d{1,2}:\d{1,2})\s+(.+?)(\([^)]+\)|\<[^>]+\>)$/m
export const reQQExport = new RegExp(reQQExportLineTest, 'gm')


export class QQExportLogImporter extends LogImporter {
  // 2022-05-10 11:28:25 名字(12345)
  check(text: string): boolean {
    if (reQQExportLineTest.test(text)) {
      return true;
    }
    return false;
  }

  get name() {
    return 'QQ导出格式'
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
        item.IMUserId = 'QQ:' + m[3].slice(1, -1);
        item.nickname = m[2];
        this.setCharInfo(charInfo, item);
        item.time = dayjs(m[1]).unix();
        item.message = '';
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
