import dayjs from "dayjs";
import { useStore } from "~/store";
import { CharItem, LogItem } from "../types";
import { LogImporter } from "./_logImpoter";

import customParseFormat from 'dayjs/plugin/customParseFormat';
dayjs.extend(customParseFormat);

export const reFvttExportLineTest = /^\[(\d+\/\d+\/\d+, \d+:\d+:\d+ [AP]M)\] (.+)$/m;
export const reFvttExport = /^\[(\d+\/\d+\/\d+, \d+:\d+:\d+ [AP]M)\] ([^\n]+)\n(.*)/s;


export class FvttLogImporter extends LogImporter {
  // [3/21/2023, 3:51:38 PM] 张大胆
  // Bite The Winter Wolf attacks with its Bite. If the target is a creature, it must make a Strength saving throw or be knocked prone.
  // 攻击 伤害 豁免骰 DC 14 力量
  // 天生武器 已装备 熟练 1 动作 5 英尺
  // ---------------------------
  // [3/21/2023, 3:51:58 PM] 张大胆
  // 17
  // 1d20 + 4 + 2 = 11 + 4 + 2 = 17
  // ---------------------------
  // [3/21/2023, 3:52:05 PM] 张大胆
  // 11
  // 4d6 + 4 = 7 + 4 = 11
  // ---------------------------
  // [3/21/2023, 3:52:25 PM] 星尘
  // 力量 Saving Throw DC: 14
  // Actors
  // 维赫勒
  // ...20
  // 2d20kh 19 5 19
  // 张大胆
  // ...5
  // 2d20kl 1 1 14
  // Public Roll Group DC:
  // ---------------------------
  // [3/25/2023, 1:51:27 AM] 星尘
  // 11111
  // ---------------------------
  // [3/25/2023, 1:51:54 AM] 星尘
  // （）
  // ---------------------------
  // [3/25/2023, 1:51:58 AM] 星尘
  // （121212）
  check(text: string): boolean {
    if (reFvttExportLineTest.test(text)) {
      return true;
    }
    return false;
  }

  get name() {
    return 'fvtt日志格式'
  }

  parse(text: string) {
    const store = useStore();

    const charInfo = new Map<string, CharItem>();
    const items = [] as LogItem[];
    let startText = '';

    for (let i of text.split('---------------------------')) {
      const m = reFvttExport.exec(i.trim());
      if (m) {
        if (m[3].trim() === '') continue;
        const item = {} as LogItem;
        item.nickname = m[2];
        item.IMUserId = this.getAutoIMUserId(store.pcList.length, item.nickname);
        this.setCharInfo(charInfo, item);
        item.time = dayjs(m[1], 'M/D/YYYY, h:m:s A').unix();
        item.message = m[3].trimStart() + '\n\n';
        items.push(item);
      }
    }

    return { items, charInfo, startText };
  }
}
