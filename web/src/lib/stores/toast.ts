import { writable } from 'svelte/store';

export interface ToastMessage {
  id: string;
  type: 'success' | 'error' | 'info';
  message: string;
}

export const toasts = writable<ToastMessage[]>([]);

export function showToast(type: 'success' | 'error' | 'info', message: string, duration: number = 3000) {
  const id = Math.random().toString(36).substring(2, 9);

  toasts.update((current) => [...current, { id, type, message }]);

  setTimeout(() => {
    toasts.update((current) => current.filter((t) => t.id !== id));
  }, duration);
}
