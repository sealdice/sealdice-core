export type RollupWarningLike = {
  code?: string;
  id?: string;
  message?: string;
};

export function shouldSuppressRollupWarning(warning: RollupWarningLike): boolean {
  const id = warning.id ?? '';
  const message = warning.message ?? '';

  if (
    warning.code === 'INVALID_ANNOTATION' &&
    id.includes('/@vueuse+core@') &&
    id.includes('/node_modules/@vueuse/core/') &&
    message.includes('#__PURE__')
  ) {
    return true;
  }

  if (
    warning.code === 'EVAL' &&
    id.includes('/eruda@') &&
    id.includes('/node_modules/eruda/eruda.js')
  ) {
    return true;
  }

  return false;
}
