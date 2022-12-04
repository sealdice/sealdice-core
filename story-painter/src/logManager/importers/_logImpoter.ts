import dayjs from "dayjs";
import { LogItem } from "~/store";
import type { LogManager } from "../logManager";

export interface TextInfo {
  items: LogItem[];
  nicknames: Set<string>;
  startText: string;
  exporter?: string;
}

export class LogImporter {
  parent: LogManager;

  constructor(man: LogManager) {
    this.parent = man;
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
