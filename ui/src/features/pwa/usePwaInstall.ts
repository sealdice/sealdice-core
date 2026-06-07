import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import {
  isStandaloneDisplayMode,
  shouldShowPwaInstallEntry,
  type PwaBeforeInstallPromptEvent,
  type PwaInstallOutcome,
} from './pwaState';

const canInstall = ref(false);
const isInstalled = ref(false);
const installing = ref(false);
const initialized = ref(false);
const isSupported = ref(false);

let deferredPrompt: PwaBeforeInstallPromptEvent | null = null;
let cleanup: (() => void) | null = null;

function getNavigatorStandalone(): boolean {
  if (typeof navigator === 'undefined') return false;
  return 'standalone' in navigator ? Boolean((navigator as Navigator & { standalone?: boolean }).standalone) : false;
}

function syncStandaloneState(): void {
  if (typeof window === 'undefined') return;
  const standaloneMatches = window.matchMedia?.('(display-mode: standalone)').matches ?? false;
  isInstalled.value = isStandaloneDisplayMode(standaloneMatches, getNavigatorStandalone());
  if (isInstalled.value) {
    canInstall.value = false;
    deferredPrompt = null;
  }
}

function handleBeforeInstallPrompt(event: Event): void {
  const promptEvent = event as PwaBeforeInstallPromptEvent;
  event.preventDefault();
  deferredPrompt = promptEvent;
  canInstall.value = shouldShowPwaInstallEntry(true, isInstalled.value);
}

function handleAppInstalled(): void {
  deferredPrompt = null;
  canInstall.value = false;
  isInstalled.value = true;
}

function attachListeners(): void {
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

export function usePwaInstall() {
  onMounted(attachListeners);
  onBeforeUnmount(() => {
    if (!cleanup) return;
    cleanup();
    cleanup = null;
    initialized.value = false;
  });

  const installAvailable = computed(() => canInstall.value && !isInstalled.value);

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
        canInstall.value = false;
        return 'installed';
      }
      canInstall.value = false;
      return 'dismissed';
    } finally {
      installing.value = false;
    }
  }

  return {
    isSupported,
    canInstall: installAvailable,
    isInstalled,
    installing,
    install,
  };
}
