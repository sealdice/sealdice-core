import { createAlova } from 'alova';
import fetchAdapter from 'alova/fetch';
import { createApis, withConfigType, mountApis } from './createApis';

export const alovaInstance = createAlova({
  baseURL: 'http://127.0.0.1:3211',
  requestAdapter: fetchAdapter(),
  beforeRequest: method => {},
  responded: res => {
    return res.json();
  }
});

export const $$userConfigMap = withConfigType({});

const api = createApis(alovaInstance, $$userConfigMap);

mountApis(api);

export default api;
export { api };
export * from './globals';
