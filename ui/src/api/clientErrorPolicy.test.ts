import { ApiError, createApiErrorFeedback, createNetworkErrorFeedback } from './client';

function assertEqual(actual: unknown, expected: unknown): void {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
}

function assertNotEqual(actual: unknown, expected: unknown): void {
  if (actual === expected) {
    throw new Error(`expected ${String(actual)} to differ from ${String(expected)}`);
  }
}

type TestVNode = {
  type?: unknown;
  props?: Record<string, unknown> | null;
  children?: unknown;
};

function findStyledCodeNode(node: unknown, code: string): TestVNode | undefined {
  if (!node || typeof node !== 'object') return undefined;

  const vnode = node as TestVNode;
  if (vnode.type === 'span' && vnode.children === ` ${code} `) {
    const style = vnode.props?.style;
    if (style && typeof style === 'object' && (style as Record<string, unknown>).color === 'red') {
      return vnode;
    }
  }

  if (Array.isArray(vnode.children)) {
    for (const child of vnode.children) {
      const matched = findStyledCodeNode(child, code);
      if (matched) return matched;
    }
  }

  return undefined;
}

function makeApiError(status: number, data: unknown = { detail: 'boom' }): ApiError {
  return new ApiError({
    status,
    statusText: `status ${status}`,
    data,
  });
}

function assertHasParagraphs(node: unknown, expectedCount: number): void {
  if (!node || typeof node !== 'object') {
    throw new Error('expected vnode object');
  }

  const vnode = node as TestVNode;
  const children = Array.isArray(vnode.children) ? vnode.children : [];
  const paragraphCount = children.filter(child => {
    return !!child && typeof child === 'object' && (child as TestVNode).type === 'p';
  }).length;

  assertEqual(paragraphCount, expectedCount);
}

const unauthorized = createApiErrorFeedback(makeApiError(401, { detail: 'unauthorized' }));
assertEqual(unauthorized.kind, 'dialog');
if (unauthorized.kind === 'dialog') {
  assertEqual(unauthorized.title, '身份信息');
  assertEqual(unauthorized.clearSession, true);
  assertEqual(typeof unauthorized.content, 'function');
  assertHasParagraphs(unauthorized.content(), 2);
  assertNotEqual(String(unauthorized.content()), '无效的令牌。错误码: 401 错误信息:ApiError: unauthorized');
  if (!findStyledCodeNode(unauthorized.content(), '401')) {
    throw new Error('expected 401 to be rendered as a red span vnode');
  }
}

const notFound = createApiErrorFeedback(makeApiError(404));
assertEqual(notFound.kind, 'dialog');
if (notFound.kind === 'dialog') {
  assertEqual(notFound.title, '接口报错');
  assertEqual(notFound.clearSession, undefined);
  assertEqual(typeof notFound.content, 'function');
  assertHasParagraphs(notFound.content(), 2);
}

const serverError = createApiErrorFeedback(makeApiError(500));
assertEqual(serverError.kind, 'dialog');
if (serverError.kind === 'dialog') {
  assertEqual(serverError.title, '接口报错');
  assertEqual(serverError.clearSession, true);
  assertEqual(typeof serverError.content, 'function');
  assertHasParagraphs(serverError.content(), 2);
  if (!findStyledCodeNode(serverError.content(), '500')) {
    throw new Error('expected 500 to be rendered as a red span vnode');
  }
}

const forbidden = createApiErrorFeedback(makeApiError(403, { detail: '展示模式不支持该操作' }));
assertEqual(forbidden.kind, 'business');
if (forbidden.kind === 'business') {
  assertEqual(forbidden.content, '展示模式不支持该操作');
}

const loginUnauthorized = createApiErrorFeedback(
  makeApiError(401, { detail: '密码错误' }),
  { pathname: '/sd-api/v2/base/login' },
);
assertEqual(loginUnauthorized.kind, 'business');
if (loginUnauthorized.kind === 'business') {
  assertEqual(loginUnauthorized.content, '密码错误');
}

const badRequest = createApiErrorFeedback(makeApiError(400, { detail: '参数错误' }));
assertEqual(badRequest.kind, 'business');
if (badRequest.kind === 'business') {
  assertEqual(badRequest.content, '参数错误');
}

const validationError = createApiErrorFeedback(makeApiError(422, { detail: '字段格式不正确' }));
assertEqual(validationError.kind, 'business');
if (validationError.kind === 'business') {
  assertEqual(validationError.content, '字段格式不正确');
}

const networkError = createNetworkErrorFeedback(new Error('Failed to fetch'));
assertEqual(networkError.kind, 'dialog');
if (networkError.kind === 'dialog') {
  assertEqual(networkError.title, '请求报错');
  assertEqual(typeof networkError.content, 'function');
  assertHasParagraphs(networkError.content(), 2);
}
