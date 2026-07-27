import { defineStore } from 'pinia';
import { computed, shallowRef } from 'vue';
import {
  isStandaloneDisplayMode,
  shouldShowPwaInstallEntry,
  type PwaBeforeInstallPromptEvent,
  type PwaInstallOutcome,
} from './pwaState';

export const usePwaInstallStore = defineStore('pwa-install', () => {
  const promptAvailable = shallowRef(false);
  const isInstalled = shallowRef(false);
  const installing = shallowRef(false);
  const initialized = shallowRef(false);
  const isSupported = shallowRef(false);

  let deferredPrompt: PwaBeforeInstallPromptEvent | null = null;
  let cleanup: (() => void) | null = null;
  let consumerCount = 0;

  function getNavigatorStandalone(): boolean {
    if (typeof navigator === 'undefined') return false;
    return 'standalone' in navigator ? Boolean((navigator as Navigator & { standalone?: boolean }).standalone) : false;
  }

  function syncStandaloneState(): void {
    if (typeof window === 'undefined') return;
    const standaloneMatches = window.matchMedia?.('(display-mode: standalone)').matches ?? false;
    isInstalled.value = isStandaloneDisplayMode(standaloneMatches, getNavigatorStandalone());
    if (isInstalled.value) {
      promptAvailable.value = false;
      deferredPrompt = null;
    }
  }

  function handleBeforeInstallPrompt(event: Event): void {
    const promptEvent = event as PwaBeforeInstallPromptEvent;
    event.preventDefault();
    deferredPrompt = promptEvent;
    promptAvailable.value = shouldShowPwaInstallEntry(true, isInstalled.value);
  }

  function handleAppInstalled(): void {
    deferredPrompt = null;
    promptAvailable.value = false;
    isInstalled.value = true;
  }

  function attachListeners(): void {
    // PWA 安装提示由浏览器事件驱动，store 统一保存 deferredPrompt，避免组件重挂载后丢状态。
    if (typeof window === 'undefined' || initialized.value) return;
    initialized.value = true;
    syncStandaloneState();
    isSupported.value = 'onbeforeinstallprompt' in window;

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt as EventListener);
    window.addEventListener('appinstalled', handleAppInstalled);

    cleanup = () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt as EventListener);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }

  function detachListeners(): void {
    if (!cleanup) {
      initialized.value = false;
      return;
    }
    cleanup();
    cleanup = null;
    initialized.value = false;
  }

  function retainListeners(): void {
    consumerCount += 1;
    attachListeners();
  }

  function releaseListeners(): void {
    consumerCount = Math.max(0, consumerCount - 1);
    if (consumerCount > 0) return;
    detachListeners();
  }

  const canInstall = computed(() => promptAvailable.value && !isInstalled.value);

  async function install(): Promise<PwaInstallOutcome> {
    if (!deferredPrompt) return 'unavailable';

    installing.value = true;
    try {
      const promptEvent = deferredPrompt;
      deferredPrompt = null;
      await promptEvent.prompt();
      const choice = await promptEvent.userChoice;
      if (choice.outcome === 'accepted') {
        isInstalled.value = true;
        promptAvailable.value = false;
        return 'installed';
      }
      promptAvailable.value = false;
      return 'dismissed';
    } finally {
      installing.value = false;
    }
  }

  return {
    canInstall,
    isInstalled,
    installing,
    isSupported,
    attachListeners,
    detachListeners,
    retainListeners,
    releaseListeners,
    install,
  };
});
