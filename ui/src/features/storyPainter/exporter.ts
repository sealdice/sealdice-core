import type { FileChild, Paragraph, ParagraphChild } from 'docx';
import type { StoryPainterLogItem } from './types';

export interface StoryPainterDocxEntry {
  time?: string;
  timeColor?: string;
  nickname?: string;
  nicknameColor?: string;
  messageLines: string[];
  messageColor?: string;
}

export async function saveStoryPainterBlob(blob: Blob, filename: string): Promise<void> {
  const { saveAs } = await import('file-saver');
  saveAs(blob, filename);
}

export async function saveStoryPainterText(text: string, filename: string): Promise<void> {
  const { saveAs } = await import('file-saver');
  saveAs(new Blob([text], { type: 'text/plain;charset=utf-8' }), filename);
}

export async function exportStoryPainterRaw(items: StoryPainterLogItem[], filename = '跑团记录(未处理).txt'): Promise<void> {
  const text = items
    .map((item) => `${item.nickname}(${item.IMUserId}) ${item.time}\n${item.message}\n`)
    .join('\n');
  await saveStoryPainterText(text, filename);
}

export async function exportStoryPainterDoc(html: string, filename = '跑团记录.doc'): Promise<void> {
  const { saveAs } = await import('file-saver');
  const text = `MIME-Version: 1.0
Content-Type: multipart/related; boundary="----=_NextPart_WritingBug"

------=_NextPart_WritingBug
Content-Type: text/html; charset="utf-8"

<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
${html}
</body>
</html>
------=_NextPart_WritingBug--`;

  saveAs(new Blob([text], { type: 'application/msword' }), filename);
}

export async function exportStoryPainterDocx(entries: StoryPainterDocxEntry[], filename = '跑团记录.docx'): Promise<void> {
  const [docxApi, { saveAs }] = await Promise.all([
    import('docx'),
    import('file-saver'),
  ]);
  const { AlignmentType, Document, Packer, Paragraph, TextRun } = docxApi;
  const children: FileChild[] = entries.length > 0
    ? entries.flatMap((entry) => buildDocxParagraphs(entry, docxApi))
    : [new Paragraph({ children: [new TextRun({ text: '' })], alignment: AlignmentType.LEFT })];

  const document = new Document({
    sections: [{ properties: {}, children }],
  });
  const blob = await Packer.toBlob(document);
  saveAs(blob, filename);
}

export function extractMessageLines(el: HTMLElement | null): string[] {
  if (!el) return [''];
  const clone = el.cloneNode(true) as HTMLElement;
  const doc = el.ownerDocument || document;

  clone.querySelectorAll('img').forEach((img) => {
    const src = img.getAttribute('src') || '';
    img.replaceWith(doc.createTextNode(src ? `[图:${src}]` : '[图:无可用链接]'));
  });

  const lines = (clone.innerText || clone.textContent || '').replace(/\u00A0/g, ' ').split(/\r?\n/);
  while (lines.length > 1 && lines[lines.length - 1]?.trim() === '') {
    lines.pop();
  }
  return lines.length ? lines : [''];
}

export function readElementColor(el: HTMLElement | null): string | undefined {
  if (!el) return undefined;
  return el.style.color || window.getComputedStyle(el).color || undefined;
}

export function supportsStoryPainterDocxExport(): boolean {
  return true;
}

function buildDocxParagraphs(
  entry: StoryPainterDocxEntry,
  docxApi: typeof import('docx'),
): Paragraph[] {
  const { AlignmentType, Paragraph, TextRun } = docxApi;
  const lines = entry.messageLines.length > 0 ? [...entry.messageLines] : [''];
  const firstLine = lines.shift() ?? '';
  const timeText = (entry.time ?? '').trim();
  const nicknameText = (entry.nickname ?? '').trim();
  const timeColor = colorToDocx(entry.timeColor) ?? '666666';
  const nicknameColor = colorToDocx(entry.nicknameColor) ?? colorToDocx(entry.messageColor) ?? '333333';
  const messageColor = colorToDocx(entry.messageColor) ?? nicknameColor;

  const runs: ParagraphChild[] = [];
  if (timeText) runs.push(new TextRun({ text: timeText, color: timeColor }));
  if (timeText && (nicknameText || firstLine)) runs.push(new TextRun({ text: ' ' }));
  if (nicknameText) runs.push(new TextRun({ text: nicknameText, color: nicknameColor }));
  if (nicknameText && firstLine) runs.push(new TextRun({ text: ' ' }));
  if (firstLine) runs.push(new TextRun({ text: firstLine, color: messageColor }));
  if (runs.length === 0) runs.push(new TextRun({ text: '' }));

  const paragraphs = [
    new Paragraph({
      children: runs,
      spacing: { after: 120 },
      alignment: AlignmentType.LEFT,
    }),
  ];

  lines.forEach((line) => {
    paragraphs.push(new Paragraph({
      children: [new TextRun({ text: line, color: messageColor })],
      indent: { left: 800 },
      spacing: { after: 120 },
      alignment: AlignmentType.LEFT,
    }));
  });

  return paragraphs;
}

function colorToDocx(color?: string): string | undefined {
  if (!color) return undefined;
  let value = color.trim();
  if (!value) return undefined;
  if (value.startsWith('#')) {
    value = value.slice(1);
    if (value.length === 3) {
      value = value.split('').map((char) => char + char).join('');
    }
    return value.toUpperCase();
  }
  const rgbMatch = value.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)/i);
  if (!rgbMatch) return undefined;
  return rgbMatch.slice(1, 4).map((segment) => {
    const n = Number(segment);
    if (Number.isNaN(n) || n < 0) return '00';
    return Math.min(255, n).toString(16).padStart(2, '0');
  }).join('').toUpperCase();
}
