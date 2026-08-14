export type StorySelectableLog = {
  pitch?: boolean;
};

export function setStoryLogsSelected<T extends StorySelectableLog>(
  logs: T[],
  selected: boolean
): void {
  for (const log of logs) {
    log.pitch = selected;
  }
}
