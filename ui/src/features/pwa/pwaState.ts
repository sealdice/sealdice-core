export type PwaInstallOutcome = 'installed' | 'dismissed' | 'unavailable';

export interface PwaBeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void> | void;
  userChoice: Promise<{
    outcome: 'accepted' | 'dismissed';
    platform?: string;
  }>;
}

export function isStandaloneDisplayMode(
  displayModeStandalone: boolean,
  navigatorStandalone?: boolean,
): boolean {
  return displayModeStandalone || navigatorStandalone === true;
}

