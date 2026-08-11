import { it } from 'vitest';
import { updateEndpointOperationErrors } from './endpointActionState';

it('keeps endpoint operation errors scoped to the account that produced them', () => {
  const firstFailure = updateEndpointOperationErrors({}, 'endpoint-a', '账号状态更新失败');
  const secondFailure = updateEndpointOperationErrors(firstFailure, 'endpoint-b', '账号删除失败');
  const retriedFirstEndpoint = updateEndpointOperationErrors(secondFailure, 'endpoint-a');

  if (
    JSON.stringify(secondFailure) !==
    JSON.stringify({
      'endpoint-a': '账号状态更新失败',
      'endpoint-b': '账号删除失败',
    })
  ) {
    throw new Error(`unexpected operation errors = ${JSON.stringify(secondFailure)}`);
  }
  if (JSON.stringify(retriedFirstEndpoint) !== JSON.stringify({ 'endpoint-b': '账号删除失败' })) {
    throw new Error(`unexpected retry state = ${JSON.stringify(retriedFirstEndpoint)}`);
  }
});
