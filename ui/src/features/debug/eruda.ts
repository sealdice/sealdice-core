const ERUDA_STORAGE_KEY = 'sd-debug-eruda-enabled';

type ErudaStorage = Pick<Storage, 'getItem' | 'setItem'>;

type ErudaApi = {
  init(): void;
  destroy(): void;
};

type ErudaLoader = () => Promise<ErudaApi>;

function getBrowserStorage(): ErudaStorage | undefined {
  if (typeof localStorage === 'undefined') return undefined;
  return localStorage;
}

function readStoredErudaEnabled(storage: ErudaStorage | undefined = getBrowserStorage()): boolean {
  try {
    return storage?.getItem(ERUDA_STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

function writeStoredErudaEnabled(
  enabled: boolean,
  storage: ErudaStorage | undefined = getBrowserStorage(),
): void {
  try {
    storage?.setItem(ERUDA_STORAGE_KEY, enabled ? '1' : '0');
  } catch {
    // localStorage may be unavailable in embedded/private contexts.
  }
}

function createErudaLoader(): Promise<ErudaApi> {
  return import('eruda').then(module => module.default);
}

export function createErudaController(loader: ErudaLoader = createErudaLoader) {
  let eruda: ErudaApi | null = null;
  let enabled = false;
  let initPromise: Promise<void> | null = null;

  async function enable(): Promise<void> {
    if (enabled) return;
    if (initPromise) return initPromise;

    initPromise = loader()
      .then(instance => {
        eruda = instance;
        eruda.init();
        enabled = true;
      })
      .finally(() => {
        initPromise = null;
      });

    return initPromise;
  }

  function disable(): void {
    enabled = false;
    if (!eruda) return;
    eruda.destroy();
    eruda = null;
  }

  return {
    enable,
    disable,
    isEnabled() {
      return enabled;
    },
  };
}

const erudaController = createErudaController();

export function isErudaEnabled(storage?: ErudaStorage): boolean {
  return readStoredErudaEnabled(storage);
}

export async function enableEruda(storage?: ErudaStorage): Promise<void> {
  writeStoredErudaEnabled(true, storage);
  await erudaController.enable();
}

export function disableEruda(storage?: ErudaStorage): void {
  writeStoredErudaEnabled(false, storage);
  erudaController.disable();
}

export async function setErudaEnabled(enabled: boolean, storage?: ErudaStorage): Promise<void> {
  if (enabled) {
    await enableEruda(storage);
    return;
  }
  disableEruda(storage);
}

export async function syncErudaFromStorage(storage?: ErudaStorage): Promise<void> {
  if (!readStoredErudaEnabled(storage)) return;
  await erudaController.enable();
}

export {
  ERUDA_STORAGE_KEY,
  readStoredErudaEnabled,
  writeStoredErudaEnabled,
};
export type { ErudaStorage };
