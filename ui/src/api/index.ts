export { ApiError, setupApiClient, reportNetworkError } from './client';
export { getApiBaseUrl, joinApiBasePath } from './config';

// Generated API (hey-api / openapi-ts output)
export * from './generated';
export {
  getSdApiV2BaseSettingValue,
  putSdApiV2BaseSettingValue,
  postSdApiV2BaseSettingMailTest,
  postSdApiV2BaseSettingUpgrade,
} from './generated';
export { client } from './generated/client.gen';
