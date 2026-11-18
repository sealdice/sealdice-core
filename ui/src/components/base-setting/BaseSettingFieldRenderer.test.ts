// @vitest-environment happy-dom
import { mount, type VueWrapper } from '@vue/test-utils';
import { afterEach, expect, it, vi } from 'vitest';
import { h, nextTick } from 'vue';
import { NDialogProvider, NInput, NInputNumber, NSwitch } from 'naive-ui';
import type {
  BaseSettingFieldModel,
  BaseSettingValueModel,
} from '@/features/baseSetting/viewModel';
import BaseSettingFieldRenderer from './BaseSettingFieldRenderer.vue';

let wrapper: VueWrapper | undefined;
afterEach(() => wrapper?.unmount());

function renderField(field: Partial<BaseSettingFieldModel>, model: Partial<BaseSettingValueModel>) {
  wrapper = mount(NDialogProvider, {
    slots: {
      default: () =>
        h(BaseSettingFieldRenderer, {
          field: {
            id: 'test',
            label: '测试设置',
            keys: [],
            keywords: [],
            options: [],
            kind: 'text',
            ...field,
          },
          model: model as BaseSettingValueModel,
          isContainerMode: false,
          runAction: vi.fn(),
        }),
    },
  });
  return wrapper.getComponent(BaseSettingFieldRenderer);
}

it('清空区间终点会同步修正起点，保证保存的区间仍然有效', async () => {
  const field = renderField(
    { kind: 'number-pair', keys: ['messageDelayRangeStart', 'messageDelayRangeEnd'] },
    { messageDelayRangeStart: 2, messageDelayRangeEnd: 5 }
  );
  field.findAllComponents(NInputNumber)[1]!.vm.$emit('update:value', null);
  await nextTick();
  expect(field.emitted('updateField')).toEqual([
    ['messageDelayRangeEnd', 0],
    ['messageDelayRangeStart', 0],
  ]);
});

it('依赖开关关闭时禁用目标开关并显示提示', () => {
  const field = renderField(
    {
      kind: 'boolean',
      key: 'officialQQUseNameCard',
      disabledWhen: { key: 'officialQQUseMarkdown', equals: false },
      disabledHint: '请先启用 Markdown。',
    },
    { officialQQUseMarkdown: false, officialQQUseNameCard: false }
  );

  expect(field.getComponent(NSwitch).props('disabled')).toBe(true);
  expect(field.text()).toContain('请先启用 Markdown。');
});

it('依赖开关开启时目标开关可用', () => {
  const field = renderField(
    {
      kind: 'boolean',
      key: 'officialQQUseNameCard',
      disabledWhen: { key: 'officialQQUseMarkdown', equals: false },
      disabledHint: '请先启用 Markdown。',
    },
    { officialQQUseMarkdown: true, officialQQUseNameCard: false }
  );

  expect(field.getComponent(NSwitch).props('disabled')).toBe(false);
});

it('名片渲染模板使用多行输入并渲染 Markdown 预览', () => {
  const field = renderField(
    {
      kind: 'textarea',
      key: 'officialQQUseNameCardTemplate',
      previewSample: '木落 HP10/10',
      previewBody: 'D100=50 成功',
      previewFallback: '<font color="#FF0000" size="1">{card}</font>\n\n',
    },
    {
      officialQQUseNameCardTemplate: '<font color="#FF0000" size="1">{card}</font>\n\n',
    }
  );

  expect(field.getComponent(NInput).props('type')).toBe('textarea');
  const preview = field.find('.name-card-template-preview');
  expect(preview.exists()).toBe(true);
  expect(preview.text()).toContain('木落 HP10/10');
  expect(preview.text()).toContain('D100=50 成功');
  const value = preview.find('.name-card-template-preview-value');
  expect(value.html()).toContain('<font color="#FF0000" size="1">木落 HP10/10</font>');
});

it('旧默认模板仅精确匹配时升级，带换行按原样预览', () => {
  const render = (value: string) =>
    renderField(
      {
        kind: 'textarea',
        key: 'officialQQUseNameCardTemplate',
        previewSample: '木落 HP10/10',
        previewBody: 'D100=50 成功',
        previewFallback: '<font color="#FF0000" size="1">{card}</font>\n\n',
      },
      { officialQQUseNameCardTemplate: value }
    ).find('.name-card-template-preview-value');

  // 精确等于旧默认模板：升级为带空行的新默认，名片与正文分成两段
  const upgraded = render('<font color="#FF0000" size="1">{card}</font>');
  expect(upgraded.findAll('p').length).toBe(2);

  // 旧默认模板 + 换行：运行时不升级，预览同样按单个换行渲染（同一段落内软换行）
  const asIs = render('<font color="#FF0000" size="1">{card}</font>\n');
  const paragraphs = asIs.findAll('p');
  expect(paragraphs.length).toBe(1);
  expect(paragraphs[0]!.text()).toContain('木落 HP10/10');
  expect(paragraphs[0]!.text()).toContain('D100=50 成功');
});

it('名片模板里的引用只包裹名片，不吞掉正文', () => {
  const field = renderField(
    {
      kind: 'textarea',
      key: 'officialQQUseNameCardTemplate',
      previewSample: '木落 HP10/10',
      previewBody: 'D100=50 成功',
      previewFallback: '> {card}\n\n',
    },
    {
      officialQQUseNameCardTemplate: '> {card}\n\n',
    }
  );

  const preview = field.find('.name-card-template-preview-value');
  const blockquote = preview.find('blockquote');
  expect(blockquote.exists()).toBe(true);
  expect(blockquote.text()).toContain('木落 HP10/10');
  expect(blockquote.text()).not.toContain('D100=50 成功');
  expect(preview.text()).toContain('D100=50 成功');
});

it('关闭需确认的开关时，只有用户确认后才修改草稿', async () => {
  const field = renderField(
    { kind: 'boolean', key: 'logSizeNoticeEnable', confirmMessage: '确认关闭日志提示？' },
    { logSizeNoticeEnable: true }
  );
  await field.getComponent(NSwitch).trigger('click');
  expect(field.emitted('updateField')).toBeUndefined();
  const cancel = Array.from(document.querySelectorAll('button')).find(
    button => button.textContent === '取消'
  );
  expect(cancel).toBeDefined();
  cancel!.click();
  await nextTick();
  expect(field.emitted('updateField')).toBeUndefined();

  await field.getComponent(NSwitch).trigger('click');
  const confirm = Array.from(document.querySelectorAll('button')).find(
    button => button.textContent === '确定'
  );
  expect(confirm).toBeDefined();
  confirm!.click();
  await nextTick();
  expect(field.emitted('updateField')).toEqual([['logSizeNoticeEnable', false]]);
});
