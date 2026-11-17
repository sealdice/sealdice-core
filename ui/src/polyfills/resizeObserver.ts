import { ResizeObserver as ResizeObserverPolyfill } from '@juggle/resize-observer';

if (typeof globalThis.ResizeObserver !== 'function') {
  globalThis.ResizeObserver = ResizeObserverPolyfill as typeof ResizeObserver;
}
