import { CharItem, LogItem } from "../types";
import { LogImporter, TextInfo } from "./_logImpoter";


export class SealDiceLogImporter extends LogImporter {
  latestData: any;

  get name() {
    return '海豹JSON格式'
  }

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
    const charInfo = new Map<string, CharItem>();
    const data = this.latestData as { items: LogItem[] };
    let startText = '';
    for (let i of data.items) {
      this.setCharInfo(charInfo, i);
      // 这个\r\n替换是为了防止logman因为新旧文本不同，导致重新格式化
      i.message = i.message.replaceAll('\r\n', '\n');
      i.message += '\n\n';
    }
    console.log(data);
    return { items: data.items, charInfo, startText };
  }
}
