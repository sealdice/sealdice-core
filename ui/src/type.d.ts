interface JsScriptInfo {
  name: string;
  enable: boolean;
  version: string;
  author: string;
  license: string;
  homepage: string;
  desc: string;
  grant?: any;
  updateTime: number;
  installTime: number;
  errText: string;
  filename: string;
  updateUrls?: string[];
  official: boolean;
  builtin: boolean;
  builtinUpdated: boolean;
}

interface JsPluginConfigItem {
  key: string;
  type: string;
  defaultValue: any;
  value: any;
  option: any[];
  deprecated: boolean;
  description: string;
}
interface JsPluginConfig {
  pluginName: string;
  configs: Map<string, JsPluginConfigItem>;
}
interface HelpDocData {
  helpInfo: HelpDocHelpInfo;
  docTree: HelpDoc[];
}

interface HelpDocHelpInfo {
  [key: string]: number;
}

interface HelpDoc {
  name: string;
  path: string;
  group: string;
  type: ".json" | ".xlsx";
  isDir: boolean;
  loadStatus: 0 | 1 | 2;
  deleted: boolean;

  children: HelpDoc[] | null;
}

interface HelpTextItemQuery {
  pageNum: number;
  pageSize: number;
  total: number;
  id?: number;
  group?: string;
  from?: string;
  title?: string;
}

interface HelpTextItem {
  id: number;
  group: string;
  from: string;
  title: string;
  content: string
  packageName: string
  keyWords: string
}

interface AdvancedConfig {
  enable: boolean,
  storyLogBackendUrl: string,
  storyLogApiVersion: string,
  storyLogBackendToken: string,
}