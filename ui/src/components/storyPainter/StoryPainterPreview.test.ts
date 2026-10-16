// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, effectScope, h } from 'vue';
import { readFileSync } from 'node:fs';
import StoryPainterPreview from './StoryPainterPreview.vue';
import { useStoryPainter } from '@/features/storyPainter/useStoryPainter';
import { collectStoryPainterForumText } from '@/features/storyPainter/textExport';
import {
  createStoryPainterParquetDataset,
  type StoryPainterParquetDataset,
} from '@/features/storyPainter/parquetDataset';
import type { StoryPainterLogItem } from '@/features/storyPainter/types';

const scopes: ReturnType<typeof effectScope>[] = [];
afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
  scopes.splice(0).forEach(scope => scope.stop());
});

it('从 Parquet 直接打开多行论坛，预览与复制、导出一致', async () => {
  vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(600);
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(800);
  const dataset = await createStoryPainterParquetDataset(
    new Blob([readFileSync('src/features/storyPainter/testdata/consecutive.parquet')])
  );
  const scope = effectScope();
  scopes.push(scope);
  const painter = scope.run(useStoryPainter)!;
  await painter.setDataset(dataset);
  painter.mode.value = 'bbspineapple';
  const copyText = () =>
    collectStoryPainterForumText({
      chunks: painter.iterPreviewChunks(1),
      chars: painter.chars.value,
      exportOptions: painter.exportOptions,
      forumOptions: painter.forumOptions,
      pineapple: true,
      colorByItem: painter.colorByItem,
    });
  const wrapper = mount(
    defineComponent({
      setup: () => () =>
        h(StoryPainterPreview, {
          mode: painter.mode.value === 'editor' ? 'preview' : painter.mode.value,
          items: painter.previewItems.value,
          readItems: painter.readPreviewItems,
          chars: painter.chars.value,
          options: painter.exportOptions,
          forumOptions: painter.forumOptions,
          addVoiceMark: false,
          copyText,
        }),
    }),
    {
      global: {
        stubs: {
          NFlex: { template: '<div><slot /></div>' },
          NButton: { template: '<button><slot /></button>' },
          NCheckbox: true,
          NIcon: true,
          NEmpty: true,
        },
      },
    }
  );
  try {
    await flushPromises();
    const expected =
      '[color=silver]<Alice>[/color][color=#db2777] 第一句\n第二句 [/color]\n[color=silver]<Bob>[/color][color=#ea580c] 接话 [/color]';
    expect(await copyText()).toBe(expected);
    await vi.waitFor(() =>
      expect(
        wrapper
          .findAll('pre')
          .map(row => row.text())
          .join('\n')
      ).toBe(expected)
    );
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(wrapper.findComponent(StoryPainterPreview).emitted('copy')).toEqual([[expected]]);

    painter.renameChar(0, 'Alicia');
    painter.updateChar(0, { color: '#0284c7', role: '主持人' });
    painter.forumOptions.bbsUseColorName = true;
    const renamed = expected
      .replace(/Alice/g, 'Alicia')
      .replace('#db2777', 'blue')
      .replace('#ea580c', 'orange');
    await vi.waitFor(() =>
      expect(
        wrapper
          .findAll('pre')
          .map(row => row.text())
          .join('\n')
      ).toBe(renamed)
    );
    expect(await copyText()).toBe(renamed);
    painter.mode.value = 'preview';
    await vi.waitFor(() =>
      expect(wrapper.findAll('._nickname').map(row => row.text())).toEqual([
        '<Alicia>:',
        '<Alicia>:',
        '<Bob>:',
      ])
    );
    painter.mode.value = 'bbspineapple';
    painter.deleteChar(1);
    const deleted = renamed.split('\n[color=silver]<Bob>')[0]!;
    await vi.waitFor(() =>
      expect(
        wrapper
          .findAll('pre')
          .map(row => row.text())
          .join('\n')
      ).toBe(deleted)
    );
    expect(await copyText()).toBe(deleted);
    painter.setAllItemsFromEditor([{ ...painter.items.value[0]!, message: '修改后的正文' }]);
    const edited = '[color=silver]<Alicia>[/color][color=blue] 修改后的正文 [/color]';
    await vi.waitFor(() => expect(wrapper.get('pre').text()).toBe(edited));
    expect(await copyText()).toBe(edited);
  } finally {
    wrapper.unmount();
  }
});

it('多行论坛滚动跨读取块时保留发言边界，返回前部时显示最新改名', async () => {
  vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockImplementation(function (
    this: HTMLElement
  ) {
    return this.classList.contains('preview-list') ? 280 : 28;
  });
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(800);
  const rows: StoryPainterLogItem[] = Array.from({ length: 260 }, (_, index) => ({
    id: index + 1,
    index,
    nickname: index === 259 ? 'Bob' : 'Alice',
    IMUserId: index === 259 ? '1002' : '1001',
    message: `正文-${index}`,
    time: 10,
    isDice: false,
    commandId: 0,
  }));
  const dataset: StoryPainterParquetDataset = {
    numRows: rows.length,
    readRows: vi.fn(async (start, end, columns) =>
      rows.slice(start, end).map(row => (columns ? { ...row, message: '' } : row))
    ),
    async *iterRows() {
      for (let start = 0; start < rows.length; start += 120) yield rows.slice(start, start + 120);
    },
  };
  const scope = effectScope();
  scopes.push(scope);
  const painter = scope.run(useStoryPainter)!;
  await painter.setDataset(dataset);
  painter.mode.value = 'bbspineapple';
  const wrapper = mount(
    defineComponent({
      setup: () => () =>
        h(StoryPainterPreview, {
          mode: 'bbspineapple',
          items: painter.previewItems.value,
          readItems: painter.readPreviewItems,
          chars: painter.chars.value,
          options: painter.exportOptions,
          forumOptions: painter.forumOptions,
          addVoiceMark: false,
          copyText: async () => '',
        }),
    }),
    {
      global: { stubs: { NFlex: true, NCheckbox: true, NButton: true, NIcon: true, NEmpty: true } },
    }
  );
  try {
    await vi.waitFor(() => expect(wrapper.text()).toContain('正文-0'));
    expect(wrapper.findAll('pre').length).toBeLessThan(100);
    const list = wrapper.get('.preview-list');
    list.element.scrollTop = 125 * 28;
    await list.trigger('scroll');
    await vi.waitFor(() => expect(wrapper.find('[data-index="125"] pre').text()).toBe('正文-125'));
    expect(wrapper.find('[data-index="120"] pre').text()).toBe('正文-120');
    expect(wrapper.find('[data-index="0"]').exists()).toBe(false);
    list.element.scrollTop = 250 * 28;
    await list.trigger('scroll');
    await vi.waitFor(() =>
      expect(wrapper.find('[data-index="259"] pre').text()).toBe(
        '[color=silver]<Bob>[/color][color=#ea580c] 正文-259 [/color]'
      )
    );
    expect(wrapper.find('[data-index="258"] pre').text()).toBe('正文-258 [/color]');
    painter.renameChar(0, 'Alicia');
    list.element.scrollTop = 0;
    await list.trigger('scroll');
    await vi.waitFor(() =>
      expect(wrapper.find('[data-index="0"] pre').text()).toBe(
        '[color=silver]<Alicia>[/color][color=#db2777] 正文-0'
      )
    );
    expect(
      vi
        .mocked(dataset.readRows)
        .mock.calls.filter(call => !call[2])
        .every(([start, end]) => end - start <= 120)
    ).toBe(true);
  } finally {
    wrapper.unmount();
  }
});
