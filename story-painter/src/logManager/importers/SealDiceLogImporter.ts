import { LogItem } from "~/store";
import { LogImporter, TextInfo } from "./_logImpoter";



export class SealDiceLogImporter extends LogImporter {
  latestData: any;

  check(text: string): boolean {
    let isTrpgLog = false;
    try {
      const sealFormat = JSON.parse(text);
      if (sealFormat.items && sealFormat.items.length > 0) {
        const keys = Object.keys(sealFormat.items[0]);
        isTrpgLog = keys.includes('isDice') && keys.includes('message');
        this.latestData = sealFormat;
      }
    } catch (e) { }

    return isTrpgLog;
  }

  parse(text: string): TextInfo {
    const nicknames = new Map<string, string>();
    const items = this.latestData as LogItem[];
    let startText = '';
    for (let i of items) {
      let role = '角色'
      if (i.nickname.toLowerCase().startsWith('ob')) {
        role = '隐藏'
      }
      if (i.isDice) {
        role = '骰子'
      }
      nicknames.set(i.nickname, role);
    }
    return { items, nicknames, startText };
  }
}
