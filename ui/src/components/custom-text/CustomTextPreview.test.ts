// @vitest-environment happy-dom
import { mount } from '@vue/test-utils';
import { afterEach, expect, it, vi } from 'vitest';
import type { TextItemCompatibleInfo } from '@/api';
import * as markdown from '@/features/customText/markdown';
import CustomTextPreview from './CustomTextPreview.vue';

afterEach(() => vi.restoreAllMocks());

function observeImages() {
  const images: HTMLImageElement[] = [];
  const createElement = document.createElement.bind(document);
  vi.spyOn(document, 'createElement').mockImplementation((tagName, options) => {
    const element = createElement(tagName, options);
    if (tagName === 'img') images.push(element as HTMLImageElement);
    return element;
  });
  return images;
}
const sample: TextItemCompatibleInfo = {
  version: 'v2',
  textV1: '**V1 结果**',
  textV2: '__V2 结果__',
  errV1: '',
  errV2: '',
  presetExists: true,
};

it.each(['V1', 'V2'])('%s 图片默认占位，仅点击该图片时请求并保留尺寸', async version => {
  const images = observeImages();
  const text = '![示例 #320px #200px](https://example.com/test.png)';
  const wrapper = mount(CustomTextPreview, {
    props: { info: { ...sample, textV1: text, textV2: text } },
  });
  try {
    expect(wrapper.find('img').exists()).toBe(false);
    expect(images).toHaveLength(0);
    const region = wrapper.get(`[aria-label="${version} Markdown 预览"]`);
    const button = region.get('button');
    expect(button.attributes('aria-label')).toBe('加载图片：示例');
    await button.get('span').trigger('click');
    expect(images).toHaveLength(1);
    expect(images[0]!.src).toBe('https://example.com/test.png');
    expect(images[0]!.referrerPolicy).toBe('no-referrer');
    expect(button.attributes('disabled')).toBeDefined();
    await button.trigger('click');
    expect(images).toHaveLength(1);
    images[0]!.dispatchEvent(new Event('load'));
    expect(region.get('img').attributes()).toMatchObject({
      alt: '示例',
      width: '320',
      height: '200',
    });
    const other = version === 'V1' ? 'V2' : 'V1';
    expect(wrapper.get(`[aria-label="${other} Markdown 预览"]`).find('img').exists()).toBe(false);
    await wrapper.setProps({
      info: {
        ...sample,
        textV1: text,
        textV2: text,
        [version === 'V1' ? 'textV1' : 'textV2']: '![新图片](https://example.com/new.png)',
      },
    });
    expect(region.find('img').exists()).toBe(false);
    expect(region.get('button').attributes('data-image-src')).toBe('https://example.com/new.png');
    expect(images).toHaveLength(1);
  } finally {
    wrapper.unmount();
  }
});

it('图片请求失败后保留占位并允许重试，点击链接中的图片不会跳转', async () => {
  const images = observeImages();
  const wrapper = mount(CustomTextPreview, {
    props: {
      info: { ...sample, textV2: '[![示例](https://example.com/a.png)](https://example.com)' },
    },
  });
  try {
    const button = wrapper.get('[aria-label="V2 Markdown 预览"] button');
    const click = new MouseEvent('click', { bubbles: true, cancelable: true });
    button.element.dispatchEvent(click);
    expect(click.defaultPrevented).toBe(true);
    expect(images).toHaveLength(1);
    images[0]!.dispatchEvent(new Event('error'));
    expect(button.text()).toBe('图片加载失败，点击重试');
    expect(button.attributes('disabled')).toBeUndefined();
    await button.trigger('click');
    expect(images).toHaveLength(2);
    images[1]!.dispatchEvent(new Event('load'));
    expect(wrapper.get('[aria-label="V2 Markdown 预览"] img').attributes('src')).toBe(
      'https://example.com/a.png'
    );
  } finally {
    wrapper.unmount();
  }
});

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
  const render = vi.spyOn(markdown, 'renderMarkdownIfFormatted');
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
    expect(render).not.toHaveBeenCalledWith(version === 'v1' ? info.textV1 : info.textV2);
  } finally {
    wrapper.unmount();
  }
});

it('格式化成功时保留原文并渲染有效的盒装公式', () => {
  const text = '$$\\boxed{42/60}$$';
  const wrapper = mount(CustomTextPreview, {
    props: { info: { ...sample, textV2: text } },
  });
  try {
    expect(wrapper.get('[aria-label="V2 原文预览"]').text()).toBe(text);
    expect(wrapper.find('[aria-label="V2 Markdown 预览"] .katex-display').exists()).toBe(true);
    expect(wrapper.find('.katex-error').exists()).toBe(false);
  } finally {
    wrapper.unmount();
  }
});

it('格式化成功但公式语法无效时显示公式错误并保留原文与其他 Markdown', () => {
  const text = '**示例**\n\n$$\\frac{1}$$';
  const wrapper = mount(CustomTextPreview, {
    props: { info: { ...sample, textV2: text } },
  });
  try {
    expect(wrapper.get('[aria-label="V2 原文预览"]').text()).toBe(text);
    expect(wrapper.get('[aria-label="V2 Markdown 预览"] .katex-error').text()).toContain(
      '\\frac{1}'
    );
    expect(wrapper.get('[aria-label="V2 Markdown 预览"] strong').text()).toBe('示例');
    expect(wrapper.get('[aria-label="V1 Markdown 预览"] strong').text()).toBe('V1 结果');
  } finally {
    wrapper.unmount();
  }
});

it.each(['v1', 'v2'])(
  '%s Markdown 渲染器异常不影响原文和另一版预览，更新后可恢复',
  async version => {
    const failingText = '**示例渲染异常**';
    const render = markdown.renderMarkdownIfFormatted;
    vi.spyOn(markdown, 'renderMarkdownIfFormatted').mockImplementation(text => {
      if (text === failingText) throw new Error('Synthetic rendering failure');
      return render(text);
    });
    const info = { ...sample, [version === 'v1' ? 'textV1' : 'textV2']: failingText };
    const wrapper = mount(CustomTextPreview, { props: { info } });
    try {
      const label = version === 'v1' ? 'V1' : 'V2';
      const other = version === 'v1' ? 'V2' : 'V1';
      expect(wrapper.get(`[aria-label="${label} 原文预览"]`).text()).toBe(failingText);
      expect(wrapper.get(`[aria-label="${label} Markdown 预览"] [role="status"]`).text()).toContain(
        'Markdown 渲染失败'
      );
      expect(wrapper.get(`[aria-label="${other} Markdown 预览"] strong`).text()).toBe(
        `${other} 结果`
      );
      await wrapper.setProps({ info: sample });
      expect(wrapper.get(`[aria-label="${label} Markdown 预览"] strong`).text()).toBe(
        `${label} 结果`
      );
      expect(wrapper.find('[role="status"]').exists()).toBe(false);
    } finally {
      wrapper.unmount();
    }
  }
);
