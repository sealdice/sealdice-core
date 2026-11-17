// @vitest-environment happy-dom
import { mount } from '@vue/test-utils';
import { expect, it } from 'vitest';
import type { TextTemplateItem } from '@/features/customText/types';
import CustomTextEntryCard from './CustomTextEntryCard.vue';

it('条目编辑只发出操作事件，不修改父级传入的文案数组', async () => {
  const items: TextTemplateItem[] = [
    ['first', 1],
    ['second', 2],
  ];
  const wrapper = mount(CustomTextEntryCard, {
    props: { category: 'core', keyName: 'reply', items },
  });
  try {
    await wrapper.findAll('textarea')[0]!.setValue('edited');
    expect(wrapper.emitted('updateItem')?.at(-1)).toEqual(['reply', 0, 'edited']);
    await wrapper.findAll('[aria-label="删除随机文案"]')[0]!.trigger('click');
    expect(wrapper.emitted('removeItem')?.at(-1)).toEqual(['reply', 0]);
    expect(items).toEqual([
      ['first', 1],
      ['second', 2],
    ]);
  } finally {
    wrapper.unmount();
  }
});
