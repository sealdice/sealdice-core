import { LogImporter } from "./_logImpoter";



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
}
