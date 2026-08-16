import { describe, expect, it } from 'vitest';
import { setStoryLogsSelected } from './selection';

describe('setStoryLogsSelected', () => {
  it('selects every log when the current selection is partial', () => {
    const logs = [{ pitch: true }, { pitch: false }, { pitch: false }];

    setStoryLogsSelected(logs, true);

    expect(logs.every(log => log.pitch)).toBe(true);
  });

  it('clears every log when all logs are selected', () => {
    const logs = [{ pitch: true }, { pitch: true }];

    setStoryLogsSelected(logs, false);

    expect(logs.every(log => !log.pitch)).toBe(true);
  });
});
