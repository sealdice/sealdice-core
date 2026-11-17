import { describe, expect, it } from 'vitest';
import { shouldBlockUnload } from './closeGuard';

describe('upload close guard', () => {
  it('does not block when nothing is pending', () => {
    expect(
      shouldBlockUnload({
        controlledReloadArmed: false,
        hasUnsavedChanges: false,
        hasActiveUploads: false,
      })
    ).toBe(false);
  });

  it('blocks on unsaved changes', () => {
    expect(
      shouldBlockUnload({
        controlledReloadArmed: false,
        hasUnsavedChanges: true,
        hasActiveUploads: false,
      })
    ).toBe(true);
  });

  it('blocks on active uploads', () => {
    expect(
      shouldBlockUnload({
        controlledReloadArmed: false,
        hasUnsavedChanges: false,
        hasActiveUploads: true,
      })
    ).toBe(true);
  });

  it('never blocks a controlled reload', () => {
    expect(
      shouldBlockUnload({
        controlledReloadArmed: true,
        hasUnsavedChanges: true,
        hasActiveUploads: true,
      })
    ).toBe(false);
  });
});
