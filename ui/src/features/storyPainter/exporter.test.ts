// @vitest-environment happy-dom
import { expect, it, vi } from 'vitest';
import { inflateRawSync } from 'node:zlib';
import { saveAs } from 'file-saver';
import { exportStoryPainterDocx, extractMessageLines } from './exporter';
import { renderPreviewHtml } from './renderers';
import { defaultStoryPainterOptions } from './types';

vi.mock('file-saver', () => ({ saveAs: vi.fn() }));

it('DOCX 从预览 HTML 保留正文换行、中间空行和图片占位及续行排版', async () => {
  const mount = document.createElement('div');
  mount.innerHTML = renderPreviewHtml(
    {
      id: 1,
      nickname: 'Alice',
      IMUserId: '1001',
      time: 10,
      isDice: false,
      commandId: 0,
      message: '第一行\n\n第二行 & <正文>\n[CQ:image,file=https://example.com/image.png]',
    },
    [],
    defaultStoryPainterOptions(),
    '#db2777'
  );
  const messageLines = extractMessageLines(mount.querySelector('._message'));
  expect(messageLines).toEqual([
    '第一行',
    '',
    '第二行 & <正文>',
    '[图:https://example.com/image.png]',
  ]);
  await exportStoryPainterDocx([
    {
      nickname: '<Alice>:',
      messageLines,
      messageColor: '#db2777',
    },
  ]);
  const blob = vi.mocked(saveAs).mock.calls[0]![0] as Blob;
  const zip = Buffer.from(await blob.arrayBuffer());
  let xml = '';
  for (let offset = 0; offset < zip.length - 46; offset++) {
    if (zip.readUInt32LE(offset) !== 0x02014b50) continue;
    const nameSize = zip.readUInt16LE(offset + 28);
    if (zip.subarray(offset + 46, offset + 46 + nameSize).toString() !== 'word/document.xml')
      continue;
    const local = zip.readUInt32LE(offset + 42);
    const start = local + 30 + zip.readUInt16LE(local + 26) + zip.readUInt16LE(local + 28);
    xml = inflateRawSync(zip.subarray(start, start + zip.readUInt32LE(offset + 20))).toString();
    break;
  }
  const doc = new DOMParser().parseFromString(xml, 'application/xml');
  const paragraphs = [...doc.getElementsByTagName('w:p')];
  expect(
    paragraphs.map(p => [...p.getElementsByTagName('w:t')].map(t => t.textContent).join(''))
  ).toEqual(['<Alice>: 第一行', '', '第二行 & <正文>', '[图:https://example.com/image.png]']);
  expect(
    paragraphs.slice(1).map(p => p.getElementsByTagName('w:ind')[0]?.getAttribute('w:left'))
  ).toEqual(['800', '800', '800']);
  expect(
    paragraphs.map(p => p.getElementsByTagName('w:spacing')[0]?.getAttribute('w:after'))
  ).toEqual(['120', '120', '120', '120']);
});
