import dayjs from "dayjs";
import { LogItem } from "~/store";
import type { LogManager } from "../logManager";

export interface TextInfo {
  items: LogItem[];
  nicknames: Map<string, string>;
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
      data = `${start + this.tmpIMUserId.size}`;
      this.tmpIMUserId.set(name, data);
    }
    return data;
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
