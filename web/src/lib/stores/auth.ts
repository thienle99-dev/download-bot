import { writable } from 'svelte/store';
import { getToken, setToken, clearToken } from '../api';

const token = getToken();

export const isAuthenticated = writable<boolean>(!!token);

export function login(password: string): boolean {
  if (password) {
    setToken(password);
    isAuthenticated.set(true);
    return true;
  }
  return false;
}

export function logout() {
  clearToken();
  isAuthenticated.set(false);
  window.location.hash = '/login';
}
