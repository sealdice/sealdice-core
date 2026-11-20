// @vitest-environment happy-dom
import { mount } from '@vue/test-utils';
import { expect, it } from 'vitest';
import type { TextItemCompatibleInfo } from '@/api';
import CustomTextPreview from './CustomTextPreview.vue';
const sample: TextItemCompatibleInfo = {
  version: 'v2',
  textV1: '**V1 结果**',
  textV2: '__V2 结果__',
  errV1: '',
  errV2: '',
  presetExists: true,
};

it.each(['v1', 'v2'])(
  '选用 %s 时仍按 V2 原文、V2 Markdown、V1 原文、V1 Markdown 的顺序显示两版效果',
  version => {
    const wrapper = mount(CustomTextPreview, { props: { info: { ...sample, version } } });
    try {
      const regions = wrapper
        .findAll('[role="region"]:not([aria-label="文案效果预览"])')
        .map(region => region.attributes('aria-label'));
      expect(regions).toEqual([
        'V2 原文预览',
        'V2 Markdown 预览',
        'V1 原文预览',
        'V1 Markdown 预览',
      ]);
      expect(wrapper.get('[aria-label="V2 原文预览"]').text()).toBe(sample.textV2);
      expect(wrapper.get('[aria-label="V1 原文预览"]').text()).toBe(sample.textV1);
      expect(wrapper.get('[aria-label="V2 Markdown 预览"] strong').text()).toBe('V2 结果');
      expect(wrapper.get('[aria-label="V1 Markdown 预览"] strong').text()).toBe('V1 结果');
    } finally {
      wrapper.unmount();
    }
  }
);

it('没有匹配的预览时不显示旧结果，保存更新后同步两版效果', async () => {
  const wrapper = mount(CustomTextPreview, { props: { info: sample } });
  try {
    await wrapper.setProps({ info: undefined });
    expect(wrapper.text()).toContain('保存文案后');
    expect(wrapper.text()).not.toContain('V2 结果');
    await wrapper.setProps({ info: { ...sample, textV2: '**保存后的结果**' } });
    expect(wrapper.get('[aria-label="V2 Markdown 预览"] strong').text()).toBe('保存后的结果');
    expect(wrapper.get('[aria-label="V1 Markdown 预览"] strong').text()).toBe('V1 结果');
  } finally {
    wrapper.unmount();
  }
});

it('求值错误只在原文显示，另一版 Markdown 预览正常', () => {
  const wrapper = mount(CustomTextPreview, {
    props: { info: { ...sample, textV1: '', errV1: 'V1 错误' } },
  });
  try {
    expect(wrapper.get('[aria-label="V1 原文预览"]').text()).toBe('V1 错误');
    expect(wrapper.find('[aria-label="V1 Markdown 预览"]').exists()).toBe(false);
    expect(wrapper.get('[aria-label="V2 Markdown 预览"] strong').text()).toBe('V2 结果');
  } finally {
    wrapper.unmount();
  }
});

it('普通文本不额外展示 Markdown 行，按版本独立判断', async () => {
  const wrapper = mount(CustomTextPreview, {
    props: { info: { ...sample, textV1: 'V1 成功', textV2: 'V2 成功' } },
  });
  try {
    expect(wrapper.find('[aria-label="V1 Markdown 预览"]').exists()).toBe(false);
    expect(wrapper.find('[aria-label="V2 Markdown 预览"]').exists()).toBe(false);
    expect(wrapper.get('[aria-label="V1 原文预览"]').text()).toBe('V1 成功');
    expect(wrapper.get('[aria-label="V2 原文预览"]').text()).toBe('V2 成功');
    await wrapper.setProps({ info: { ...sample, textV1: 'V1 成功' } });
    expect(wrapper.find('[aria-label="V1 Markdown 预览"]').exists()).toBe(false);
    expect(wrapper.get('[aria-label="V2 Markdown 预览"] strong').text()).toBe('V2 结果');
  } finally {
    wrapper.unmount();
  }
});

it.each(['v1', 'v2'])('%s 格式化报错时不渲染错误信息中包含的 Markdown', version => {
  const info = { ...sample };
  if (version === 'v1') {
    info.textV1 = '格式化错误V1:"**文案中的语法**"';
    info.errV1 = '格式化错误V1';
  } else {
    info.textV2 = '格式化错误V2:"**文案中的语法**"';
    info.errV2 = '格式化错误V2';
  }
  const wrapper = mount(CustomTextPreview, { props: { info } });
  try {
    const label = version === 'v1' ? 'V1' : 'V2';
    expect(wrapper.find(`[aria-label="${label} Markdown 预览"]`).exists()).toBe(false);
    expect(wrapper.get(`[aria-label="${label} 原文预览"]`).text()).toContain('格式化错误');
    const other = version === 'v1' ? 'V2' : 'V1';
    expect(wrapper.get(`[aria-label="${other} Markdown 预览"] strong`).text()).toBe(
      `${other} 结果`
    );
  } finally {
    wrapper.unmount();
  }
});
