import { copyText } from './copyText';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

type FakeTextarea = {
  value: string;
  remove: () => void;
  select: () => void;
  setAttribute: (name: string, value: string) => void;
  style: Record<string, string>;
};

const originalNavigator = globalThis.navigator;
const originalDocument = globalThis.document;
const appendedTextareas: FakeTextarea[] = [];

Object.defineProperty(globalThis, 'navigator', {
  configurable: true,
  value: {},
});

Object.defineProperty(globalThis, 'document', {
  configurable: true,
  value: {
    body: {
      append(element: FakeTextarea) {
        appendedTextareas.push(element);
      },
    },
    createElement(tagName: string) {
      if (tagName !== 'textarea') throw new Error(`unexpected element ${tagName}`);
      return {
        value: '',
        remove() {},
        select() {},
        setAttribute() {},
        style: {},
      } satisfies FakeTextarea;
    },
    execCommand(command: string) {
      return command === 'copy';
    },
  },
});

function setClipboard(writeText: (text: string) => Promise<void>) {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText },
  });
}

async function assertRejects(action: () => Promise<void>) {
  let rejected = false;
  try {
    await action();
  } catch {
    rejected = true;
  }
  assertEqual(rejected, true);
}

let nativeCopiedText = '';
setClipboard(async text => {
  nativeCopiedText = text;
});
await copyText('native copy');
assertEqual(nativeCopiedText, 'native copy');

setClipboard(async () => {
  throw new Error('clipboard denied');
});
await copyText('fallback copy');
assertEqual(appendedTextareas.at(-1)?.value, 'fallback copy');

Object.defineProperty(navigator, 'clipboard', {
  configurable: true,
  value: undefined,
});
document.execCommand = () => false;
await assertRejects(() => copyText('rejected copy'));

Object.defineProperty(globalThis, 'navigator', {
  configurable: true,
  value: originalNavigator,
});
Object.defineProperty(globalThis, 'document', {
  configurable: true,
  value: originalDocument,
});
