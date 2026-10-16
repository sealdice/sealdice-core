// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import StoryPainterPreview from './StoryPainterPreview.vue';
import { useStoryPainter } from '@/features/storyPainter/useStoryPainter';
import { collectStoryPainterForumText } from '@/features/storyPainter/textExport';
import type { StoryPainterLogItem } from '@/features/storyPainter/types';
import type { StoryPainterParquetDataset } from '@/features/storyPainter/parquetDataset';

afterEach(() => vi.restoreAllMocks());

it('从 Parquet 直接打开多行论坛，预览与复制、导出一致', async () => {
  vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(600);
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(800);
  const rows: StoryPainterLogItem[] = ['第一句', '第二句', '接话'].map((message, index) => ({
    id: index + 1,
    index,
    nickname: index === 2 ? 'Bob' : 'Alice',
    IMUserId: index === 2 ? '1002' : '1001',
    message,
    time: 10,
    isDice: false,
    commandId: 0,
  }));
  const dataset: StoryPainterParquetDataset = {
    numRows: rows.length,
    async readRows(start, end, columns) {
      return rows.slice(start, end).map(row => (columns ? { ...row, message: '' } : row));
    },
    async *iterRows() {
      for (const row of rows) yield [row];
    },
  };
  const painter = useStoryPainter();
  await painter.setDataset(dataset);
  painter.mode.value = 'bbspineapple';
  const copyText = () =>
    collectStoryPainterForumText({
      chunks: painter.iterPreviewChunks(),
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
          mode: 'bbspineapple',
          items: painter.previewItems.value,
          itemIndexes: painter.visibleIndexes.value,
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
    expect(
      wrapper
        .findAll('pre')
        .map(row => row.text())
        .join('\n')
    ).toBe(expected);
    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(wrapper.findComponent(StoryPainterPreview).emitted('copy')).toEqual([[expected]]);
  } finally {
    wrapper.unmount();
  }
});
