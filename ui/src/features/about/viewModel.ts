export type AboutLink = {
  label: string;
  href: string;
  description: string;
};

export type AboutContributor = {
  username: string;
  user?: string;
  src?: string;
  onlyName?: boolean;
  href?: string;
  info?: string;
};

export type AboutCreditLine = {
  text: string;
  linkText?: string;
  href?: string;
  tail?: string;
};

export type AboutCreditSection = {
  title: string;
  contributors?: AboutContributor[];
  lines?: AboutCreditLine[];
};

export type AboutOverviewInput = {
  appName?: string;
  appChannel?: string;
  version?: {
    value?: string;
    simple?: string;
    code?: number;
    latest?: string;
    latestNote?: string;
    latestCode?: number;
  };
  runtime?: {
    uptime?: number;
    OS?: string;
    arch?: string;
    justForTest?: boolean;
    containerMode?: boolean;
  };
};

export type AboutOverviewSummary = {
  appName: string;
  versionText: string;
  latestVersionText: string;
  latestNote: string;
  channelText: string;
  runtimeText: string;
  uptimeText: string;
  hasNewVersion: boolean;
  containerMode: boolean;
  justForTest: boolean;
};

const contributorUsers = new Map<string, string>([
  ['木落', 'fy0'],
  ['逐风', 'MintCider'],
  ['暮星', 'MX-fox'],
  ['JohNSoN', 'Xiangze-Li'],
  ['Bugtower100', 'bugtower100'],
  ['Bugtower', 'bugtower100'],
  ['只是另一个 ID', 'JustAnotherID'],
  ['檀轶步棋', 'oissevalt'],
  ['脑', 'f44r'],
]);

export const ABOUT_LINKS: AboutLink[] = [
  {
    label: '官方网站',
    href: 'https://github.com/sealdice-ce/sealdice-ce',
    description: '社区版项目主页、发布信息与议题反馈入口。',
  },
  {
    label: '使用手册',
    href: 'https://dice.weizaima.com/manual/',
    description: '指令、部署、扩展与常见问题文档。',
  },
  {
    label: '投喂海豹',
    href: 'https://dice.weizaima.com/feed/',
    description: '支持项目持续维护与基础设施开销。',
  },
  {
    label: '源码',
    href: 'https://github.com/sealdice/sealdice-core',
    description: '核心仓库源码与开发进度。',
  },
];

export const ABOUT_CREDIT_SECTIONS: AboutCreditSection[] = [
  {
    title: '社区协力',
    lines: [
      {
        text: '特别鸣谢参与测试、反馈问题，帮助完善海豹指令的各位。以下名单排名不分先后。',
      },
    ],
  },
  {
    title: 'V1.5 版本',
    contributors: [
      { username: '木落' },
      { username: 'PaienNate' },
      { username: 'Fripine' },
      { username: '只是另一个 ID' },
      { username: '逐风' },
      { username: '暮星' },
      { username: 'JohNSoN' },
      { username: 'Bugtower100' },
      { username: '希望潇洒的风：Ceeling', info: 'UI', user: 'charyflys' },
    ],
  },
  {
    title: 'V1.5.0 特别致谢',
    contributors: [
      { username: '雪桃', user: 'LoranaAurelia', info: '提供新的魔法阵代理' },
      {
        username: '[骰]诺扣提',
        src: 'https://d1.sealdice.com/images/bird.jpg',
        href: 'https://dice.weizaima.com/public-dice',
        info: '公骰 (协助测试)',
      },
      {
        username: '山本健一',
        user: 'kenichiLyon',
        src: 'https://d1.sealdice.com/images/kenichiLyon.jpg',
        info: '协助测试',
      },
      { username: '白鱼', user: 'baiyu-yu', info: '协助测试' },
      {
        username: '[骰]Gaza(3658887052)',
        src: 'https://firehomework.top/img/6819CBB71F64D7BBFC90DCF4432B76BC.jpg',
        info: '协助测试',
        onlyName: true,
      },
    ],
  },
  {
    title: 'V1.4 版本',
    contributors: [
      { username: '只是另一个 ID' },
      { username: 'Szzrain' },
      { username: 'JohNSoN' },
      { username: '檀轶步棋' },
      { username: '脑' },
      { username: '木落' },
      { username: 'Director259' },
      { username: '浣熊旅記', user: 'VolEurr0Se', info: 'dnd5文档' },
      { username: '调零', user: 'zeroxilo', info: 'dnd5文档' },
      { username: '凤吹风雪', user: 'cherrybird7', info: '7版怪锤' },
      { username: '奈亚猫猫汉化组', info: '7版怪锤', onlyName: true },
      { username: '稚鸢音', info: '7版怪锤', onlyName: true },
      { username: 'Fripine' },
      { username: '逐风' },
      { username: '炽热', user: 'yichere' },
      { username: '暮星' },
      { username: '0.018', user: '0018cha' },
      { username: '冷筱华', user: 'shakugannosaints' },
    ],
  },
  {
    title: 'V1.4 安卓端',
    contributors: [
      { username: 'Szzrain' },
      { username: 'PaienNate' },
      { username: '木落' },
    ],
  },
  {
    title: 'V1.4.5 特别致谢',
    contributors: [
      { username: 'Linwenxuan', user: 'Linwenxuan04' },
    ],
  },
  {
    title: 'V1.3 版本',
    contributors: [
      { username: '只是另一个 ID' },
      { username: 'Szzrain' },
      { username: 'JohNSoN' },
      { username: '檀轶步棋' },
      { username: '脑' },
      { username: '木落' },
    ],
  },
  {
    title: 'V1.3 安卓端',
    contributors: [
      { username: 'Szzrain' },
      { username: 'PaienNate' },
    ],
  },
  {
    title: 'V1.2 版本',
    contributors: [
      { username: '木落' },
      { username: 'Szzrain' },
      { username: '于言诺', user: 'yuyannuo' },
      { username: '檀轶步棋' },
      { username: '脑' },
      { username: '熊米', user: 'SunnyJoyce' },
      { username: '浣熊旅記', user: 'VolEurr0Se' },
      { username: '流溪', user: 'lxy071130' },
      { username: '病', user: 'nodisease' },
    ],
  },
  {
    title: 'V1.2 安卓端',
    contributors: [
      { username: '木末君', user: '96368a' },
      { username: 'Szzrain' },
      { username: '极夜幻想', user: 'JiYeHuanXiang' },
    ],
  },
  {
    title: 'V1.1 版本',
    lines: [
      { text: 'Szzrain - 实现了 Discord 和 Kook(开黑啦) 两个平台的海豹接入' },
      {
        text: '于言诺 - 制作了很多海豹扩展和牌堆，如养猫、踢海豹、赛博功德、风味月饼、万圣节糖果等等。协助撰写了一些海豹的文档和教程，并找出了众多海豹的 bug',
      },
      { text: '云陌 - 海豹文档教程协力，同时也找了很多海豹的 bug' },
      {
        text: '',
        linkText: '星尘',
        href: 'https://github.com/kagangtuya-star',
        tail: ' - 编写了海豹同网络登录的教程，友情提供了用于指令参考的 fvtt，以及一些建议和 bug 反馈',
      },
    ],
  },
  {
    title: 'V1.0版本',
    lines: [
      { text: 'Ariel船长 - 早期测试参与者，协助解决登录流程问题' },
      { text: 'Raycel - 早期测试参与者，协助解决登录流程问题' },
      { text: 'kuma - 早期测试参与者，海豹的第一次全指令全流程测试' },
      { text: '卟啵 - 早期测试参与者，回报了中文路径和空格路径问题，协助解决了登录流程问题' },
      { text: '蜜瓜包 - 早期测试参与者，默认文档中“怪物之锤查询”的编纂者之一' },
      { text: '月森优姬 - 早期测试参与者，提出了大量各种各样建议和 BUG 反馈，纠正了一些与规则书不统一的问题，COC 同义词和默认技能点数的编纂者' },
      { text: '清茶 - 在 4 月 7 日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令' },
      { text: '脑 - 在 4 月 7 日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令' },
      { text: 'Greed锦鲤 - 在 4 月 7 日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令' },
      { text: '格莱德 - 在 4 月 7 日的可靠性测试中，参与构造了让旧版海豹进程崩溃的指令' },
      { text: '我来逛街 - 提出很多建议；帮助改进了 DND5E 同义词列表，增加许多常用说法' },
    ],
  },
  {
    title: '手册编写',
    contributors: [
      { username: '只是另一个 ID' },
      { username: 'JohNSoN' },
      { username: '暮星' },
      { username: 'Szzrain' },
      { username: '木落' },
      {
        username: '山本健一',
        user: 'kenichiLyon',
        src: 'https://d1.sealdice.com/images/kenichiLyon.jpg',
      },
      { username: '逐风' },
      { username: '脑' },
      { username: 'Bugtower', user: 'bugtower100' },
      { username: 'Monad', user: 'Mitxoleta' },
      { username: '以炽热挥剑', user: 'yichirehuijian' },
      { username: '檀轶步棋' },
      { username: '流溪', user: 'lxy071130' },
      { username: 'PaienNate' },
      { username: '病', user: 'nodisease' },
      { username: '綮小灰', user: 'Stanlty998' },
      { username: '大概不全，下版本纠正……', onlyName: true },
    ],
  },
  {
    title: '参考',
    lines: [
      { text: '斯塔尼亚 - 塔系核心作者，指令实现过程中部分参考了塔系核心的指令表现' },
      { text: '赵喵喵 - ZhaoDice 作者，主要指令参考之一' },
      { text: 'Dice! 核心的开发者们 - 同样的，在骰点格式和输出表现方面进行了参考' },
      { text: 'FVTT - 经典的 DND 跑团平台，指令参考之一' },
    ],
  },
];

export function resolveContributorUser(contributor: AboutContributor): string {
  return contributor.user ?? contributorUsers.get(contributor.username) ?? contributor.username;
}

export function buildContributorHref(contributor: AboutContributor): string {
  if (contributor.onlyName) return '';
  if (contributor.href) return contributor.href;
  return `https://github.com/${resolveContributorUser(contributor)}`;
}

export function buildAvatarUrl(contributor: AboutContributor): string {
  if (contributor.src) return contributor.src;
  return `/sd-api/utils/ga/${encodeURIComponent(resolveContributorUser(contributor))}`;
}

export function getAboutOverviewSummary(overview: AboutOverviewInput | undefined): AboutOverviewSummary {
  const version = overview?.version;
  const runtime = overview?.runtime;

  return {
    appName: cleanText(overview?.appName, 'SealDice-CE'),
    versionText: cleanText(version?.value || version?.simple, '读取中'),
    latestVersionText: cleanText(version?.latest, '读取中'),
    latestNote: cleanText(version?.latestNote),
    channelText: formatChannel(overview?.appChannel),
    runtimeText: runtime?.OS && runtime.arch ? `${runtime.OS} - ${runtime.arch}` : '读取中',
    uptimeText: formatUptime(runtime?.uptime),
    hasNewVersion: Number(version?.code ?? 0) < Number(version?.latestCode ?? 0),
    containerMode: runtime?.containerMode === true,
    justForTest: runtime?.justForTest === true,
  };
}

function formatChannel(channel: string | undefined): string {
  if (channel === 'stable') return '稳定版';
  if (channel === 'dev') return '开发版';
  if (channel === 'nightly') return '每日构建';
  return channel ? channel : '未知';
}

function formatUptime(seconds: number | undefined): string {
  if (!Number.isFinite(seconds) || seconds === undefined || seconds < 0) return '读取中';

  const total = Math.trunc(seconds);
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  if (days > 0) return `${days} 天 ${hours} 小时`;
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`;
  if (minutes > 0) return `${minutes} 分钟`;
  return `${total} 秒`;
}

function cleanText(value: unknown, fallback = ''): string {
  return typeof value === 'string' && value.trim() ? value.trim() : fallback;
}
