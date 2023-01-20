import { LogItem } from "../types";

export interface indexInfoListItem {
    indexStart: number;
    indexContent: number;
    indexEnd: number;
    item: LogItem;
}

export type LogExportInfo = { text: string, indexInfoList: indexInfoListItem[] };


export class LogExporter {
    doExport(items: LogItem[]): LogExportInfo | undefined {
        return undefined;
    }
}
