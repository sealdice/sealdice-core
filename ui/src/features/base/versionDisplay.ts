import { shouldShowBuildMetaData } from './appChannel';

export type VersionDisplayInput = {
  simple?: string;
  detail?: {
    major?: number;
    minor?: number;
    patch?: number;
    buildMetaData?: string;
  };
};

const VERSION_FALLBACK_TEXT = '-';

/** 后端 buildMetaData 形如 `20260810.1a2b3c4`，即构建日期与提交号。 */
const BUILD_METADATA_PATTERN = /^(\d{4})(\d{2})(\d{2})\.([0-9a-f]+)$/i;

const HASH_DISPLAY_LENGTH = 4;

/** 主版本号 `X.Y.Z`，不含先行版本号与编译信息。 */
function formatMainVersion(input: VersionDisplayInput | undefined): string | undefined {
  const detail = input?.detail;
  if (
    typeof detail?.major === 'number' &&
    typeof detail.minor === 'number' &&
    typeof detail.patch === 'number'
  ) {
    return `${detail.major}.${detail.minor}.${detail.patch}`;
  }
  // 拿不到结构化版本号时退回 simple，并去掉其中的先行版本号后缀。
  const simple = input?.simple?.trim();
  if (!simple) return undefined;
  const [main] = simple.split('-', 1);
  return main || undefined;
}

/**
 * 编译信息压缩为 `MMDD.4位hash`。年份对使用者定位构建没有帮助，
 * 7 位 hash 在顶栏里也偏长，取 4 位已足够与近期提交对应。
 */
export function formatBuildMetaData(buildMetaData: string | undefined): string | undefined {
  const matched = BUILD_METADATA_PATTERN.exec(buildMetaData?.trim() ?? '');
  if (!matched) return undefined;
  const [, , month, day, hash] = matched;
  return `${month}${day}.${hash.slice(0, HASH_DISPLAY_LENGTH).toLowerCase()}`;
}

/**
 * 顶栏版本号。正式版与自编译只显示主版本号；开发版与未知追加 `+MMDD.4位hash`。
 * badge 已经承载了渠道语义，因此这里省去 `-dev` 后缀，不重复表达同一件事。
 */
export function formatDisplayVersion(
  input: VersionDisplayInput | undefined,
  channel: string | undefined
): string {
  const main = formatMainVersion(input);
  if (!main) return VERSION_FALLBACK_TEXT;
  if (!shouldShowBuildMetaData(channel)) return main;

  const meta = formatBuildMetaData(input?.detail?.buildMetaData);
  return meta ? `${main}+${meta}` : main;
}
