import axios, { type AxiosRequestConfig } from "axios";
import qs from "qs";
const http = axios.create({
  baseURL: "/sd-api",
  timeout: 10000,
});



http.interceptors.request.use((config) => {
  // console.log(config.data)
  try {
    if ('ContentType' in config && config.ContentType === "application/x-www-form-urlencoded") {
      config.data =
        config.data && qs.stringify(config.data, { indices: false });
    }
    const token = localStorage.getItem('t')
    if (token) {
      config.headers.Authorization = token
      config.headers['token'] = token
    }
  } catch (e) {
    // console.log(e);
  }
  // console.log(config);
  return config;
});

http.interceptors.response.use(
  async (response) => {
    // HTTP响应状态码正常
    if (response.status === 200) {
      return Promise.resolve(response)
      // return Promise.reject(response.data)
    } else {
      console.error("服务器出错或者连接不到服务器")
      return Promise.reject(response);
    }
  },
  (error) => {

    if (error.code === 'ECONNABORTED' || error.code === 'ERR_NETWORK')
      // KMessage("连接不到服务器",'danger')
      console.error("连接不到服务器")

    return Promise.reject(error);
  }
);



// eslint-disable-next-line @typescript-eslint/no-explicit-any
export default function request<T = any>(
  method: 'post' | 'get' | 'put' | 'delete',
  url: string,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  submitData?: any,
  ContentType?: 'form' | 'json' | 'formdata', config?: AxiosRequestConfig) {
  let file: FormData, contentType: string
  switch (ContentType) {
    case "form":
      contentType = "application/x-www-form-urlencoded";
      break;
    case "formdata":
      contentType = "multipart/form-data";
      file = new FormData();
      for (const key in submitData) {
        if (!(submitData[key] instanceof Array)) {
          // console.log(submitData[key]);
          file.append(key, submitData[key]);
        } else {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          submitData[key].forEach((item: any) => {
            file.append(key, item);
          });
        }
      }
      // for(let i:number = 0;i<submitData.length;i++) {
      //   file.append('file',submitData[i]);
      // }
      submitData = file;
      break;
    default:
      contentType = "application/json";
  }
  return new Promise<T>((resolve, reject) => {
    const reqParams = Object.assign({
      method,
      url,
      [method.toLowerCase() === "get" //|| method.toLowerCase() === "delete"
        ? "params"
        : "data"]: submitData,
      contentType,
    }, config);
    http(reqParams)
      .then((res) => {
        resolve(res.data);
      })
      .catch((err) => {
        console.error(err);
        reject(err);
      });
  });
}

export function createRequest(baseUrl: string) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return function <T = any>(
    method: 'post' | 'get' | 'put' | 'delete',
    url: string,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    submitData?: any,
    ContentType?: 'form' | 'json' | 'formdata',
    config?: AxiosRequestConfig
  ) {
    return request<T>(method, baseUrl + url, submitData, ContentType, config)
  }
}
