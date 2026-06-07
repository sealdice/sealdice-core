import {
  ABOUT_CREDIT_SECTIONS,
  ABOUT_LINKS,
  buildAvatarUrl,
  buildContributorHref,
  getAboutOverviewSummary,
  resolveContributorUser,
} from './viewModel.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertIncludes = (actual: string, expected: string) => {
  if (!actual.includes(expected)) throw new Error(`expected ${actual} to include ${expected}`);
};

assertEqual(ABOUT_LINKS.length, 4);
assertEqual(ABOUT_LINKS[0]?.label, '官方网站');
assertEqual(ABOUT_LINKS[0]?.href, 'https://github.com/sealdice-ce/sealdice-ce');

assertEqual(resolveContributorUser({ username: '木落' }), 'fy0');
assertEqual(resolveContributorUser({ username: '希望潇洒的风：Ceeling', user: 'charyflys' }), 'charyflys');
assertEqual(resolveContributorUser({ username: 'UnknownName' }), 'UnknownName');

assertEqual(buildContributorHref({ username: '[骰]Gaza(3658887052)', onlyName: true }), '');
assertEqual(buildContributorHref({ username: '[骰]诺扣提', href: 'https://dice.weizaima.com/public-dice' }), 'https://dice.weizaima.com/public-dice');
assertEqual(buildContributorHref({ username: 'JohNSoN' }), 'https://github.com/Xiangze-Li');

assertIncludes(buildAvatarUrl({ username: '暮星' }), '/sd-api/utils/ga/MX-fox');
assertEqual(buildAvatarUrl({ username: '雪桃', src: 'https://example.com/a.png' }), 'https://example.com/a.png');

const summary = getAboutOverviewSummary({
  appName: 'SealDice-CE',
  appChannel: 'dev',
  version: {
    value: '1.5.0-dev+20260521',
    simple: '1.5.0-dev',
    code: 10500,
    latest: '1.5.1',
    latestNote: 'bugfix',
    latestCode: 10501,
  },
  runtime: {
    uptime: 3661,
    OS: 'linux',
    arch: 'amd64',
    justForTest: false,
    containerMode: true,
  },
});

assertEqual(summary.appName, 'SealDice-CE');
assertEqual(summary.versionText, '1.5.0-dev+20260521');
assertEqual(summary.latestVersionText, '1.5.1');
assertEqual(summary.channelText, '开发版');
assertEqual(summary.runtimeText, 'linux - amd64');
assertEqual(summary.uptimeText, '1 小时 1 分钟');
assertEqual(summary.hasNewVersion, true);
assertEqual(summary.containerMode, true);

const fallbackSummary = getAboutOverviewSummary(undefined);
assertEqual(fallbackSummary.appName, 'SealDice-CE');
assertEqual(fallbackSummary.versionText, '读取中');
assertEqual(fallbackSummary.latestVersionText, '读取中');
assertEqual(fallbackSummary.channelText, '未知');
assertEqual(fallbackSummary.uptimeText, '读取中');

assertEqual(ABOUT_CREDIT_SECTIONS[0]?.title, '社区协力');
assertEqual(ABOUT_CREDIT_SECTIONS.some(section => section.title === 'V1.5 版本'), true);
assertEqual(ABOUT_CREDIT_SECTIONS.some(section => section.title === '参考'), true);
