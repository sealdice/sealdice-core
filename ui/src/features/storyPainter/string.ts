export function replaceAllText(
  value: string,
  search: string | RegExp,
  replacement: string | ((substring: string, ...args: string[]) => string),
): string {
  if (typeof search === 'string') {
    return value.split(search).join(replacement as string);
  }
  return value.replace(search, replacement as string);
}
