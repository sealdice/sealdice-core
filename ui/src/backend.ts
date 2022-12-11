import axios from 'axios'
import axiosRetry from 'axios-retry'
import { ofetch } from 'ofetch'


axiosRetry(axios, {
  retries: 3,
  retryDelay: (retryCount) => {
    return retryCount * 1000
  }
})

export function newRequestClient(baseURL: string) {
  const client = axios.create({
    baseURL: baseURL,
    timeout: 35000,
    withCredentials: false,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json'
    },
  });

  client.interceptors.response.use(function (response) {
    return response.data;
  }, function (error) {
    return Promise.reject(error);
  });

  return client;
}


export const urlBase = process.env.NODE_ENV == 'development' ?
  '//'+window.location.hostname+":"+3211 :
  '//'+window.location.hostname+":"+location.port


// 逐渐使用ofetch替换axios
export const apiFetch = ofetch.create({
  baseURL: urlBase,
  retry: 3,
  method: 'POST'
})

export const backend = newRequestClient(urlBase)
