export const connectionsQueryKey = () => ['connect-connections'] as const;
export const connectProtocolsQueryKey = () => ['connect-protocols'] as const;
export const connectSchemasQueryKey = () => ['connect-schemas'] as const;
export const connectSignInfoQueryKey = () => ['connect-sign-info'] as const;
export const connectEndpointConfigQueryKey = (id: string) =>
  ['connect-endpoint-config', id] as const;
