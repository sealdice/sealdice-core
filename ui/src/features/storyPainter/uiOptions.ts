import type { StoryPainterOptions } from './types';

export interface StoryPainterOptionItem {
  label: string;
  desc: string;
  key: keyof StoryPainterOptions;
}

export const storyPainterOptionList: StoryPainterOptionItem[] = [
  {
    label: '骰子指令过滤',
    desc: '开启后，不显示pc指令，正常显示指令结果',
    key: 'commandHide',
  },
  {
    label: '表情图片过滤',
    desc: '开启后，文本内所有的表情包和图片将被隐藏不显示',
    key: 'imageHide',
  },
  {
    label: '场外发言过滤',
    desc: '开启后，所有以(和（为开头的发言将被吃掉不显示',
    key: 'offTopicHide',
  },
  {
    label: '时间显示过滤',
    desc: '开启后，日期和时间会在导出结果中隐藏',
    key: 'timeHide',
  },
  {
    label: '平台帐号隐藏',
    desc: '开启后，IM 平台账号（如 QQ 号）将在导出结果中不显示',
    key: 'userIdHide',
  },
  {
    label: '年月日不展示',
    desc: '开启后，导出结果的日期将只显示几点几分(如果可能)',
    key: 'yearHide',
  },
  {
    label: '首行缩进对齐',
    desc: '开启后，缩进将以名字为基准进行对齐',
    key: 'textIndentAll',
  },
];
