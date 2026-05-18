import { clearAccessToken, currentAccessToken, setAccessToken } from '@/features/auth/state';
import { queryClient } from '@/queryClient';
import { client } from './generated/client.gen';
import { getApiBaseUrl } from './config';
import { ApiError } from './error';
import { showApiFeedback } from './apiFeedback';
import { createHttpErrorFeedback, createNetworkErrorFeedback } from './httpStatusFeedback';

let configured = false;

export function setupApiClient(): void {
  if (configured) return;
  configured = true;

  client.setConfig({
    baseUrl: getApiBaseUrl(),
    credentials: 'include',
  });

  client.interceptors.request.use((request) => {
    const token = currentAccessToken();
    if (!token) return request;

    const headers = new Headers(request.headers);
    if (!headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${token}`);
    }

    return new Request(request, { headers });
  });

  client.interceptors.response.use((response) => {
    const newToken = response.headers.get('new-token');
    if (newToken) {
      setAccessToken(newToken);
    }

    return response;
  });

  client.interceptors.error.use((error, response, request) => {
    if (!response) {
      showApiFeedback(createNetworkErrorFeedback(error), clearApiSession);
      return error;
    }

    const apiError = new ApiError({
      status: response.status,
      statusText: response.statusText,
      data: error,
      request,
      response,
    });

    const pathname = new URL(request.url).pathname;
    if (response.status === 401 && pathname !== '/sd-api/v2/base/login') {
      clearApiSession();
    }

    showApiFeedback(createHttpErrorFeedback(apiError), clearApiSession);

    return apiError;
  });
}

function clearApiSession(): void {
  clearAccessToken();
  queryClient.clear();
}
