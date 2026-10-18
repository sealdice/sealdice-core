import {
  ERUDA_STORAGE_KEY,
  createErudaController,
  isErudaEnabled,
  readStoredErudaEnabled,
  writeStoredErudaEnabled,
} from './eruda';
import { expect, it } from 'vitest';

it('调试面板按持久化偏好启停', async () => {
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
  expect(readStoredErudaEnabled(storage)).toBe(false);
  writeStoredErudaEnabled(true, storage);
  expect(readStoredErudaEnabled(storage)).toBe(true);
  expect(isErudaEnabled(storage)).toBe(true);
  writeStoredErudaEnabled(false, storage);
  expect(storage.getItem(ERUDA_STORAGE_KEY)).toBe('0');
  expect(readStoredErudaEnabled(storage)).toBe(false);

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

  expect(controller.isEnabled()).toBe(false);
  await controller.enable();
  expect(controller.isEnabled()).toBe(true);
  expect(initCount).toBe(1);

  await controller.enable();
  expect(initCount).toBe(1);

  controller.disable();
  expect(controller.isEnabled()).toBe(false);
  expect(destroyCount).toBe(1);

  controller.disable();
  expect(destroyCount).toBe(1);
});
