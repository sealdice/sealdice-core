import dayjs from "dayjs";
import type { LogManager } from "../logManager";
import { CharItem, LogItem, packNameId } from "../types";


export interface TextInfo {
  items: LogItem[];
  charInfo: Map<string, CharItem>;
  // nicknames: Map<string, string>;
  startText: string;
  exporter?: string;
}

export class LogImporter {
  parent: LogManager;
  tmpIMUserId = new Map<string, string>();

  constructor(man: LogManager) {
    this.parent = man;
  }

  getAutoIMUserId(start: number, name: string) {
    let data = this.tmpIMUserId.get(name);
    if (!data) {
      data = `${1001+start + this.tmpIMUserId.size}`;
      this.tmpIMUserId.set(name, data);
    }
    return data;
  }

  setCharInfo(charInfo: Map<string, CharItem>, item: LogItem) {
    const id = packNameId(item);
    if (!charInfo.get(id)) {
      let role = item.role
      if (!role) {
        role = '角色'
        if (item.nickname.toLowerCase().startsWith('ob')) {
          role = '隐藏'
        }
        if (item.isDice) {
          role = '骰子'
        }
      }

      charInfo.set(id, {
        name: item.nickname,
        IMUserId: item.IMUserId,
        role: role as any,
        color: '',
      })
    }
  }

  parseTime(arg0: string): [number, string | undefined] {
    const t = dayjs(arg0).unix();
    if (isNaN(t)) {
      return [0, arg0]
    } else {
      return [t, undefined];
    }
  }

  check(text: string): boolean { return false }
  parse(text: string): TextInfo { return null as any }
}
