import {
  ABOUT_CREDIT_SECTIONS,
  ABOUT_LINKS,
  buildAvatarUrl,
  buildContributorHref,
  getAboutOverviewSummary,
  resolveContributorUser,
} from './viewModel.js';
import { expect, it } from 'vitest';

it('规范版本信息和关于页链接', async () => {
  const assertIncludes = (actual: string, expected: string) => {
    if (!actual.includes(expected)) throw new Error(`expected ${actual} to include ${expected}`);
  };

  expect(ABOUT_LINKS.length).toBe(4);
  expect(ABOUT_LINKS[0]?.label).toBe('官方网站');
  expect(ABOUT_LINKS[0]?.href).toBe('https://dice.weizaima.com/');
  expect(ABOUT_LINKS[0]?.description).toBe('SealDice 项目主页，提供下载、在线试用与社区服务入口。');
  expect(ABOUT_LINKS[1]?.href).toBe('https://docs.sealdice.com/');
  expect(ABOUT_LINKS[3]?.href).toBe('https://github.com/sealdice/sealdice-core');
  assertIncludes(ABOUT_LINKS[3]?.description ?? '', 'Issue 反馈');

  expect(resolveContributorUser({ username: '木落' })).toBe('fy0');
  expect(resolveContributorUser({ username: '希望潇洒的风：Ceeling', user: 'charyflys' })).toBe(
    'charyflys'
  );
  expect(resolveContributorUser({ username: 'UnknownName' })).toBe('UnknownName');

  expect(buildContributorHref({ username: '[骰]Gaza(3658887052)', onlyName: true })).toBe('');
  expect(
    buildContributorHref({ username: '[骰]诺扣提', href: 'https://dice.weizaima.com/public-dice' })
  ).toBe('https://dice.weizaima.com/public-dice');
  expect(buildContributorHref({ username: 'JohNSoN' })).toBe('https://github.com/Xiangze-Li');

  assertIncludes(buildAvatarUrl({ username: '暮星' }), '/sd-api/utils/ga/MX-fox');
  expect(buildAvatarUrl({ username: '雪桃', src: 'https://example.com/a.png' })).toBe(
    'https://example.com/a.png'
  );

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

  expect(summary.appName).toBe('SealDice-CE');
  expect(summary.versionText).toBe('1.5.0-dev+20260521');
  expect(summary.latestVersionText).toBe('1.5.1');
  expect(summary.channelText).toBe('开发版');
  expect(summary.runtimeText).toBe('linux - amd64');
  expect(summary.modeText).toBe('容器模式');
  expect(summary.uptimeText).toBe('1 小时 1 分钟');
  expect(summary.hasNewVersion).toBe(true);
  expect(summary.containerMode).toBe(true);

  const fallbackSummary = getAboutOverviewSummary(undefined);
  expect(fallbackSummary.appName).toBe('SealDice');
  expect(fallbackSummary.versionText).toBe('读取中');
  expect(fallbackSummary.latestVersionText).toBe('读取中');
  expect(fallbackSummary.channelText).toBe('未知');
  expect(fallbackSummary.modeText).toBe('读取中');
  expect(fallbackSummary.uptimeText).toBe('读取中');

  expect(ABOUT_CREDIT_SECTIONS[0]?.title).toBe('社区协力');
  const v16Section = ABOUT_CREDIT_SECTIONS.find(section => section.title === 'V1.6 版本');
  expect(v16Section?.contributors?.length).toBe(15);
  expect(v16Section?.contributors?.some(contributor => contributor.user === 'Wesley-Young')).toBe(
    true
  );
  expect(v16Section?.contributors?.some(contributor => contributor.user === 'ShiaNyaa')).toBe(true);

  const v15Section = ABOUT_CREDIT_SECTIONS.find(section => section.title === 'V1.5 版本');
  expect(v15Section?.contributors?.length).toBe(19);
  expect(v15Section?.contributors?.some(contributor => contributor.user === 'GUERdugujian')).toBe(
    true
  );
  expect(v15Section?.contributors?.some(contributor => contributor.user === 'conglinyizhi')).toBe(
    true
  );
  expect(ABOUT_CREDIT_SECTIONS.some(section => section.title === 'V1.5 版本')).toBe(true);
  expect(ABOUT_CREDIT_SECTIONS.some(section => section.title === '参考')).toBe(true);
});
