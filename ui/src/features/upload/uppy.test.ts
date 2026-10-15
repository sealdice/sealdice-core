import { describe, expect, it } from 'vitest';
import { getUppy, uploadEndpoint } from './uppy';

describe('upload uppy singleton', () => {
  it('returns the same instance on repeated calls', () => {
    expect(getUppy()).toBe(getUppy());
  });

  it('targets the unified tus upload endpoint', () => {
    expect(uploadEndpoint()).toMatch(/\/sd-api\/v2\/upload\/$/);
  });
});
