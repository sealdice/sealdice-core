export function cloneSearchFormValues<T>(values: T): T {
  return structuredClone(values);
}

export function overwriteSearchFormValues<T extends Record<string, unknown>>(
  target: T,
  source: Partial<T>,
): T {
  const next = cloneSearchFormValues(source) as Partial<T> & Record<string, unknown>;

  for (const key of Object.keys(target)) {
    if (!(key in next)) {
      delete target[key];
    }
  }

  Object.assign(target, next);
  return target;
}
