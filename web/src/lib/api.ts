import type { Stats, DownloadHistory, UserStat, SystemConfig, QueueItem } from './types';

const isDev = import.meta.env.DEV;
export const BASE_URL = isDev ? 'http://localhost:8080/dashboard/api' : '/dashboard/api';

// Retrieve token from local storage
export function getToken(): string {
  return localStorage.getItem('admin_token') || '';
}

// Save token to local storage
export function setToken(token: string) {
  localStorage.setItem('admin_token', token);
}

// Clear token on logout
export function clearToken() {
  localStorage.removeItem('admin_token');
}

// Reusable fetch wrapper that includes authentication
async function apiRequest<T>(path: string, method: string = 'GET', body?: any): Promise<T> {
  const token = getToken();
  const headers: HeadersInit = {
    'Authorization': token ? `Bearer ${token}` : '',
  };

  if (body) {
    headers['Content-Type'] = 'application/json';
  }

  const url = `${BASE_URL}${path}`;
  const response = await fetch(url, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 401) {
    clearToken();
    window.location.hash = '/login';
    throw new Error('Unauthorized access. Please login.');
  }

  if (!response.ok) {
    const errData = await response.json().catch(() => ({}));
    throw new Error(errData.error || `HTTP error! Status: ${response.status}`);
  }

  return response.json();
}

export const api = {
  getStats: () => apiRequest<Stats>('/stats'),
  getHistory: () => apiRequest<DownloadHistory[]>('/history'),
  getUsers: () => apiRequest<UserStat[]>('/users'),
  getConfig: () => apiRequest<SystemConfig>('/config'),
  getQueue: () => apiRequest<QueueItem[]>('/queue'),

  deleteRecord: async (id: number): Promise<void> => {
    await apiRequest<void>(`/delete?id=${id}`, 'POST');
  },

  broadcast: (message: string) => apiRequest<{ success: number; failed: number; total: number }>('/broadcast', 'POST', { message }),
};
