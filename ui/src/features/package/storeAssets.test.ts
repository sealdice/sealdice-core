import { describe, expect, it } from 'vitest';
import type { StoreInfo, StorePackage } from '@/api';
import { getStorePackageIconUrl } from './storeAssets';

function storeAssets(icon: string): StoreInfo {
  return { banner: '', category: '', icon, readme: '', screenshots: null };
}

function storePackage(overrides: Partial<StorePackage> = {}): StorePackage {
  return {
    authors: [],
    contents: [],
    dependencies: {},
    description: '',
    download: {
      downloadCount: 0,
      hash: {},
      releaseTime: 0,
      size: 0,
      updateTime: 0,
      url: 'https://repo.sealdice.com/dice/api/store/packages/alice/app/1.2.3',
      zipUrl: '',
    },
    homepage: '',
    id: 'alice/app',
    installed: false,
    keywords: [],
    license: '',
    name: 'app',
    repository: '',
    seal: { maxVersion: '', minVersion: '' },
    storeAssets: storeAssets('assets/icon.png'),
    version: '1.2.3',
    ...overrides,
  };
}

describe('getStorePackageIconUrl', () => {
  it('loads relative icons through the backend file proxy', () => {
    expect(getStorePackageIconUrl(storePackage())).toBe(
      '/sd-api/v2/extension/store/file/alice/app/1.2.3?path=assets%2Ficon.png'
    );
  });

  it('keeps absolute icons untouched', () => {
    for (const icon of [
      'https://cdn.example.com/icon.png',
      '//cdn.example.com/icon.png',
      'data:image/png;base64,AAAA',
    ]) {
      expect(getStorePackageIconUrl(storePackage({ storeAssets: storeAssets(icon) }))).toBe(icon);
    }
  });

  it('resolves root-relative icons against the store backend origin', () => {
    expect(
      getStorePackageIconUrl(storePackage({ storeAssets: storeAssets('/assets/icon.png') }))
    ).toBe('https://repo.sealdice.com/assets/icon.png');
  });

  it('returns an empty URL when the icon or package id is missing', () => {
    expect(getStorePackageIconUrl(storePackage({ storeAssets: storeAssets('') }))).toBe('');
    expect(getStorePackageIconUrl(storePackage({ id: 'invalid' }))).toBe('');
  });
});
