import axios from 'axios';
import axiosRetry from 'axios-retry';

axiosRetry(axios, {
  retries: 3,
  retryDelay: retryCount => {
    return retryCount * 1000;
  },
});

export function newRequestClient(baseURL: string) {
  const client = axios.create({
    baseURL: baseURL,
    timeout: 35000,
    withCredentials: false,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
  });

  client.interceptors.response.use(
    function (response) {
      return response.data;
    },
    function (error) {
      return Promise.reject(error);
    },
  );

  return client;
}

// charyflys:为了方便调试，现将发后端的请求修改为代理转发
// 如果要设置测试转发端口，在.env 文件下设置 VITE_APP_APIURL="目标地址"即可
export const urlBase =
  import.meta.env.NODE_ENV == 'development'
    ? ''
    : '//' + window.location.hostname + ':' + location.port;

// 逐渐使用 ofetch 替换 axios
// 后记：发现 ofetch 也是一团糟，ky 也是一团糟，还是 axios 好用
// 2024.6.12 全都鲨了，只留下 axios

export const backend = newRequestClient(urlBase);
