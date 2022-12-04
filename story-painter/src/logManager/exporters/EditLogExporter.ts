import { indexInfoListItem, LogExporter, LogExportInfo } from "./logExporter";
import { LogItem } from "~/store";
import dayjs from "dayjs";

// 编辑页面
export class EditLogExporter extends LogExporter {
  doExport(items: LogItem[], indexOffset = 0): LogExportInfo | undefined {
    let text = ""
    let index = 0 + indexOffset
    const indexInfoList = []

    for (let i of items) {
      if (i.isRaw) {
        let indexStart = index
        let indexContent = index
        let indexEnd = index + i.message.length;
        text += i.message;
        index = indexEnd;
        const indexInfo = { indexStart, indexContent, indexEnd, item: i };
        indexInfoList.push(indexInfo);
        continue
      }

      let idSuffix = ''
      if (i.isDice) {
        idSuffix = ` #${i.id}`
      }
  
      let indexStart = index
      const timeText = i.timeText ? i.timeText : dayjs.unix(i.time).format('YYYY/MM/DD HH:mm:ss');
      let imuid = '';
      if (i.IMUserId) {
        imuid = `(${i.IMUserId})`;
      }

      text += `${i.nickname}${imuid} ${timeText}${idSuffix}\n`
      index = indexOffset + text.length
      let indexContent = index
      text += `${i.message}`
      index = indexOffset + text.length
      let indexEnd = index

      const indexInfo = { indexStart, indexContent, indexEnd, item: i };
      indexInfoList.push(indexInfo);
    }

    return { text, indexInfoList }
  }
}
