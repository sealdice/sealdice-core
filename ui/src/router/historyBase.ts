export function resolveHashHistoryBase(publicBase: string): string | undefined {
  const normalizedBase = publicBase.trim();
  if (normalizedBase === '' || normalizedBase === '.' || normalizedBase === './') {
    return undefined;
  }

  return publicBase;
}
