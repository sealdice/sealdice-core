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
    // if (!this.latestData) this.check(text);
    const nicknames = new Map<string, string>();
    const data = this.latestData as { items: LogItem[] };
    let startText = '';
    for (let i of data.items) {
      let role = '角色'
      if (i.nickname.toLowerCase().startsWith('ob')) {
        role = '隐藏'
      }
      if (i.isDice) {
        role = '骰子'
      }
      nicknames.set(i.nickname, `${i.IMUserId}`);
      i.message += '\n\n';
    }
    return { items: data.items, nicknames, startText };
  }
}
