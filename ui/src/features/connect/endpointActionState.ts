export type EndpointOperationErrors = Record<string, string>;

export function updateEndpointOperationErrors(
  current: EndpointOperationErrors,
  endpointId: string,
  message?: string
): EndpointOperationErrors {
  const { [endpointId]: _, ...remaining } = current;
  return message ? { ...remaining, [endpointId]: message } : remaining;
}
