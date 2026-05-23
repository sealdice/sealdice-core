import {
  ERUDA_STORAGE_KEY,
  createErudaController,
  isErudaEnabled,
  readStoredErudaEnabled,
  writeStoredErudaEnabled,
} from './eruda';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const createStorage = () => {
  const values = new Map<string, string>();
  return {
    getItem(key: string) {
      return values.get(key) ?? null;
    },
    setItem(key: string, value: string) {
      values.set(key, value);
    },
  };
};

const storage = createStorage();
assertEqual(readStoredErudaEnabled(storage), false);
writeStoredErudaEnabled(true, storage);
assertEqual(readStoredErudaEnabled(storage), true);
assertEqual(isErudaEnabled(storage), true);
writeStoredErudaEnabled(false, storage);
assertEqual(storage.getItem(ERUDA_STORAGE_KEY), '0');
assertEqual(readStoredErudaEnabled(storage), false);

let initCount = 0;
let destroyCount = 0;
const controller = createErudaController(async () => ({
  init() {
    initCount += 1;
  },
  destroy() {
    destroyCount += 1;
  },
}));

assertEqual(controller.isEnabled(), false);
await controller.enable();
assertEqual(controller.isEnabled(), true);
assertEqual(initCount, 1);

await controller.enable();
assertEqual(initCount, 1);

controller.disable();
assertEqual(controller.isEnabled(), false);
assertEqual(destroyCount, 1);

controller.disable();
assertEqual(destroyCount, 1);
