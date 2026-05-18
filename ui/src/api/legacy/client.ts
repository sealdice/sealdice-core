import { ApiError } from '@/api/error';

export type LegacyHttpMethod = 'get' | 'post' | 'put' | 'delete';
export type LegacyContentType = 'json' | 'form' | 'formdata';

export interface LegacyRequestOptions extends Omit<RequestInit, 'body' | 'method'> {
  timeout?: number;
}

const LEGACY_BASE_URL = '/sd-api';
const LEGACY_TOKEN_STORAGE_KEY = 't';

function normalizePath(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  return `${LEGACY_BASE_URL}${normalized}`;
}

function readStorageValue(key: string): string {
  try {
    return localStorage.getItem(key)?.trim() ?? '';
  } catch {
    return '';
  }
}

function writeStorageValue(key: string, value: string): void {
  try {
    if (value) {
      localStorage.setItem(key, value);
    } else {
      localStorage.removeItem(key);
    }
  } catch {
    // 忽略本地存储异常，避免阻断主流程。
  }
}

function buildQuery(data: unknown): string {
  if (!data || typeof data !== 'object') {
    return '';
  }

  const params = new URLSearchParams();
  Object.entries(data as Record<string, unknown>).forEach(([key, value]) => {
    if (value === undefined || value === null) {
      return;
    }
    if (Array.isArray(value)) {
      value.forEach(item => params.append(key, String(item)));
      return;
    }
    params.set(key, String(value));
  });
  return params.toString();
}

function buildFormData(data: unknown): FormData {
  const formData = new FormData();
  if (!data || typeof data !== 'object') {
    return formData;
  }

  Object.entries(data as Record<string, unknown>).forEach(([key, value]) => {
    if (value === undefined || value === null) {
      return;
    }
    const values = Array.isArray(value) ? value : [value];
    values.forEach(item => {
      if (item instanceof Blob) {
        formData.append(key, item);
      } else {
        formData.append(key, String(item));
      }
    });
  });
  return formData;
}

function buildRequestBody(
  method: LegacyHttpMethod,
  data: unknown,
  contentType: LegacyContentType,
): BodyInit | undefined {
  if (method === 'get' || data === undefined || data === null) {
    return undefined;
  }
  if (contentType === 'form') {
    return buildQuery(data);
  }
  if (contentType === 'formdata') {
    return buildFormData(data);
  }
  return JSON.stringify(data);
}

function buildHeaders(contentType: LegacyContentType): Headers {
  const headers = new Headers({
    Accept: 'application/json',
  });
  const token = getLegacyAccessToken();
  if (token) {
    headers.set('Authorization', token);
    headers.set('token', token);
  }
  if (contentType === 'json') {
    headers.set('Content-Type', 'application/json');
  }
  if (contentType === 'form') {
    headers.set('Content-Type', 'application/x-www-form-urlencoded');
  }
  return headers;
}

async function parseResponse(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return null;
  }
  const contentType = response.headers.get('content-type') ?? '';
  if (contentType.includes('application/json')) {
    return JSON.parse(text);
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

export function getLegacyAccessToken(): string {
  return readStorageValue(LEGACY_TOKEN_STORAGE_KEY);
}

export function setLegacyAccessToken(token: string): void {
  writeStorageValue(LEGACY_TOKEN_STORAGE_KEY, token.trim());
}

export function clearLegacyAccessToken(): void {
  setLegacyAccessToken('');
}

export async function legacyRequest<T>(
  method: LegacyHttpMethod,
  path: string,
  data?: unknown,
  contentType: LegacyContentType = 'json',
  options: LegacyRequestOptions = {},
): Promise<T> {
  const query = method === 'get' ? buildQuery(data) : '';
  const requestUrl = `${normalizePath(path)}${query ? `?${query}` : ''}`;
  const controller = new AbortController();
  const timeoutId =
    options.timeout && options.timeout > 0
      ? globalThis.setTimeout(() => controller.abort(), options.timeout)
      : undefined;

  try {
    const response = await fetch(requestUrl, {
      ...options,
      method: method.toUpperCase(),
      headers: buildHeaders(contentType),
      body: buildRequestBody(method, data, contentType),
      signal: options.signal ?? controller.signal,
    });
    const payload = await parseResponse(response);
    if (!response.ok) {
      throw new ApiError({
        status: response.status,
        statusText: response.statusText,
        data: payload,
        response,
      });
    }
    return payload as T;
  } finally {
    if (timeoutId) {
      globalThis.clearTimeout(timeoutId);
    }
  }
}
