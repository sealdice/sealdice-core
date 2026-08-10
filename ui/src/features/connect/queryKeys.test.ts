import { it } from 'vitest';
import {
  connectEndpointConfigQueryKey,
  connectProtocolsQueryKey,
  connectSchemasQueryKey,
  connectSignInfoQueryKey,
  connectionsQueryKey,
} from './queryKeys';

it('keeps connect query keys stable and scoped by endpoint where needed', () => {
  const keys = [
    connectionsQueryKey(),
    connectProtocolsQueryKey(),
    connectSchemasQueryKey(),
    connectSignInfoQueryKey(),
    connectEndpointConfigQueryKey('ep-1'),
  ];
  if (JSON.stringify(keys) !== JSON.stringify([
    ['connect-connections'],
    ['connect-protocols'],
    ['connect-schemas'],
    ['connect-sign-info'],
    ['connect-endpoint-config', 'ep-1'],
  ])) {
    throw new Error(`unexpected query keys = ${JSON.stringify(keys)}`);
  }
});
